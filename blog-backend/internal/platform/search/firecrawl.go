package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// firecrawlDefaultBaseURL points at the v2 API root. The version is part of the
// base URL rather than of each request path, so an operator pointing this at a
// self-hosted instance also decides which API version the gateway speaks.
const firecrawlDefaultBaseURL = "https://api.firecrawl.dev/v2"

// Firecrawl limits and prices, taken from the Search API reference and the
// billing page.
const (
	firecrawlMaxResults = 100
	firecrawlMaxQuery   = 500
	// A search costs 2 credits per 10 results, rounded up to the next block.
	// Scraped results cost an extra credit each on top of that.
	firecrawlCreditsPerBlock = 2
	firecrawlResultsPerBlock = 10
	firecrawlPaymentRequired = 402
)

// Sources Firecrawl can search. The gateway only uses the two that map onto its
// own topics; images have no place in a text search result.
const (
	firecrawlSourceWeb  = "web"
	firecrawlSourceNews = "news"
)

// firecrawlDateLayout is the American order Google's custom range operator
// expects, e.g. "12/1/2024".
const firecrawlDateLayout = "1/2/2006"

// FirecrawlProvider adapts the Firecrawl Search API. Firecrawl is a scraper
// first and a search engine second: it runs a keyword search and can hand back
// the full text of every hit in the same call, which is the one thing that sets
// it apart here. That is also what makes it expensive, so the page text is only
// fetched when the caller actually asked for raw content.
type FirecrawlProvider struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	scrapeFormat string
}

// FirecrawlOptions carries the provider-private defaults stored in the
// gateway's provider configuration.
type FirecrawlOptions struct {
	// ScrapeFormat is what a raw-content request asks for per result:
	// "markdown" for the whole page, "summary" for a condensed version. It is a
	// cost decision as much as a content one, since every scraped result adds a
	// credit to the search.
	ScrapeFormat string `json:"scrape_format"`
}

// NewFirecrawl builds a Firecrawl adapter. An empty base URL selects the public
// v2 API.
func NewFirecrawl(apiKey, base string, options FirecrawlOptions, client *http.Client) *FirecrawlProvider {
	format := strings.TrimSpace(options.ScrapeFormat)
	switch format {
	case "markdown", "summary":
	default:
		format = "markdown"
	}
	return &FirecrawlProvider{
		apiKey:       strings.TrimSpace(apiKey),
		baseURL:      baseURL(base, firecrawlDefaultBaseURL),
		client:       httpClient(client),
		scrapeFormat: format,
	}
}

func (p *FirecrawlProvider) Name() string { return ProviderFirecrawl }

func (p *FirecrawlProvider) Capabilities() Capability {
	return Capability{
		// Firecrawl returns pages, never a written answer.
		Answer:     false,
		RawContent: true,
		// includeDomains / excludeDomains are native parameters here.
		DomainFilter: true,
		// It honours the Google operator set: site:, filetype:, inurl:,
		// intitle:, quoted phrases and leading "-".
		SearchOperators: true,
		Pagination:      false,
	}
}

func (p *FirecrawlProvider) Search(ctx context.Context, req Request) (Response, error) {
	if p.apiKey == "" {
		return Response{}, newError(ProviderFirecrawl, KindAuthFailed, 0, "未配置 Firecrawl API Key")
	}

	body, err := json.Marshal(p.body(req))
	if err != nil {
		return Response{}, newError(ProviderFirecrawl, KindBadRequest, 0, err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return Response{}, newError(ProviderFirecrawl, KindBadRequest, 0, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, newError(ProviderFirecrawl, classifyTransport(err), 0, err.Error())
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 8<<20))
	if err != nil {
		return Response{}, newError(ProviderFirecrawl, KindUnavailable, httpResp.StatusCode, err.Error())
	}
	if httpResp.StatusCode != http.StatusOK {
		return Response{}, newError(ProviderFirecrawl, firecrawlErrorKind(httpResp.StatusCode),
			httpResp.StatusCode, firecrawlErrorMessage(responseBody))
	}

	var payload firecrawlResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return Response{}, newError(ProviderFirecrawl, KindUnavailable, httpResp.StatusCode, "解析 Firecrawl 响应失败: "+err.Error())
	}
	// Firecrawl reports some failures as a 200 carrying success=false. Treating
	// that as a result would hand the caller an empty list and charge them for
	// it, so it is turned back into an error here.
	if !payload.Success && payload.Error != "" {
		return Response{}, newError(ProviderFirecrawl, KindUnavailable, httpResp.StatusCode, payload.Error)
	}
	return p.normalize(req, payload), nil
}

type firecrawlRequest struct {
	Query          string           `json:"query"`
	Limit          int              `json:"limit"`
	Sources        []string         `json:"sources,omitempty"`
	IncludeDomains []string         `json:"includeDomains,omitempty"`
	ExcludeDomains []string         `json:"excludeDomains,omitempty"`
	TBS            string           `json:"tbs,omitempty"`
	Country        string           `json:"country,omitempty"`
	ScrapeOptions  *firecrawlScrape `json:"scrapeOptions,omitempty"`
}

