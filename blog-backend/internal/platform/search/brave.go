package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
