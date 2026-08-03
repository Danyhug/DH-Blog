package aigateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"dh-blog/internal/platform/search"

	"github.com/gin-gonic/gin"
)

// maxPassthroughRequestBody caps the caller's body. Tavily's own limits (300
// include domains) fit comfortably; anything larger is abuse.
const maxPassthroughRequestBody = 64 << 10

// braveForwardedParams is the allowlist of Brave query parameters. An allowlist
// rather than a blind copy: it keeps a caller from smuggling in parameters the
// gateway has no accounting for, and drops any attempt to override the
// credential.
var braveForwardedParams = []string{
	"q", "count", "offset", "country", "search_lang", "ui_lang", "freshness",
	"safesearch", "spellcheck", "text_decorations", "extra_snippets",
	"result_filter", "goggles", "goggles_id", "units", "summary",
}

// TavilyPassthrough handles POST /api/gateway/v1/tavily/search.
// The request and response bodies are Tavily's own, so an existing Tavily SDK
// only needs its base URL repointed at the blog.
func (h *handler) TavilyPassthrough(c *gin.Context) {
	body, err := readPassthroughBody(c)
	if err != nil {
		writeGatewayError(c, err)
		return
	}
	h.forward(c, search.ProviderTavily, "tavily/search", search.PassthroughRequest{
		Method: http.MethodPost,
		Path:   "/search",
		Body:   sanitizeTavilyBody(body),
	})
}

// BravePassthrough handles GET /api/gateway/v1/brave/web/search.
func (h *handler) BravePassthrough(c *gin.Context) {
	h.forward(c, search.ProviderBrave, "brave/web/search", search.PassthroughRequest{
		Method: http.MethodGet,
		Path:   "/web/search",
		Query:  filterQuery(c.Request.URL.Query(), braveForwardedParams),
	})
}

// ExaPassthrough handles POST /api/gateway/v1/exa/search.
func (h *handler) ExaPassthrough(c *gin.Context) {
	body, err := readPassthroughBody(c)
	if err != nil {
		writeGatewayError(c, err)
		return
	}
	h.forward(c, search.ProviderExa, "exa/search", search.PassthroughRequest{
		Method: http.MethodPost,
		Path:   "/search",
		Body:   sanitizeJSONBody(body, "apiKey", "api_key", "x-api-key"),
	})
}

// FirecrawlPassthrough handles POST /api/gateway/v1/firecrawl/search.
//
// Only the search route is forwarded. Firecrawl's scrape and crawl endpoints
// are a different product with different pricing, and the gateway has no
// accounting for either — exposing them here would be an unmetered proxy.
func (h *handler) FirecrawlPassthrough(c *gin.Context) {
	body, err := readPassthroughBody(c)
	if err != nil {
		writeGatewayError(c, err)
		return
	}
	h.forward(c, search.ProviderFirecrawl, "firecrawl/search", search.PassthroughRequest{
		Method: http.MethodPost,
		Path:   "/search",
		Body:   sanitizeJSONBody(body, "apiKey", "api_key"),
	})
}

func (h *handler) forward(c *gin.Context, provider, endpoint string, req search.PassthroughRequest) {
	result, err := h.service.Passthrough(c.Request.Context(), apiKeyFrom(c), provider, endpoint, req, c.ClientIP())
	if err != nil {
		writeGatewayError(c, asGatewayError(err))
		return
	}
	// Both success and upstream failure bodies go back untouched: an SDK on the
	// other end expects to parse the provider's own error shape.
	c.Header("X-Gateway-Provider", result.Provider)
	if result.Cached {
		c.Header("X-Gateway-Cached", "1")
	}
	c.Data(result.Status, result.ContentType, result.Body)
}

func readPassthroughBody(c *gin.Context) ([]byte, *GatewayError) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxPassthroughRequestBody+1))
	if err != nil {
		return nil, newGatewayError(http.StatusBadRequest, "invalid_request", "读取请求体失败: "+err.Error(), "")
	}
	if len(body) > maxPassthroughRequestBody {
		return nil, newGatewayError(http.StatusRequestEntityTooLarge, "invalid_request", "请求体过大", "")
	}
	return body, nil
}

// sanitizeTavilyBody strips any credential the caller tried to pass through and
// re-serializes the object so two callers sending the same parameters in a
// different key order share one cache entry.
func sanitizeTavilyBody(body []byte) []byte {
	return sanitizeJSONBody(body, "api_key", "apiKey")
}

// sanitizeJSONBody removes the named fields and re-serializes. A body that is
// not a JSON object is forwarded unchanged and left for the upstream to reject.
func sanitizeJSONBody(body []byte, drop ...string) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	for _, field := range drop {
		delete(payload, field)
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return canonical
}

// filterQuery keeps only the allowlisted parameters, preserving repeats.
func filterQuery(values url.Values, allowed []string) url.Values {
	filtered := url.Values{}
	for _, name := range allowed {
		if items, ok := values[name]; ok {
			filtered[name] = items
		}
	}
	return filtered
}
