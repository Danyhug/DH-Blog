package aigateway

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"dh-blog/internal/platform/search"

	"github.com/gin-gonic/gin"
)

// Request body limits. They are the gateway's own, deliberately no looser than
// the tightest upstream so a request never silently loses a constraint.
const (
	maxQueryLength  = 400
	maxDomainFilter = 50
)

const apiKeyContextKey = "gatewayAPIKey"

type handler struct{ service *Service }

func newHandler(service *Service) *handler { return &handler{service: service} }

// searchBody is the wire shape of a gateway search request. Pointer fields
// distinguish "not supplied" from an explicit false.
type searchBody struct {
	Query             string   `json:"query"`
	Provider          string   `json:"provider"`
	MaxResults        int      `json:"max_results"`
	Topic             string   `json:"topic"`
	Freshness         string   `json:"freshness"`
	Country           string   `json:"country"`
	Language          string   `json:"language"`
	IncludeDomains    []string `json:"include_domains"`
	ExcludeDomains    []string `json:"exclude_domains"`
	IncludeAnswer     bool     `json:"include_answer"`
	IncludeRawContent bool     `json:"include_raw_content"`
	AllowFallback     *bool    `json:"allow_fallback"`
	NoCache           bool     `json:"no_cache"`
}

// errorBody is the gateway's error envelope.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Provider  string `json:"provider,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func writeGatewayError(c *gin.Context, err *GatewayError) {
	c.AbortWithStatusJSON(err.Status, errorBody{Error: errorDetail{
		Type:     err.Type,
		Message:  err.Message,
		Provider: err.Provider,
	}})
}

// authenticate resolves the gateway credential before any handler runs.
func (h *handler) authenticate(c *gin.Context) {
	plain := extractAPIKey(c.GetHeader("Authorization"), c.GetHeader("X-API-Key"))
	key, err := h.service.authenticate(c.Request.Context(), plain)
	if err != nil {
		writeGatewayError(c, authGatewayError(err))
		return
	}
	c.Set(apiKeyContextKey, key)
	c.Next()
}

func apiKeyFrom(c *gin.Context) *APIKey {
	if value, ok := c.Get(apiKeyContextKey); ok {
		if key, valid := value.(*APIKey); valid {
			return key
		}
	}
	return nil
}

// Search handles POST /api/gateway/v1/search.
func (h *handler) Search(c *gin.Context) {
	var body searchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeGatewayError(c, newGatewayError(http.StatusBadRequest, "invalid_request", "请求体解析失败: "+err.Error(), ""))
		return
	}
	h.runSearch(c, body)
}

// SearchGET handles GET /api/gateway/v1/search, which exists so a plain curl
// or a minimal agent can call the gateway without composing JSON.
func (h *handler) SearchGET(c *gin.Context) {
	body := searchBody{
		Query:             firstNonEmpty(c.Query("query"), c.Query("q")),
		Provider:          c.Query("provider"),
		Topic:             c.Query("topic"),
		Freshness:         c.Query("freshness"),
		Country:           c.Query("country"),
		Language:          c.Query("language"),
		IncludeDomains:    splitList(c.Query("include_domains")),
		ExcludeDomains:    splitList(c.Query("exclude_domains")),
		IncludeAnswer:     queryBool(c.Query("include_answer")),
		IncludeRawContent: queryBool(c.Query("include_raw_content")),
		NoCache:           queryBool(c.Query("no_cache")),
	}
	if raw := c.Query("max_results"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeGatewayError(c, newGatewayError(http.StatusBadRequest, "invalid_request", "max_results 必须是整数", ""))
			return
		}
		body.MaxResults = parsed
	}
	if raw := c.Query("allow_fallback"); raw != "" {
		value := queryBool(raw)
		body.AllowFallback = &value
	}
	h.runSearch(c, body)
}

func (h *handler) runSearch(c *gin.Context, body searchBody) {
	req, err := normalizeSearch(body)
	if err != nil {
		writeGatewayError(c, err)
		return
	}
	result, searchErr := h.service.Search(c.Request.Context(), apiKeyFrom(c), req, c.ClientIP())
	if searchErr != nil {
		writeGatewayError(c, asGatewayError(searchErr))
		return
	}
	c.JSON(http.StatusOK, result)
}

