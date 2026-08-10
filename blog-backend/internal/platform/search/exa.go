package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const exaDefaultBaseURL = "https://api.exa.ai"

// exaDefaultAdminBaseURL hosts team management, which is a different service
// from search: it rejects a search key and wants a team service key instead.
const exaDefaultAdminBaseURL = "https://admin-api.exa.ai"

// exaUsageWindow describes what the usage endpoint counts over. Exa reports the
// last 30 days by default, which is not the calendar month the gateway counts in.
const exaUsageWindow = "最近 30 天"

// Exa limits, taken from the Search API reference.
const (
	exaMaxResults      = 100
	exaDefaultType     = "auto"
	exaPaymentRequired = 402
)

// exaDateLayout is the ISO 8601 shape Exa expects for publish-date bounds.
const exaDateLayout = "2006-01-02T15:04:05.000Z"

// ExaProvider adapts the Exa search API. Exa is neural rather than keyword
// based: it has no search operators and no result-page offset, but it filters
// domains natively and can return page full text.
type ExaProvider struct {
	apiKey     string
	baseURL    string
	client     *http.Client
	searchType string

	// usageServiceKey and usageKeyID address the team-management API. They are
	// optional: without both, the adapter reports no usage and the gateway
	// falls back to its own counter.
	adminBaseURL    string
	usageServiceKey string
	usageKeyID      string

	// now is injectable because Exa expresses freshness as absolute publish
	// dates, which have to be derived from the current time.
	now func() time.Time
}

// ExaOptions carries the provider-private defaults stored in the gateway's
// provider configuration.
type ExaOptions struct {
	SearchType string `json:"search_type"`
	// UsageServiceKey is a team service key, not a search key: the usage
	// endpoint answers 401 to the latter. UsageKeyID is the UUID of the search
	// key whose spend should be read. Both are needed for usage reporting.
	UsageServiceKey string `json:"usage_service_key"`
	UsageKeyID      string `json:"usage_key_id"`
	// UsageBaseURL overrides the team-management host, for tests and proxies.
	UsageBaseURL string `json:"usage_base_url"`
}

// NewExa builds an Exa adapter. An empty base URL selects the public API.
func NewExa(apiKey, base string, options ExaOptions, client *http.Client) *ExaProvider {
	searchType := strings.TrimSpace(options.SearchType)
	switch searchType {
	case "instant", "fast", "auto", "deep-lite", "deep", "deep-reasoning":
	default:
		searchType = exaDefaultType
	}
	return &ExaProvider{
		apiKey:          strings.TrimSpace(apiKey),
		baseURL:         baseURL(base, exaDefaultBaseURL),
		client:          httpClient(client),
		searchType:      searchType,
		adminBaseURL:    baseURL(options.UsageBaseURL, exaDefaultAdminBaseURL),
		usageServiceKey: strings.TrimSpace(options.UsageServiceKey),
		usageKeyID:      strings.TrimSpace(options.UsageKeyID),
		now:             time.Now,
	}
}

func (p *ExaProvider) Name() string { return ProviderExa }

func (p *ExaProvider) Capabilities() Capability {
	return Capability{
		// Exa only synthesizes an answer in its deep-research modes, which are
		// a different latency and cost class; the gateway does not promise one.
		Answer:          false,
		RawContent:      true,
		DomainFilter:    true,
		SearchOperators: false,
		Pagination:      false,
	}
}

func (p *ExaProvider) Search(ctx context.Context, req Request) (Response, error) {
	if p.apiKey == "" {
		return Response{}, newError(ProviderExa, KindAuthFailed, 0, "未配置 Exa API Key")
	}

	body, err := json.Marshal(p.body(req))
	if err != nil {
		return Response{}, newError(ProviderExa, KindBadRequest, 0, err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return Response{}, newError(ProviderExa, KindBadRequest, 0, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, newError(ProviderExa, classifyTransport(err), 0, err.Error())
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 8<<20))
	if err != nil {
		return Response{}, newError(ProviderExa, KindUnavailable, httpResp.StatusCode, err.Error())
	}
	if httpResp.StatusCode != http.StatusOK {
		return Response{}, newError(ProviderExa, exaErrorKind(httpResp.StatusCode), httpResp.StatusCode, exaErrorMessage(responseBody))
	}

	var payload exaResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return Response{}, newError(ProviderExa, KindUnavailable, httpResp.StatusCode, "解析 Exa 响应失败: "+err.Error())
	}
	return p.normalize(req, payload), nil
}

