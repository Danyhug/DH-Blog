package aigateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"dh-blog/internal/modules/agentapi"
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
// content-writing via the tools the agent module contributes.
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
// hand it to a protocol server assembled per request with the tools this key
// may see, write back whatever it says.
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

	key := apiKeyFrom(c)
	ctx := context.WithValue(c.Request.Context(), mcpKeyCtxKey{}, key)
	ctx = context.WithValue(ctx, mcpClientIPCtxKey{}, c.ClientIP())
	// The content-writing tools read the caller through agentapi.Identity; the
	// key now implements that contract, so one credential serves both layers.
	ctx = agentapi.IdentityContext(ctx, key)

	// The protocol server's tool table is fixed at construction, but the
	// visible tools are per key — assemble one here so a key without write
	// scopes never even sees the writing tools. The cost is a few small
	// structs per request.
	server := mcp.New(mcpServerName, mcpServerVersion, mcpInstructions)
	server.Register(h.mcpToolsFor(key)...)

	response, isNotification := server.Handle(ctx, body)
	if isNotification {
		c.Status(http.StatusAccepted)
		return
	}
	c.JSON(http.StatusOK, response)
}

// rateLimitedTool enforces the key's per-minute allowance on content tools.
// The search path already applies RateLimitPerMin internally; without this a
// stolen write key could spam creates and uploads at full speed.
type rateLimitedTool struct {
	mcp.Tool
	allow func() bool
}

func (t rateLimitedTool) Call(ctx context.Context, args json.RawMessage) mcp.Result {
	if !t.allow() {
		return mcp.ToolError("请求过于频繁，请稍后再试")
	}
	return t.Tool.Call(ctx, args)
}

// mcpToolsFor filters the tool table down to what this key may see. Extra
// tools that declare a Scope() are gated on the key's scopes; tools without
// one (search side) are always visible. The built-in web search is mounted
// unconditionally. The auth middleware guarantees a key here; a nil one
// defensively sees the base tool set unwrapped (rate limiting needs a key).
func (h *handler) mcpToolsFor(key *APIKey) []mcp.Tool {
	tools := make([]mcp.Tool, 0, len(h.extraTools)+1)
	tools = append(tools, h.webSearch)
	for _, tool := range h.extraTools {
		scoped, ok := tool.(interface{ Scope() string })
		if ok && (key == nil || !key.HasScope(scoped.Scope())) {
			continue
		}
		if key == nil {
			tools = append(tools, tool)
			continue
		}
		// Content tools share the search budget: one RateLimitPerMin pool per
		// key, regardless of which endpoint the call arrives on.
		tools = append(tools, rateLimitedTool{Tool: tool, allow: func() bool {
			return h.service.rateAllowed(key)
		}})
	}
	return tools
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