// normalizeSearch validates and defaults the request. Rejecting here keeps
// invalid input from ever reaching an upstream and burning a credit.
func normalizeSearch(body searchBody) (SearchRequest, *GatewayError) {
	invalid := func(message string) *GatewayError {
		return newGatewayError(http.StatusBadRequest, "invalid_request", message, "")
	}

	query := strings.TrimSpace(body.Query)
	if query == "" {
		return SearchRequest{}, invalid("query 不能为空")
	}
	if len([]rune(query)) > maxQueryLength {
		return SearchRequest{}, invalid("query 长度不能超过 400 个字符")
	}

	provider := strings.ToLower(strings.TrimSpace(body.Provider))
	switch provider {
	case "", providerAuto, search.ProviderBrave, search.ProviderTavily, search.ProviderExa:
	default:
		return SearchRequest{}, invalid("provider 仅支持 auto、brave、tavily、exa")
	}

	maxResults := body.MaxResults
	if maxResults == 0 {
		maxResults = 5
	}
	if maxResults < 1 || maxResults > 20 {
		return SearchRequest{}, invalid("max_results 取值范围为 1-20")
	}

	topic := strings.ToLower(strings.TrimSpace(body.Topic))
	switch topic {
	case "":
		topic = search.TopicGeneral
	case search.TopicGeneral, search.TopicNews:
	default:
		return SearchRequest{}, invalid("topic 仅支持 general、news")
	}

	freshness := strings.ToLower(strings.TrimSpace(body.Freshness))
	if freshness != "" && !validFreshness(freshness) {
		return SearchRequest{}, invalid("freshness 仅支持 day、week、month、year 或 YYYY-MM-DDtoYYYY-MM-DD")
	}

	country := strings.TrimSpace(body.Country)
	if country != "" && len(country) != 2 {
		return SearchRequest{}, invalid("country 必须是 2 位 ISO 国家代码")
	}

	if len(body.IncludeDomains) > maxDomainFilter || len(body.ExcludeDomains) > maxDomainFilter {
		return SearchRequest{}, invalid("域名过滤列表最多 50 项")
	}

	allowFallback := true
	if body.AllowFallback != nil {
		allowFallback = *body.AllowFallback
	}

	return SearchRequest{
		Query:             query,
		Provider:          provider,
		MaxResults:        maxResults,
		Topic:             topic,
		Freshness:         freshness,
		Country:           country,
		Language:          strings.TrimSpace(body.Language),
		IncludeDomains:    body.IncludeDomains,
		ExcludeDomains:    body.ExcludeDomains,
		IncludeAnswer:     body.IncludeAnswer,
		IncludeRawContent: body.IncludeRawContent,
		AllowFallback:     allowFallback,
		NoCache:           body.NoCache,
	}, nil
}

func validFreshness(freshness string) bool {
	switch freshness {
	case search.FreshnessDay, search.FreshnessWeek, search.FreshnessMonth, search.FreshnessYear:
		return true
	}
	return freshnessRangePattern.MatchString(freshness)
}

// freshnessRangePattern mirrors the custom range shape the adapters accept, so
// a value that would be silently dropped upstream is rejected here instead.
var freshnessRangePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}to\d{4}-\d{2}-\d{2}$`)

// providerStatus is one row of GET /api/gateway/v1/providers.
type providerStatus struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	HomeURL        string `json:"home_url"`
	DocsURL        string `json:"docs_url"`
	LogoURL        string `json:"logo_url"`
	Healthy        bool   `json:"healthy"`
	MonthlyQuota   int    `json:"monthly_quota"`
	MonthlyUsed    int    `json:"monthly_used"`
	SupportsAnswer bool   `json:"supports_answer"`
	SupportsRaw    bool   `json:"supports_raw_content"`
}

// Providers handles GET /api/gateway/v1/providers so an agent can see what the
// gateway can currently do before choosing parameters.
func (h *handler) Providers(c *gin.Context) {
	statuses, err := h.service.ProviderStatuses(c.Request.Context(), apiKeyFrom(c))
	if err != nil {
		writeGatewayError(c, asGatewayError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"providers": statuses})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func queryBool(raw string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && value
}
