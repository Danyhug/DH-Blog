package aigateway

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxFetchURLLength = 2048

// fetchBody is the wire shape of a page fetch. The length fields are pointers
// because "not supplied" picks a per-entry default: the whole page over HTTP,
// a context-sized piece over MCP.
type fetchBody struct {
	URL        string `json:"url"`
	MaxLength  *int   `json:"max_length"`
	StartIndex int    `json:"start_index"`
	NoCache    bool   `json:"no_cache"`
}

// Fetch handles POST /api/gateway/v1/fetch.
func (h *handler) Fetch(c *gin.Context) {
	var body fetchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeGatewayError(c, newGatewayError(http.StatusBadRequest, "invalid_request", "请求体解析失败: "+err.Error(), ""))
		return
	}
	h.runFetch(c, body)
}

// FetchGET handles GET /api/gateway/v1/fetch?url=..., for curl and minimal agents.
func (h *handler) FetchGET(c *gin.Context) {
	body := fetchBody{URL: c.Query("url"), NoCache: queryBool(c.Query("no_cache"))}
	if raw := c.Query("max_length"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeGatewayError(c, newGatewayError(http.StatusBadRequest, "invalid_request", "max_length 必须是整数", ""))
			return
		}
		body.MaxLength = &parsed
	}
	if raw := c.Query("start_index"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeGatewayError(c, newGatewayError(http.StatusBadRequest, "invalid_request", "start_index 必须是整数", ""))
			return
		}
		body.StartIndex = parsed
	}
	h.runFetch(c, body)
}

func (h *handler) runFetch(c *gin.Context, body fetchBody) {
	// Over HTTP the caller owns its buffer, so an unset length means the whole page.
	req, err := normalizeFetch(body, 0, 0)
	if err != nil {
		writeGatewayError(c, err)
		return
	}
	result, fetchErr := h.service.Fetch(c.Request.Context(), apiKeyFrom(c), req, c.ClientIP(), "fetch")
	if fetchErr != nil {
		writeGatewayError(c, asGatewayError(fetchErr))
		return
	}
	c.JSON(http.StatusOK, result)
}

// normalizeFetch validates a fetch. defaultLength applies when max_length is
// absent; limit, when positive, caps an explicit max_length.
func normalizeFetch(body fetchBody, defaultLength, limit int) (FetchRequest, *GatewayError) {
	invalid := func(message string) *GatewayError {
		return newGatewayError(http.StatusBadRequest, "invalid_request", message, "")
	}

	target, message := validateFetchURL(body.URL)
	if message != "" {
		return FetchRequest{}, invalid(message)
	}

	length := defaultLength
	if body.MaxLength != nil {
		length = *body.MaxLength
		if length < 1 {
			return FetchRequest{}, invalid("max_length 必须大于 0")
		}
		if limit > 0 && length > limit {
			return FetchRequest{}, invalid("max_length 不能超过 " + strconv.Itoa(limit))
		}
	}
	if body.StartIndex < 0 {
		return FetchRequest{}, invalid("start_index 不能为负数")
	}
	return FetchRequest{URL: target, MaxLength: length, StartIndex: body.StartIndex, NoCache: body.NoCache}, nil
}

// validateFetchURL accepts a public http(s) URL and returns it normalised, or
// a reason it was refused.
//
// The provider fetches the page, not the gateway, so this is not the gateway
// protecting its own network. It is the gateway refusing to be the way in to
// someone else's: a self-hosted Firecrawl sits on the operator's network and
// would happily read http://192.168.1.1/ for anyone holding a search key.
// Literal private addresses and localhost names are refused; a public name
// that resolves to a private address is the provider's to guard against.
func validateFetchURL(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "url 不能为空"
	}
	if len(raw) > maxFetchURLLength {
		return "", "url 长度不能超过 2048 个字符"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "url 格式不正确"
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", "url 只支持 http 与 https"
	}
	// Credentials in the URL would be handed to the provider verbatim.
	if parsed.User != nil {
		return "", "url 不能包含用户名或密码"
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" {
		return "", "url 缺少主机名"
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", "不能抓取本机地址"
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
			ip.IsUnspecified() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
			return "", "不能抓取内网或保留地址"
		}
	}
	parsed.Scheme = scheme
	parsed.Fragment = ""
	return parsed.String(), ""
}
