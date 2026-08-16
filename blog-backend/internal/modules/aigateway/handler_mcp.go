package aigateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"dh-blog/internal/platform/mcp"

	"github.com/gin-gonic/gin"
)

// MCP server identity and transport limits. The name/version pair is what
// initialize advertises; clients mount the server under whatever name they
// configured themselves, so these only matter for display.
const (
	mcpServerName    = "dh-blog"
	mcpServerVersion = "2.0.0"
	// A long article's Markdown can already exceed the old limit and a base64
	// image always will; malicious oversized bodies are bounded by the API-key
	// auth and rate limiting instead.
	maxMCPRequestBody = 8 << 20
)

// mcpInstructions is handed to the model at initialize, so it reads as a direct
// statement of what this server is: the environment's search, plus blog
// content-writing once the writing tools land in a later step.
const mcpInstructions = "本服务器提供该环境的联网搜索能力，以及向本博客写入内容的能力。" +
	"需要现网信息时调用 web_search，它返回标题、链接与摘要，通常不必再逐条抓取网页。" +
	"需要向博客写入或修改内容时，使用 list_articles、get_article、create_article、" +
	"update_article、upload_image 等写作工具。" +
	"不要凭记忆回答会随时间变化的问题。"

// mcpKeyCtxKey and mcpClientIPCtxKey carry the caller's identity through the
// transport and into the tools: mcp.Tool methods only receive a context, so the
// HTTP layer stashes the authenticated key and the client IP it already knows.
type mcpKeyCtxKey struct{}

type mcpClientIPCtxKey struct{}

func mcpKeyFromContext(ctx context.Context) *APIKey {
	value, _ := ctx.Value(mcpKeyCtxKey{}).(*APIKey)
	return value
}

func mcpClientIPFromContext(ctx context.Context) string {
	value, _ := ctx.Value(mcpClientIPCtxKey{}).(string)
	return value
}

// mcpTransportError builds the JSON-RPC error the handler answers for problems
// the transport catches before the protocol layer could see the body. Such a
// request has no id we can echo, hence the fixed null.
func mcpTransportError(code int, message string) mcp.Response {
	return mcp.Response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &mcp.RPCError{Code: code, Message: message}}
}

// MCP handles POST /api/gateway/v1/mcp.
//
// Auth, rate limiting, quota, routing, caching and accounting all come from the
// group middleware and Service. The handler is only a transport: read the body,
// hand it to the protocol server assembled at construction time, write back
// whatever it says.
func (h *handler) MCP(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxMCPRequestBody+1))
	if err != nil {
		c.JSON(http.StatusOK, mcpTransportError(mcp.ParseError, "读取请求体失败"))
		return
	}
	if len(body) > maxMCPRequestBody {
		c.JSON(http.StatusOK, mcpTransportError(mcp.InvalidRequest, "请求体过大"))
		return
	}

	ctx := context.WithValue(c.Request.Context(), mcpKeyCtxKey{}, apiKeyFrom(c))
	ctx = context.WithValue(ctx, mcpClientIPCtxKey{}, c.ClientIP())

	response, isNotification := h.mcp.Handle(ctx, body)
	if isNotification {
		c.Status(http.StatusAccepted)
		return
	}
	c.JSON(http.StatusOK, response)
}

// MCPNotAllowed answers the transport's optional verbs. The gateway never
// pushes messages to the client, so there is no stream to open and no session
// to delete; the spec asks for 405 in exactly this case.
func (h *handler) MCPNotAllowed(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, errorBody{Error: errorDetail{
		Type:    "method_not_allowed",
		Message: "该端点只接受 POST，网关不提供服务端推送流",
	}})
}

func mcpFailureText(err error) string {
	var gatewayErr *GatewayError
	if errors.As(err, &gatewayErr) {
		if gatewayErr.Provider != "" {
			return "搜索失败（" + gatewayErr.Type + " / " + gatewayErr.Provider + "）：" + gatewayErr.Message
		}
		return "搜索失败（" + gatewayErr.Type + "）：" + gatewayErr.Message
	}
	return "搜索失败：" + err.Error()
}
