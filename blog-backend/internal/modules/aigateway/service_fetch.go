package aigateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"dh-blog/internal/platform/search"

	"github.com/sirupsen/logrus"
)

// Page fetches read one URL through a provider that can render it (today only
// Firecrawl's scrape endpoint) and hand back its main content as Markdown.
//
// They reuse the search path's machinery wholesale — key rate limit and quota,
// provider whitelist, health, pacing, credential rotation, accounting, request
// log, result cache — because a fetch spends the same upstream credits as a
// search and has to be governed by the same budget.

const (
	// maxCachedFetchBytes keeps an outsized page out of the in-memory cache.
	// A page that large is rare, and paging through it re-fetches instead.
	maxCachedFetchBytes = 2 << 20
	// fetchTimeoutMargin is how much sooner than the gateway's own client the
	// provider is told to give up, so a slow page is abandoned upstream — and
	// not billed — rather than finished for a caller who has already gone.
	fetchTimeoutMargin = 2 * time.Second
)

// ErrNoFetchProvider means no provider that can read pages is usable.
var ErrNoFetchProvider = errors.New("没有可用的网页抓取供应商（需启用 Firecrawl 并配置密钥）")

// FetchRequest is a validated page fetch.
type FetchRequest struct {
	URL string
	// MaxLength caps the characters returned; 0 returns the whole page.
	MaxLength int
	// StartIndex skips that many characters, so a long page can be read in
	// pieces. The page is cached whole, so later pieces cost nothing.
	StartIndex int
	NoCache    bool
}

// FetchMeta describes how a page was produced.
type FetchMeta struct {
	RequestID    string `json:"request_id"`
	Cached       bool   `json:"cached"`
	LatencyMS    int    `json:"latency_ms"`
	Credits      int    `json:"credits"`
	FallbackFrom string `json:"fallback_from,omitempty"`
}

// FetchResult is one page, or one piece of it, as the gateway returns it.
type FetchResult struct {
	// URL is where the page ended up after redirects.
	URL         string `json:"url"`
	Provider    string `json:"provider"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	// StatusCode is the target site's own answer. A 404 still comes back as a
	// page: the provider fetched it and billed for it.
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type,omitempty"`
	Content     string `json:"content"`
	StartIndex  int    `json:"start_index"`
	// TotalLength is the whole page's length in characters.
	TotalLength int `json:"total_length"`
	// NextStartIndex is where the next piece begins; 0 when this piece ends
	// the page.
	NextStartIndex int       `json:"next_start_index,omitempty"`
	Meta           FetchMeta `json:"meta"`
}

// cachedFetch is the whole page as stored in the result cache.
type cachedFetch struct {
	Provider string
	Page     search.FetchResponse
}

// Fetch reads one page on behalf of an authenticated key. endpoint labels the
// request log, which is the only place the HTTP and MCP entries differ.
func (s *Service) Fetch(ctx context.Context, key *APIKey, req FetchRequest, clientIP, endpoint string) (FetchResult, error) {
	started := s.now()
	entry := RequestLog{
		CreatedAt: started,
		Endpoint:  endpoint,
		Query:     truncateQuery(req.URL),
		ClientIP:  clientIP,
	}
	if key != nil {
		entry.APIKeyID = key.ID
	}

	result, err := s.fetch(ctx, key, req, &entry, started)
	entry.LatencyMS = int(s.now().Sub(started) / time.Millisecond)

	if err != nil {
		gatewayErr := asGatewayError(err)
		entry.Status = gatewayErr.LogStatus()
		entry.HTTPStatus = gatewayErr.Status
		entry.Error = gatewayErr.Message
		if entry.Provider == "" {
			entry.Provider = gatewayErr.Provider
		}
		s.enqueueLog(entry)
		return FetchResult{}, gatewayErr
	}

	result.Meta.RequestID = newRequestID()
	result.Meta.LatencyMS = entry.LatencyMS
	entry.Status = StatusOK
	entry.HTTPStatus = http.StatusOK
	entry.Provider = result.Provider
	entry.ResultCount = 1
	entry.Cached = result.Meta.Cached
	entry.Credits = result.Meta.Credits
	entry.FallbackFrom = result.Meta.FallbackFrom
	s.enqueueLog(entry)
	return result, nil
}