type firecrawlScrape struct {
	Formats []firecrawlFormat `json:"formats"`
}

type firecrawlFormat struct {
	Type string `json:"type"`
}

func (p *FirecrawlProvider) body(req Request) firecrawlRequest {
	query := strings.TrimSpace(req.Query)
	if runes := []rune(query); len(runes) > firecrawlMaxQuery {
		query = string(runes[:firecrawlMaxQuery])
	}

	body := firecrawlRequest{
		Query:   query,
		Limit:   clampResults(req.MaxResults, firecrawlMaxResults),
		Sources: []string{firecrawlSourceWeb},
	}
	if req.Topic == TopicNews {
		body.Sources = []string{firecrawlSourceNews}
	}

	// Scraping is opt-in: without scrapeOptions a result is just url, title and
	// description, and each scraped page adds a credit to the bill.
	if req.IncludeRawContent {
		body.ScrapeOptions = &firecrawlScrape{Formats: []firecrawlFormat{{Type: p.scrapeFormat}}}
	}

	body.IncludeDomains = normalizeDomains(req.IncludeDomains)
	body.ExcludeDomains = normalizeDomains(req.ExcludeDomains)
	// Firecrawl documents tbs as filtering web results only; it is still sent
	// for a news search because ignoring it upstream costs nothing, while
	// withholding it would guarantee the filter never applies.
	body.TBS = firecrawlTimeRange(req.Freshness)
	if country := strings.ToUpper(strings.TrimSpace(req.Country)); len(country) == 2 {
		body.Country = country
	}
	return body
}

// firecrawlTimeRange maps the neutral freshness values onto Google's tbs
// syntax, which Firecrawl passes through verbatim.
func firecrawlTimeRange(freshness string) string {
	switch strings.TrimSpace(freshness) {
	case FreshnessDay:
		return "qdr:d"
	case FreshnessWeek:
		return "qdr:w"
	case FreshnessMonth:
		return "qdr:m"
	case FreshnessYear:
		return "qdr:y"
	case "":
		return ""
	default:
		start, end, ok := splitCustomFreshness(freshness)
		if !ok {
			return ""
		}
		startAt, startErr := time.Parse("2006-01-02", start)
		endAt, endErr := time.Parse("2006-01-02", end)
		if startErr != nil || endErr != nil {
			return ""
		}
		return "cdr:1,cd_min:" + startAt.Format(firecrawlDateLayout) +
			",cd_max:" + endAt.Format(firecrawlDateLayout)
	}
}

// firecrawlItem covers both result shapes Firecrawl returns. A web hit carries
// `description`, a news hit carries `snippet` and `date`; everything else is
// shared, so one struct reads both without a second decode pass.
type firecrawlItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Snippet     string `json:"snippet"`
	Date        string `json:"date"`
	Markdown    string `json:"markdown"`
	Summary     string `json:"summary"`
}

type firecrawlResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Web  []firecrawlItem `json:"web"`
		News []firecrawlItem `json:"news"`
	} `json:"data"`
	Warning     string `json:"warning"`
	CreditsUsed int    `json:"creditsUsed"`
	Error       string `json:"error"`
}

func (p *FirecrawlProvider) normalize(req Request, payload firecrawlResponse) Response {
	raw := payload.Data.Web
	if req.Topic == TopicNews {
		raw = payload.Data.News
	}
	// A news query that came back with nothing may still have web hits, and an
	// empty list helps nobody.
	if len(raw) == 0 {
		if len(payload.Data.Web) > 0 {
			raw = payload.Data.Web
		} else {
			raw = payload.Data.News
		}
	}

	limit := clampResults(req.MaxResults, firecrawlMaxResults)
	if len(raw) > limit {
		raw = raw[:limit]
	}

	results := make([]Result, 0, len(raw))
	for index, item := range raw {
		text := firecrawlText(item)
		result := Result{
			Title: strings.TrimSpace(item.Title),
			URL:   strings.TrimSpace(item.URL),
			// Firecrawl returns no relevance score, so the ranking it chose is
			// the only signal there is.
			Score:       rankScore(index, len(raw)),
			PublishedAt: parsePublished(item.Date),
			Content:     firecrawlSnippet(item, text),
		}
		if req.IncludeRawContent {
			result.RawContent = text
		}
		results = append(results, result)
	}

	return Response{
		Query:   req.Query,
		Results: results,
		Credits: firecrawlCredits(payload.CreditsUsed, len(results)),
	}
}

// firecrawlText returns whichever scraped body came back, since the format is
// configurable and only one of the two fields is ever filled.
func firecrawlText(item firecrawlItem) string {
	if markdown := strings.TrimSpace(item.Markdown); markdown != "" {
		return markdown
	}
	return strings.TrimSpace(item.Summary)
}