type exaRequest struct {
	Query              string       `json:"query"`
	NumResults         int          `json:"numResults"`
	Type               string       `json:"type,omitempty"`
	Category           string       `json:"category,omitempty"`
	IncludeDomains     []string     `json:"includeDomains,omitempty"`
	ExcludeDomains     []string     `json:"excludeDomains,omitempty"`
	StartPublishedDate string       `json:"startPublishedDate,omitempty"`
	EndPublishedDate   string       `json:"endPublishedDate,omitempty"`
	Contents           *exaContents `json:"contents,omitempty"`
}

type exaContents struct {
	Highlights bool `json:"highlights,omitempty"`
	Text       bool `json:"text,omitempty"`
}

func (p *ExaProvider) body(req Request) exaRequest {
	body := exaRequest{
		Query:      strings.TrimSpace(req.Query),
		NumResults: clampResults(req.MaxResults, exaMaxResults),
		Type:       p.searchType,
		// Exa returns no snippet at all unless contents are requested, and a
		// result without a snippet is useless to an agent, so highlights are
		// always asked for.
		Contents: &exaContents{Highlights: true, Text: req.IncludeRawContent},
	}

	if req.Topic == TopicNews {
		body.Category = "news"
	}

	body.IncludeDomains = normalizeDomains(req.IncludeDomains)
	body.ExcludeDomains = normalizeDomains(req.ExcludeDomains)

	start, end := p.publishedWindow(req.Freshness)
	body.StartPublishedDate, body.EndPublishedDate = start, end
	return body
}

// publishedWindow turns the neutral freshness value into the absolute publish
// date bounds Exa expects.
func (p *ExaProvider) publishedWindow(freshness string) (string, string) {
	freshness = strings.TrimSpace(freshness)
	if freshness == "" {
		return "", ""
	}

	var span time.Duration
	switch freshness {
	case FreshnessDay:
		span = 24 * time.Hour
	case FreshnessMonth:
		span = 30 * 24 * time.Hour
	case FreshnessWeek:
		span = 7 * 24 * time.Hour
	case FreshnessYear:
		span = 365 * 24 * time.Hour
	default:
		start, end, ok := splitCustomFreshness(freshness)
		if !ok {
			return "", ""
		}
		startAt, startErr := time.Parse("2006-01-02", start)
		endAt, endErr := time.Parse("2006-01-02", end)
		if startErr != nil || endErr != nil {
			return "", ""
		}
		// The range is inclusive of the end day.
		return startAt.UTC().Format(exaDateLayout),
			endAt.UTC().Add(24*time.Hour - time.Millisecond).Format(exaDateLayout)
	}
	return p.now().UTC().Add(-span).Format(exaDateLayout), ""
}

type exaResponse struct {
	RequestID string `json:"requestId"`
	Results   []struct {
		Title         string   `json:"title"`
		URL           string   `json:"url"`
		PublishedDate string   `json:"publishedDate"`
		Author        string   `json:"author"`
		Text          string   `json:"text"`
		Summary       string   `json:"summary"`
		Highlights    []string `json:"highlights"`
	} `json:"results"`
	CostDollars struct {
		Total float64 `json:"total"`
	} `json:"costDollars"`
}

func (p *ExaProvider) normalize(req Request, payload exaResponse) Response {
	limit := clampResults(req.MaxResults, exaMaxResults)
	raw := payload.Results
	if len(raw) > limit {
		raw = raw[:limit]
	}

	results := make([]Result, 0, len(raw))
	for index, item := range raw {
		result := Result{
			Title: strings.TrimSpace(item.Title),
			URL:   strings.TrimSpace(item.URL),
			// Exa returns no relevance score on search results, so the ranking
			// itself is the only signal available.
			Score:       rankScore(index, len(raw)),
			PublishedAt: parsePublished(item.PublishedDate),
			Content:     exaSnippet(item.Highlights, item.Summary, item.Text),
		}
		if req.IncludeRawContent {
			result.RawContent = item.Text
		}
		results = append(results, result)
	}

	return Response{
		Query:        req.Query,
		Results:      results,
		Credits:      1,
		CostMicroUSD: int(payload.CostDollars.Total*1e6 + 0.5),
	}
}

