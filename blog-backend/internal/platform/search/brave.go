package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const braveDefaultBaseURL = "https://api.search.brave.com/res/v1"

// Brave limits, taken from the Web Search API reference.
const (
	braveMaxResults = 20
	braveMaxOffset  = 9
)

// BraveProvider adapts the Brave Web Search API. Brave has no native domain
// filter and no LLM answer, so domain constraints are emulated with search
// operators and answer requests are expected to be routed elsewhere.
type BraveProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client

	// mu guards the quota snapshot Brave attaches to every response. Brave
	// publishes no usage endpoint, so the headers of traffic we were going to
	// send anyway are the only way to learn where the monthly allowance stands
	// without spending more of it.
	mu    sync.Mutex
	usage *UsageReport
}

// NewBrave builds a Brave adapter. An empty base URL selects the public API.
func NewBrave(apiKey, base string, client *http.Client) *BraveProvider {
	return &BraveProvider{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: baseURL(base, braveDefaultBaseURL),
		client:  httpClient(client),
	}
}

func (p *BraveProvider) Name() string { return ProviderBrave }

func (p *BraveProvider) Capabilities() Capability {
	return Capability{
		Answer:          false,
		RawContent:      false,
		DomainFilter:    false,
		SearchOperators: true,
		Pagination:      true,
	}
}

func (p *BraveProvider) Search(ctx context.Context, req Request) (Response, error) {
	if p.apiKey == "" {
		return Response{}, newError(ProviderBrave, KindAuthFailed, 0, "未配置 Brave API Key")
	}

	endpoint := p.baseURL + "/web/search"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+p.query(req).Encode(), nil)
	if err != nil {
		return Response{}, newError(ProviderBrave, KindBadRequest, 0, err.Error())
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Subscription-Token", p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, newError(ProviderBrave, classifyTransport(err), 0, err.Error())
	}
	defer httpResp.Body.Close()
	p.observeQuota(httpResp.Header)

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 8<<20))
	if err != nil {
		return Response{}, newError(ProviderBrave, KindUnavailable, httpResp.StatusCode, err.Error())
	}
	if httpResp.StatusCode != http.StatusOK {
		return Response{}, newError(ProviderBrave, classifyStatus(httpResp.StatusCode), httpResp.StatusCode, braveErrorMessage(body))
	}

	var payload braveResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, newError(ProviderBrave, KindUnavailable, httpResp.StatusCode, "解析 Brave 响应失败: "+err.Error())
	}
	return p.normalize(req, payload), nil
}

// query builds the Brave query string, folding domain filters into search
// operators because Brave has no parameter for them.
func (p *BraveProvider) query(req Request) url.Values {
	values := url.Values{}
	values.Set("q", braveQueryString(req))
	values.Set("count", strconv.Itoa(clampResults(req.MaxResults, braveMaxResults)))
	// Decorations wrap matched terms in <strong>; the gateway returns plain text.
	values.Set("text_decorations", "0")
	values.Set("safesearch", "moderate")

	if req.Topic == TopicNews {
		values.Set("result_filter", "news")
	}
	if freshness := braveFreshness(req.Freshness); freshness != "" {
		values.Set("freshness", freshness)
	}
	if country := strings.ToUpper(strings.TrimSpace(req.Country)); len(country) == 2 {
		values.Set("country", country)
	}
	if language := strings.ToLower(strings.TrimSpace(req.Language)); language != "" {
		values.Set("search_lang", language)
	}
	if req.IncludeRawContent {
		// The closest Brave equivalent: up to 5 extra excerpts per result.
		values.Set("extra_snippets", "true")
	}
	return values
}

// braveQueryString appends site: operators for the domain filters Brave lacks.
func braveQueryString(req Request) string {
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(req.Query))

	if included := normalizeDomains(req.IncludeDomains); len(included) > 0 {
		if len(included) == 1 {
			builder.WriteString(" site:" + included[0])
		} else {
			clauses := make([]string, 0, len(included))
			for _, domain := range included {
				clauses = append(clauses, "site:"+domain)
			}
			builder.WriteString(" (" + strings.Join(clauses, " OR ") + ")")
		}
	}
	for _, domain := range normalizeDomains(req.ExcludeDomains) {
		builder.WriteString(" -site:" + domain)
	}
	return builder.String()
}

// braveFreshness maps the neutral freshness values onto Brave's codes.
func braveFreshness(freshness string) string {
	switch strings.TrimSpace(freshness) {
	case FreshnessDay:
		return "pd"
	case FreshnessWeek:
		return "pw"
	case FreshnessMonth:
		return "pm"
	case FreshnessYear:
		return "py"
	case "":
		return ""
	default:
		if start, end, ok := splitCustomFreshness(freshness); ok {
			return start + "to" + end
		}
		return ""
	}
}