// firecrawlSnippet picks the best one-line description available, falling back
// to the head of the scraped page when the search itself returned none.
func firecrawlSnippet(item firecrawlItem, text string) string {
	if description := strings.TrimSpace(item.Description); description != "" {
		return description
	}
	if snippet := strings.TrimSpace(item.Snippet); snippet != "" {
		return snippet
	}
	if runes := []rune(text); len(runes) > 500 {
		return string(runes[:500])
	}
	return text
}

// firecrawlCredits prefers what the response says it cost. The fallback is the
// documented price — 2 credits per block of 10 results — for the case where the
// field is missing, and it deliberately never returns 0: a search that reached
// Firecrawl was paid for even when it found nothing.
func firecrawlCredits(reported, results int) int {
	if reported > 0 {
		return reported
	}
	blocks := (results + firecrawlResultsPerBlock - 1) / firecrawlResultsPerBlock
	if blocks < 1 {
		blocks = 1
	}
	return blocks * firecrawlCreditsPerBlock
}

// firecrawlUsageWindow describes what /team/credit-usage counts over. Firecrawl
// resets on the subscription's own billing cycle, not the calendar month.
const firecrawlUsageWindow = "当前账单周期"

// Usage asks Firecrawl what this team has left.
//
// The endpoint is free and takes the same bearer token as a search, so it is
// the cheapest correction available to the gateway's local counter — which only
// ever sees the traffic that went through the gateway.
func (p *FirecrawlProvider) Usage(ctx context.Context) (UsageReport, error) {
	if p.apiKey == "" {
		return UsageReport{}, newError(ProviderFirecrawl, KindAuthFailed, 0, "未配置 Firecrawl API Key")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/team/credit-usage", nil)
	if err != nil {
		return UsageReport{}, newError(ProviderFirecrawl, KindBadRequest, 0, err.Error())
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return UsageReport{}, newError(ProviderFirecrawl, classifyTransport(err), 0, err.Error())
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return UsageReport{}, newError(ProviderFirecrawl, KindUnavailable, httpResp.StatusCode, err.Error())
	}
	if httpResp.StatusCode != http.StatusOK {
		return UsageReport{}, newError(ProviderFirecrawl, firecrawlErrorKind(httpResp.StatusCode),
			httpResp.StatusCode, firecrawlErrorMessage(body))
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			RemainingCredits int `json:"remainingCredits"`
			PlanCredits      int `json:"planCredits"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return UsageReport{}, newError(ProviderFirecrawl, KindUnavailable, httpResp.StatusCode, "解析 Firecrawl 用量响应失败: "+err.Error())
	}

	// Firecrawl states an allowance and what is left of it, never what was
	// spent. Without an allowance there is nothing to measure the remainder
	// against, so the adapter says it has nothing to report rather than
	// inventing a ceiling the routing policy would then act on.
	if payload.Data.PlanCredits <= 0 {
		return UsageReport{}, ErrUsageUnavailable
	}
	used := payload.Data.PlanCredits - payload.Data.RemainingCredits
	if used < 0 {
		used = 0
	}
	return UsageReport{
		Used: used, Limit: payload.Data.PlanCredits,
		Unit: UsageUnitCredit,
		// The credits belong to the team, so a second key on the same account
		// draws from the same pool — rotating to it buys nothing.
		Scope: UsageScopeAccount, Window: firecrawlUsageWindow,
	}, nil
}

// Forward calls Firecrawl in its native format.
func (p *FirecrawlProvider) Forward(ctx context.Context, req PassthroughRequest) (PassthroughResponse, error) {
	if p.apiKey == "" {
		return PassthroughResponse{}, newError(ProviderFirecrawl, KindAuthFailed, 0, "未配置 Firecrawl API Key")
	}

	response, err := forward(ctx, p.client, ProviderFirecrawl, p.baseURL+req.Path, req, func(httpReq *http.Request) {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}, nil)
	if err != nil {
		return PassthroughResponse{}, err
	}

	if response.OK() {
		var payload firecrawlResponse
		if json.Unmarshal(response.Body, &payload) == nil {
			response.Credits = firecrawlCredits(payload.CreditsUsed, len(payload.Data.Web)+len(payload.Data.News))
		} else {
			response.Credits = firecrawlCreditsPerBlock
		}
	} else {
		response.Kind = firecrawlErrorKind(response.Status)
	}
	return response, nil
}

// firecrawlErrorKind maps Firecrawl's status codes, including the 402 it
// returns once a team has burned through its credits.
func firecrawlErrorKind(status int) ErrorKind {
	if status == firecrawlPaymentRequired {
		return KindQuotaExceeded
	}
	return classifyStatus(status)
}

// firecrawlErrorMessage unwraps the {"success":false,"error":"..."} envelope.
func firecrawlErrorMessage(body []byte) string {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Details string `json:"details"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, candidate := range []string{payload.Error, payload.Message, payload.Details} {
			if candidate != "" {
				return candidate
			}
		}
	}
	return truncateMessage(string(body))
}

var (
	_ Forwarder     = (*FirecrawlProvider)(nil)
	_ UsageReporter = (*FirecrawlProvider)(nil)
)
