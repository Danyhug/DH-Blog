package aigateway

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MCP handles POST /api/gateway/v1/mcp.
//
// Auth, rate limiting, quota, routing, caching and accounting all come from the
// group middleware and Service, so this handler is only a protocol translation:
// JSON-RPC in, the same SearchRequest the HTTP API validates, text out.
func (h *handler) MCP(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxMCPRequestBody+1))
	if err != nil {
		c.JSON(http.StatusOK, rpcFailure(nil, jsonRPCParseError, "读取请求体失败"))
		return
	}
	if len(body) > maxMCPRequestBody {
		c.JSON(http.StatusOK, rpcFailure(nil, jsonRPCInvalidRequest, "请求体过大"))
		return
	}

	// JSON-RPC 批量请求在 MCP 2025-06-18 里已被移除，明确拒掉比静默只处理第一条好
	if trimmed := strings.TrimSpace(string(body)); strings.HasPrefix(trimmed, "[") {
		c.JSON(http.StatusOK, rpcFailure(nil, jsonRPCInvalidRequest, "不支持 JSON-RPC 批量请求"))
		return
	}

	var request jsonRPCRequest
	if err := json.Unmarshal(body, &request); err != nil {
		c.JSON(http.StatusOK, rpcFailure(nil, jsonRPCParseError, "请求体不是合法 JSON"))
		return
	}
	if request.Method == "" {
		c.JSON(http.StatusOK, rpcFailure(request.ID, jsonRPCInvalidRequest, "缺少 method"))
		return
	}

	// 通知没有 id，按 JSON-RPC 不能回响应；MCP 用它发 notifications/initialized
	if request.isNotification() {
		c.Status(http.StatusAccepted)
		return
	}

	c.JSON(http.StatusOK, h.dispatchMCP(c, request))
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

func (h *handler) dispatchMCP(c *gin.Context, request jsonRPCRequest) jsonRPCResponse {
	switch request.Method {
	case "initialize":
		return rpcResult(request.ID, h.mcpInitialize(request))
	case "ping":
		return rpcResult(request.ID, struct{}{})
	case "tools/list":
		return h.mcpToolList(c, request)
	case "tools/call":
		return h.mcpToolCall(c, request)
	default:
		return rpcFailure(request.ID, jsonRPCMethodNotFound, "不支持的方法: "+request.Method)
	}
}

func (h *handler) mcpInitialize(request jsonRPCRequest) mcpInitializeResult {
	var params mcpInitializeParams
	if len(request.Params) > 0 {
		_ = json.Unmarshal(request.Params, &params)
	}
	return mcpInitializeResult{
		ProtocolVersion: negotiateProtocolVersion(params.ProtocolVersion),
		Capabilities:    mcpCapabilities{Tools: &mcpToolsCapability{ListChanged: false}},
		ServerInfo:      mcpServerInfo{Name: mcpServerName, Version: mcpServerVersion},
		// 客户端会把这段交给模型，所以这里直说它是本环境的联网搜索，
		// 而不是一个"另外还有的"检索来源
		Instructions: "本服务器提供该环境的联网搜索能力。需要现网信息时调用 web_search，" +
			"它返回标题、链接与摘要，通常不必再逐条抓取网页。" +
			"不要凭记忆回答会随时间变化的问题。",
	}
}

func (h *handler) mcpToolList(c *gin.Context, request jsonRPCRequest) jsonRPCResponse {
	providers, err := h.service.ProviderStatuses(c.Request.Context(), apiKeyFrom(c))
	if err != nil {
		return rpcFailure(request.ID, jsonRPCInternalError, err.Error())
	}
	return rpcResult(request.ID, mcpToolListResult{Tools: []mcpTool{webSearchTool(providers)}})
}

func (h *handler) mcpToolCall(c *gin.Context, request jsonRPCRequest) jsonRPCResponse {
	var params mcpToolCallParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return rpcFailure(request.ID, jsonRPCInvalidParams, "params 解析失败: "+err.Error())
	}
	if params.Name != mcpToolWebSearch {
		return rpcFailure(request.ID, jsonRPCInvalidParams, "未知的工具: "+params.Name)
	}

	// 工具参数与 HTTP 接口的 body 是同一套字段，校验也就共用同一份实现
	var arguments mcpSearchArguments
	if len(params.Arguments) > 0 {
		if err := json.Unmarshal(params.Arguments, &arguments); err != nil {
			return rpcFailure(request.ID, jsonRPCInvalidParams, "arguments 解析失败: "+err.Error())
		}
	}
	req, invalid := normalizeSearch(arguments.merged())
	if invalid != nil {
		return rpcFailure(request.ID, jsonRPCInvalidParams, invalid.Message)
	}

	result, searchErr := h.service.SearchFrom(c.Request.Context(), apiKeyFrom(c), req, c.ClientIP(), "mcp/search")
	if searchErr != nil {
		// 限流、配额、上游故障都是执行期失败：写进结果让模型自己决定要不要换个问法或放弃
		return rpcResult(request.ID, mcpToolError(mcpFailureText(searchErr)))
	}
	return rpcResult(request.ID, mcpText(renderSearchResult(result)))
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