type braveResponse struct {
	Query struct {
		Original             string `json:"original"`
		MoreResultsAvailable bool   `json:"more_results_available"`
	} `json:"query"`
	Web struct {
		Results []braveResult `json:"results"`
	} `json:"web"`
	News struct {
		Results []braveResult `json:"results"`
	} `json:"news"`
}

type braveResult struct {
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Description   string   `json:"description"`
	PageAge       string   `json:"page_age"`
	Age           string   `json:"age"`
	ExtraSnippets []string `json:"extra_snippets"`
}

func (p *BraveProvider) normalize(req Request, payload braveResponse) Response {
	raw := payload.Web.Results
	if len(raw) == 0 && req.Topic == TopicNews {
		raw = payload.News.Results
	}

	limit := clampResults(req.MaxResults, braveMaxResults)
	if len(raw) > limit {
		raw = raw[:limit]
	}

	results := make([]Result, 0, len(raw))
	for index, item := range raw {
		result := Result{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Content:     strings.TrimSpace(item.Description),
			Score:       rankScore(index, len(raw)),
			PublishedAt: parsePublished(item.PageAge),
		}
		if req.IncludeRawContent && len(item.ExtraSnippets) > 0 {
			result.RawContent = strings.Join(item.ExtraSnippets, "\n\n")
		}
		results = append(results, result)
	}

	query := payload.Query.Original
	if query == "" {
		query = req.Query
	}
	// Brave bills one request regardless of the result count.
	return Response{Query: query, Results: results, Credits: 1}
}

// braveQuotaWindow describes what the long rate-limit window covers. Brave
// documents it as 2,592,000 seconds — a rolling 30 days, not a calendar month,
// so the counter does not go back to zero on the 1st.
const braveQuotaWindow = "最近 30 天"

// observeQuota records the allowance Brave states on every response.
//
// The headers carry one value per window, shortest first: "1, 15000" means one
// request per second and 15,000 per month, so only the last entry is the
// monthly figure. A limit of 0 means unlimited and is deliberately not stored —
// recording it as a ceiling would read as "nothing left".
func (p *BraveProvider) observeQuota(header http.Header) {
	limit, ok := lastRateLimitValue(header.Get("X-RateLimit-Limit"))
	if !ok || limit <= 0 {
		return
	}
	remaining, ok := lastRateLimitValue(header.Get("X-RateLimit-Remaining"))
	if !ok {
		return
	}
	used := limit - remaining
	if used < 0 {
		used = 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.usage = &UsageReport{
		Used: used, Limit: limit,
		Unit: UsageUnitRequest, Scope: UsageScopeKey, Window: braveQuotaWindow,
	}
}

// Usage returns the last thing Brave said about this subscription. It issues no
// request of its own: Brave has no usage endpoint, and polling a search just to
// read the headers would spend the very quota being measured.
func (p *BraveProvider) Usage(context.Context) (UsageReport, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.usage == nil {
		return UsageReport{}, ErrUsageUnavailable
	}
	return *p.usage, nil
}

// lastRateLimitValue reads the final entry of a "1, 15000" rate-limit header,
// which is the longest window Brave reports.
func lastRateLimitValue(value string) (int, bool) {
	parts := strings.Split(value, ",")
	for index := len(parts) - 1; index >= 0; index-- {
		field := strings.TrimSpace(parts[index])
		if field == "" {
			continue
		}
		// A policy header writes "15000;w=2592000"; the count comes first.
		if semicolon := strings.IndexByte(field, ';'); semicolon >= 0 {
			field = field[:semicolon]
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

var _ UsageReporter = (*BraveProvider)(nil)

// braveErrorMessage pulls the human-readable part out of a Brave error body.
func braveErrorMessage(body []byte) string {
	var payload struct {
		Error struct {
			ID     string `json:"id"`
			Status int    `json:"status"`
			Code   string `json:"code"`
			Detail string `json:"detail"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if payload.Error.Detail != "" {
			return payload.Error.Detail
		}
		if payload.Error.Code != "" {
			return payload.Error.Code
		}
		if payload.Message != "" {
			return payload.Message
		}
	}
	return truncateMessage(string(body))
}

func truncateMessage(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 300 {
		return message[:300] + "..."
	}
	if message == "" {
		return "上游未返回错误详情"
	}
	return message
}
