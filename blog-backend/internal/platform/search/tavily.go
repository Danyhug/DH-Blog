package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const tavilyDefaultBaseURL = "https://api.tavily.com"

// Tavily limits, taken from the Search API reference.
const (
	tavilyMaxResults     = 20
	tavilyMaxInclude     = 300
	tavilyMaxExclude     = 150
	tavilyDefaultDepth   = "basic"
	tavilyDefaultChunks  = 3
	tavilyQuotaStatusLow = 432
	tavilyQuotaStatusPay = 433
)

// TavilyProvider adapts the Tavily Search API, the richer of the two upstreams:
// it is the only one that can return an LLM answer or page full text.
type TavilyProvider struct {
	apiKey          string
	baseURL         string
	client          *http.Client
	searchDepth     string
	chunksPerSource int
}

// TavilyOptions carries the provider-private defaults stored in the gateway's
// provider configuration.
type TavilyOptions struct {
	SearchDepth     string `json:"search_depth"`
	ChunksPerSource int    `json:"chunks_per_source"`
}

// NewTavily builds a Tavily adapter. An empty base URL selects the public API.
func NewTavily(apiKey, base string, options TavilyOptions, client *http.Client) *TavilyProvider {
	depth := strings.TrimSpace(options.SearchDepth)
	switch depth {
	case "advanced", "basic", "fast", "ultra-fast":
	default:
		depth = tavilyDefaultDepth
	}
	chunks := options.ChunksPerSource
	if chunks < 1 || chunks > 3 {
		chunks = tavilyDefaultChunks
	}
	return &TavilyProvider{
		apiKey:          strings.TrimSpace(apiKey),
		baseURL:         baseURL(base, tavilyDefaultBaseURL),
		client:          httpClient(client),
		searchDepth:     depth,
		chunksPerSource: chunks,
	}
}

func (p *TavilyProvider) Name() string { return ProviderTavily }

func (p *TavilyProvider) Capabilities() Capability {
	return Capability{
		Answer:          true,
		RawContent:      true,
		DomainFilter:    true,
		SearchOperators: false,
		Pagination:      false,
	}
}

func (p *TavilyProvider) Search(ctx context.Context, req Request) (Response, error) {
	if p.apiKey == "" {
		return Response{}, newError(ProviderTavily, KindAuthFailed, 0, "未配置 Tavily API Key")
	}

	body, err := json.Marshal(p.body(req))
	if err != nil {
		return Response{}, newError(ProviderTavily, KindBadRequest, 0, err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return Response{}, newError(ProviderTavily, KindBadRequest, 0, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, newError(ProviderTavily, classifyTransport(err), 0, err.Error())
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 8<<20))
	if err != nil {
		return Response{}, newError(ProviderTavily, KindUnavailable, httpResp.StatusCode, err.Error())
	}
	if httpResp.StatusCode != http.StatusOK {
		return Response{}, newError(ProviderTavily, tavilyErrorKind(httpResp.StatusCode), httpResp.StatusCode, tavilyErrorMessage(responseBody))
	}

	var payload tavilyResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return Response{}, newError(ProviderTavily, KindUnavailable, httpResp.StatusCode, "解析 Tavily 响应失败: "+err.Error())
	}
	return p.normalize(req, payload), nil
}

type tavilyRequest struct {
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth"`
	ChunksPerSource   int      `json:"chunks_per_source,omitempty"`
	MaxResults        int      `json:"max_results"`
	Topic             string   `json:"topic,omitempty"`
	TimeRange         string   `json:"time_range,omitempty"`
	StartDate         string   `json:"start_date,omitempty"`
	EndDate           string   `json:"end_date,omitempty"`
	Country           string   `json:"country,omitempty"`
	IncludeAnswer     bool     `json:"include_answer"`
	IncludeRawContent bool     `json:"include_raw_content"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`
	IncludeUsage      bool     `json:"include_usage"`
}