func (s *Service) fetch(ctx context.Context, key *APIKey, req FetchRequest, entry *RequestLog, now time.Time) (FetchResult, error) {
	if key != nil {
		if !s.rates.allow(key.ID, key.RateLimitPerMin, now) {
			return FetchResult{}, newGatewayError(http.StatusTooManyRequests, "rate_limit_exceeded", ErrRateLimited.Error(), "")
		}
		if err := s.checkKeyQuota(ctx, key, now); err != nil {
			return FetchResult{}, err
		}
	}

	cacheKey := fetchCacheKey(req.URL)
	if !req.NoCache && s.options.CacheTTL > 0 {
		if hit, ok := s.cache.Get(cacheKey); ok {
			if payload, valid := hit.(cachedFetch); valid {
				result := pageResult(payload.Page, payload.Provider, req)
				result.Meta.Cached = true
				return result, nil
			}
		}
	}

	pool, err := s.fetchers(ctx, key, now)
	if err != nil {
		return FetchResult{}, err
	}
	upstream := search.FetchRequest{URL: req.URL, Timeout: s.fetchTimeout()}

	var lastErr error
	attempts := 0
	for _, name := range pool {
		if attempts >= maxAttempts {
			break
		}
		runtime := s.runtime(name)
		if runtime == nil || !runtime.breaker.Allow() {
			continue
		}

		attempts++
		entry.Provider = name
		var page search.FetchResponse
		providerKeyID, reached, callErr := s.callWithRotation(ctx, runtime, now, func(provider search.Provider) error {
			fetcher, ok := provider.(search.Fetcher)
			if !ok {
				return ErrNoFetchProvider
			}
			var fetchErr error
			page, fetchErr = fetcher.Fetch(ctx, upstream)
			return fetchErr
		})
		if callErr == nil {
			runtime.breaker.Report(true)
			s.accountUsage(ctx, key, name, providerKeyID, page.Credits, 0, now)
			if s.options.CacheTTL > 0 && len(page.Content) <= maxCachedFetchBytes {
				_ = s.cache.Set(cacheKey, cachedFetch{Provider: name, Page: page}, s.options.CacheTTL)
			}
			result := pageResult(page, name, req)
			result.Meta.Credits = page.Credits
			if name != pool[0] {
				result.Meta.FallbackFrom = pool[0]
			}
			return result, nil
		}
		if !reached {
			runtime.breaker.Release()
			if errors.Is(callErr, search.ErrLimiterBusy) || errors.Is(callErr, ErrNoProviderAvailable) {
				lastErr = callErr
				continue
			}
			return FetchResult{}, newGatewayError(http.StatusGatewayTimeout, "provider_timeout", callErr.Error(), name)
		}

		lastErr = callErr
		var providerErr *search.Error
		if errors.As(callErr, &providerErr) && !providerErr.Retryable() {
			// The upstream answered, and the answer was about the page or the
			// request — no evidence that the provider itself is unwell.
			runtime.breaker.Report(true)
			return FetchResult{}, gatewayErrorFromProvider(providerErr)
		}
		runtime.breaker.Report(false)
		s.noteProviderFailure(ctx, runtime, callErr, now)
		logrus.Warnf("网页抓取供应商 %s 调用失败，尝试回退: %v", name, callErr)
	}

	if lastErr == nil {
		return FetchResult{}, newGatewayError(http.StatusServiceUnavailable, "no_provider_available", ErrNoFetchProvider.Error(), "")
	}
	return FetchResult{}, s.exhausted(lastErr)
}

// fetchers is every page-reading provider this key may use right now, best
// first. The same filters as auto search apply — whitelist, health, quota — and
// the survivors are ranked by the balanced score, since no strategy has an
// opinion about which renderer reads a page better.
func (s *Service) fetchers(ctx context.Context, key *APIKey, now time.Time) ([]string, error) {
	candidates, err := s.candidates(ctx, now)
	if err != nil {
		var gatewayErr *GatewayError
		if errors.As(err, &gatewayErr) && gatewayErr.Type == "no_provider_available" {
			return nil, newGatewayError(http.StatusServiceUnavailable, "no_provider_available", ErrNoFetchProvider.Error(), "")
		}
		return nil, err
	}
	input := routeInput{Candidates: candidates}
	if key != nil {
		input.Allowed = key.AllowedList()
	}
	pool := keep(eligible(input, ""), func(item candidate) bool {
		runtime := s.runtime(item.Name)
		return runtime != nil && runtime.fetches
	})
	if len(pool) == 0 {
		return nil, newGatewayError(http.StatusServiceUnavailable, "no_provider_available", ErrNoFetchProvider.Error(), "")
	}
	sortBalanced(pool, false)
	names := make([]string, 0, len(pool))
	for _, item := range pool {
		names = append(names, item.Name)
	}
	return names, nil
}

// fetchTimeout is what the provider is told it may spend on a page.
func (s *Service) fetchTimeout() time.Duration {
	if s.options.UpstreamTimeout <= fetchTimeoutMargin {
		return 0
	}
	return s.options.UpstreamTimeout - fetchTimeoutMargin
}

// pageResult cuts the requested piece out of a whole page. Lengths count
// characters, not bytes, so a piece never ends inside a multi-byte rune.
func pageResult(page search.FetchResponse, provider string, req FetchRequest) FetchResult {
	runes := []rune(page.Content)
	start := min(req.StartIndex, len(runes))
	end := len(runes)
	if req.MaxLength > 0 && start+req.MaxLength < end {
		end = start + req.MaxLength
	}
	result := FetchResult{
		URL:         page.URL,
		Provider:    provider,
		Title:       page.Title,
		Description: page.Description,
		StatusCode:  page.StatusCode,
		ContentType: page.ContentType,
		Content:     string(runes[start:end]),
		StartIndex:  start,
		TotalLength: len(runes),
	}
	if end < len(runes) {
		result.NextStartIndex = end
	}
	return result
}

// fetchCacheKey keys the cache on the URL alone: the piece requested is cut
// out after the cache, so every piece of a page shares one entry.
func fetchCacheKey(url string) string {
	digest := sha256.Sum256([]byte(url))
	return "gw:fetch:" + hex.EncodeToString(digest[:])
}
