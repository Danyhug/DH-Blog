// Package search adapts third-party search providers behind one request and
// response shape. It holds no business logic: quota accounting, routing and
// persistence belong to the aigateway module.
package search

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Topic values accepted by Request.Topic.
const (
	TopicGeneral = "general"
	TopicNews    = "news"
)

// Freshness values accepted by Request.Freshness. A custom range is also
// allowed, written as "YYYY-MM-DDtoYYYY-MM-DD".
const (
	FreshnessDay   = "day"
	FreshnessWeek  = "week"
	FreshnessMonth = "month"
	FreshnessYear  = "year"
)

// Provider names registered by the gateway.
const (
	ProviderBrave     = "brave"
	ProviderTavily    = "tavily"
	ProviderExa       = "exa"
	ProviderFirecrawl = "firecrawl"
)

// DefaultTimeout bounds a single upstream call.
const DefaultTimeout = 15 * time.Second

// Request is the provider-neutral search request. Fields a provider cannot
// express are either emulated (see Brave's site: injection) or dropped; the
// routing policy is responsible for not sending a request to a provider that
// would silently lose something the caller asked for.
type Request struct {
	Query             string
	MaxResults        int
	Topic             string
	Freshness         string
	Country           string // ISO 3166-1 alpha-2, upper or lower case
	Language          string // ISO 639-1
	IncludeDomains    []string
	ExcludeDomains    []string
	IncludeAnswer     bool
	IncludeRawContent bool
}

// Result is one normalized search hit.
type Result struct {
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Content     string     `json:"content"`
	Score       float64    `json:"score"`
	PublishedAt *time.Time `json:"published_at"`
	RawContent  string     `json:"raw_content,omitempty"`
}

// Response is the normalized provider answer.
type Response struct {
	Query   string
	Answer  string
	Results []Result
	// Credits is how many billable units the provider charged, in that
	// provider's own accounting: search credits for Tavily, one per request for
	// Brave and Exa.
	Credits int
	// CostMicroUSD is the request's price in millionths of a US dollar, for
	// providers that bill by amount rather than by unit. Exa reports it; the
	// others leave it zero.
	CostMicroUSD int
}

// Capability describes what a provider can express natively. The routing
// policy uses it to keep requests away from providers that would drop a
// caller's requirement.
type Capability struct {
	Answer          bool // returns an LLM-written answer
	RawContent      bool // returns page full text
	DomainFilter    bool // native include/exclude domain lists
	SearchOperators bool // honours site:, filetype:, quoted phrases
	Pagination      bool // supports paging past the first result page
}

// Provider is one upstream search API.
type Provider interface {
	Name() string
	Capabilities() Capability
	Search(ctx context.Context, req Request) (Response, error)
}

// ErrorKind classifies an upstream failure so the gateway can decide between
// failing the request and falling back to another provider.
type ErrorKind string

const (
	KindAuthFailed    ErrorKind = "provider_auth_failed"
	KindBadRequest    ErrorKind = "provider_bad_request"
	KindRateLimited   ErrorKind = "provider_rate_limited"
	KindQuotaExceeded ErrorKind = "provider_quota_exceeded"
	KindUnavailable   ErrorKind = "provider_unavailable"
	KindTimeout       ErrorKind = "provider_timeout"
)

// Error is a classified upstream failure.
type Error struct {
	Provider string
	Kind     ErrorKind
	Status   int
	Message  string
}

func (e *Error) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("%s: %s (HTTP %d): %s", e.Provider, e.Kind, e.Status, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.Provider, e.Kind, e.Message)
}

// Retryable reports whether trying a different provider is worthwhile. A bad
// request or a broken credential will fail the same way everywhere the caller
// is at fault, but a credential problem is specific to one provider, so it is
// still worth falling back.
func (e *Error) Retryable() bool {
	switch e.Kind {
	case KindRateLimited, KindQuotaExceeded, KindUnavailable, KindTimeout, KindAuthFailed:
		return true
	default:
		return false
	}
}

func newError(provider string, kind ErrorKind, status int, message string) *Error {
	return &Error{Provider: provider, Kind: kind, Status: status, Message: message}
}

// classifyStatus maps an upstream HTTP status onto an ErrorKind. Tavily's
// 432/433 quota codes are handled by its own adapter before this is reached.
func classifyStatus(status int) ErrorKind {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return KindAuthFailed
	case status == http.StatusTooManyRequests:
		return KindRateLimited
	case status >= 500:
		return KindUnavailable
	case status >= 400:
		return KindBadRequest
	default:
		return KindUnavailable
	}
}

// classifyTransport maps a transport-level failure onto an ErrorKind.
func classifyTransport(err error) ErrorKind {
	if err == nil {
		return KindUnavailable
	}
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return KindTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return KindTimeout
	}
	return KindUnavailable
}

var customFreshnessPattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})to(\d{4}-\d{2}-\d{2})$`)

// splitCustomFreshness returns the two ends of a "YYYY-MM-DDtoYYYY-MM-DD"
// range, or ok=false when the value is not a custom range.
func splitCustomFreshness(freshness string) (start, end string, ok bool) {
	match := customFreshnessPattern.FindStringSubmatch(freshness)
	if match == nil {
		return "", "", false
	}
	return match[1], match[2], true
}

// normalizeDomains trims and lowercases a domain list, dropping empties and
// stripping a leading scheme or "www." so both providers see the same shape.
func normalizeDomains(domains []string) []string {
	normalized := make([]string, 0, len(domains))
	seen := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		value := strings.ToLower(strings.TrimSpace(domain))
		value = strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
		value = strings.TrimPrefix(value, "www.")
		if index := strings.IndexByte(value, '/'); index >= 0 {
			value = value[:index]
		}
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

// clampResults keeps MaxResults inside the range both providers accept.
func clampResults(maxResults, upper int) int {
	if maxResults <= 0 {
		return 5
	}
	if maxResults > upper {
		return upper
	}
	return maxResults
}

// rankScore synthesizes a descending relevance score for providers that do not
// return one, so callers can sort mixed results consistently.
func rankScore(index, total int) float64 {
	if total <= 0 {
		return 0
	}
	return 1 - float64(index)/float64(total)
}

var publishedLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// parsePublished reads the several date shapes the providers emit.
func parsePublished(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range publishedLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	return nil
}

// httpClient returns the caller's client or a timeout-bounded default.
func httpClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: DefaultTimeout}
}

// baseURL returns the configured override or the built-in default.
func baseURL(configured, fallback string) string {
	configured = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(configured), "/"))
	if configured == "" {
		return fallback
	}
	return configured
}