// exaSnippet picks the best available summary text for a result.
func exaSnippet(highlights []string, summary, text string) string {
	if joined := strings.TrimSpace(strings.Join(highlights, " … ")); joined != "" {
		return joined
	}
	if trimmed := strings.TrimSpace(summary); trimmed != "" {
		return trimmed
	}
	text = strings.TrimSpace(text)
	if runes := []rune(text); len(runes) > 500 {
		return string(runes[:500])
	}
	return text
}

// exaErrorKind maps Exa's status codes, including the 402 it returns when an
// account is out of credit.
func exaErrorKind(status int) ErrorKind {
	if status == exaPaymentRequired {
		return KindQuotaExceeded
	}
	return classifyStatus(status)
}

func exaErrorMessage(body []byte) string {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, candidate := range []string{payload.Error, payload.Message, payload.Detail} {
			if candidate != "" {
				return candidate
			}
		}
	}
	return truncateMessage(string(body))
}

// Forward calls Exa in its native format.
func (p *ExaProvider) Forward(ctx context.Context, req PassthroughRequest) (PassthroughResponse, error) {
	if p.apiKey == "" {
		return PassthroughResponse{}, newError(ProviderExa, KindAuthFailed, 0, "未配置 Exa API Key")
	}

	response, err := forward(ctx, p.client, ProviderExa, p.baseURL+req.Path, req, func(httpReq *http.Request) {
		httpReq.Header.Set("x-api-key", p.apiKey)
	}, nil)
	if err != nil {
		return PassthroughResponse{}, err
	}

	if response.OK() {
		response.Credits = 1
		var payload exaResponse
		if json.Unmarshal(response.Body, &payload) == nil {
			response.CostMicroUSD = int(payload.CostDollars.Total*1e6 + 0.5)
		}
	} else {
		response.Kind = exaErrorKind(response.Status)
	}
	return response, nil
}

var _ Forwarder = (*ExaProvider)(nil)

// Usage reads what Exa's billing system says this key has spent.
//
// Two things make it unlike the other adapters' usage calls. It needs a team
// service key rather than the search key — the endpoint answers 401 to the
// latter — plus the UUID of the key being asked about, so it stays silent until
// an operator supplies both. And what comes back is money drawn against a
// prepaid balance, not consumption of an allowance, so the report carries no
// Limit: Exa has no ceiling of its own and the one that stops routing is the
// gateway's local monthly spend cap.
func (p *ExaProvider) Usage(ctx context.Context) (UsageReport, error) {
	if p.usageServiceKey == "" || p.usageKeyID == "" {
		// 未配置团队服务密钥就没什么可报的，交给网关的本地计数
		return UsageReport{}, ErrUsageUnavailable
	}

	endpoint := p.adminBaseURL + "/team-management/api-keys/" + url.PathEscape(p.usageKeyID) + "/usage"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return UsageReport{}, newError(ProviderExa, KindBadRequest, 0, err.Error())
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-api-key", p.usageServiceKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return UsageReport{}, newError(ProviderExa, classifyTransport(err), 0, err.Error())
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return UsageReport{}, newError(ProviderExa, KindUnavailable, httpResp.StatusCode, err.Error())
	}
	if httpResp.StatusCode != http.StatusOK {
		return UsageReport{}, newError(ProviderExa, exaErrorKind(httpResp.StatusCode),
			httpResp.StatusCode, exaErrorMessage(body))
	}

	var payload struct {
		TotalCostUSD *float64 `json:"total_cost_usd"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return UsageReport{}, newError(ProviderExa, KindUnavailable, httpResp.StatusCode,
			"解析 Exa 用量响应失败: "+err.Error())
	}
	if payload.TotalCostUSD == nil {
		// 账单系统还没有这把 key 的数据，不是错误
		return UsageReport{}, ErrUsageUnavailable
	}

	used := int(*payload.TotalCostUSD*1e6 + 0.5)
	if used < 0 {
		used = 0
	}
	return UsageReport{
		Used:   used,
		Limit:  0,
		Unit:   UsageUnitMicroUSD,
		Scope:  UsageScopeKey,
		Window: exaUsageWindow,
	}, nil
}

var _ UsageReporter = (*ExaProvider)(nil)