func (p *TavilyProvider) body(req Request) tavilyRequest {
	body := tavilyRequest{
		Query:             strings.TrimSpace(req.Query),
		SearchDepth:       p.searchDepth,
		MaxResults:        clampResults(req.MaxResults, tavilyMaxResults),
		IncludeAnswer:     req.IncludeAnswer,
		IncludeRawContent: req.IncludeRawContent,
		IncludeUsage:      true,
	}
	// chunks_per_source is rejected by the ultra-fast depth, which returns a
	// single summary per URL instead of chunks.
	if p.searchDepth != "ultra-fast" {
		body.ChunksPerSource = p.chunksPerSource
	}

	if req.Topic == TopicNews {
		body.Topic = TopicNews
	} else {
		body.Topic = TopicGeneral
	}

	switch freshness := strings.TrimSpace(req.Freshness); freshness {
	case FreshnessDay, FreshnessWeek, FreshnessMonth, FreshnessYear:
		body.TimeRange = freshness
	case "":
	default:
		if start, end, ok := splitCustomFreshness(freshness); ok {
			body.StartDate, body.EndDate = start, end
		}
	}

	// Tavily names countries in English; the gateway speaks ISO codes. An
	// unmapped code is dropped rather than rejected.
	if country, ok := tavilyCountry(req.Country); ok && body.Topic == TopicGeneral {
		body.Country = country
	}

	body.IncludeDomains = truncateDomains(normalizeDomains(req.IncludeDomains), tavilyMaxInclude)
	body.ExcludeDomains = truncateDomains(normalizeDomains(req.ExcludeDomains), tavilyMaxExclude)
	return body
}

func truncateDomains(domains []string, limit int) []string {
	if len(domains) > limit {
		return domains[:limit]
	}
	return domains
}

type tavilyResponse struct {
	Query     string `json:"query"`
	Answer    string `json:"answer"`
	RequestID string `json:"request_id"`
	Results   []struct {
		Title         string  `json:"title"`
		URL           string  `json:"url"`
		Content       string  `json:"content"`
		Score         float64 `json:"score"`
		RawContent    string  `json:"raw_content"`
		PublishedDate string  `json:"published_date"`
	} `json:"results"`
	Usage struct {
		Credits int `json:"credits"`
	} `json:"usage"`
}

func (p *TavilyProvider) normalize(req Request, payload tavilyResponse) Response {
	limit := clampResults(req.MaxResults, tavilyMaxResults)
	raw := payload.Results
	if len(raw) > limit {
		raw = raw[:limit]
	}

	results := make([]Result, 0, len(raw))
	for _, item := range raw {
		results = append(results, Result{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Content:     strings.TrimSpace(item.Content),
			Score:       item.Score,
			PublishedAt: parsePublished(item.PublishedDate),
			RawContent:  item.RawContent,
		})
	}

	query := payload.Query
	if query == "" {
		query = req.Query
	}
	credits := payload.Usage.Credits
	if credits <= 0 {
		// Fall back to the documented price of the configured depth.
		credits = 1
		if p.searchDepth == "advanced" {
			credits = 2
		}
	}
	return Response{Query: query, Answer: strings.TrimSpace(payload.Answer), Results: results, Credits: credits}
}

// tavilyErrorKind maps Tavily's status codes, including the two custom quota
// codes that are not part of the standard HTTP range.
func tavilyErrorKind(status int) ErrorKind {
	switch status {
	case tavilyQuotaStatusLow, tavilyQuotaStatusPay:
		return KindQuotaExceeded
	default:
		return classifyStatus(status)
	}
}

// tavilyErrorMessage unwraps Tavily's {"detail":{"error":"..."}} envelope.
func tavilyErrorMessage(body []byte) string {
	var payload struct {
		Detail struct {
			Error string `json:"error"`
		} `json:"detail"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if payload.Detail.Error != "" {
			return payload.Detail.Error
		}
		if payload.Error != "" {
			return payload.Error
		}
	}
	return truncateMessage(string(body))
}
