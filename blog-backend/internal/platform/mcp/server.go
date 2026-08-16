package mcp

import (
	"context"
	"encoding/json"
	"strings"
)

// Definition is a tool's description, what tools/list advertises to the model.
type Definition struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// TextContent is one text block inside a tool Result.
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Result is a tool's answer. IsError marks execution failure: MCP wants tool
// failures inside the result rather than as a JSON-RPC error, so the model can
// read what went wrong and adjust instead of the client surfacing a transport
// error.
type Result struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Text wraps a plain-text tool answer.
func Text(text string) Result {
	return Result{Content: []TextContent{{Type: "text", Text: text}}}
}

// ToolError wraps a failed tool run as a result, so failures stay visible to
// the model instead of ending up as transport-level errors.
func ToolError(text string) Result {
	result := Text(text)
	result.IsError = true
	return result
}

// Tool is a tool mounted on the MCP endpoint.
type Tool interface {
	// Definition receives ctx because a tool's description may depend on the
	// caller's identity: web_search's provider enum depends on which providers
	// the current key can reach.
	Definition(ctx context.Context) Definition
	Call(ctx context.Context, args json.RawMessage) Result
}

// Server implements the JSON half of MCP over Streamable HTTP: it turns raw
// request bodies into JSON-RPC responses and dispatches tools/call onto the
// registered tools. The HTTP layer, auth, rate limiting and accounting stay
// with the module that mounts the server, so this type stays business-free.
type Server struct {
	name         string
	version      string
	instructions string
	tools        []Tool
}

// New builds a Server that announces name/version/instructions at initialize.
func New(name, version, instructions string) *Server {
	return &Server{name: name, version: version, instructions: instructions}
}

// Register appends tools to the registry. tools/list reports them in order.
func (s *Server) Register(tools ...Tool) {
	s.tools = append(s.tools, tools...)
}

// Handle processes one JSON-RPC request body and returns the response to write,
// or isNotification=true when the request expects none (the caller answers HTTP
// 202 with no body). The response can be marshalled as-is.
func (s *Server) Handle(ctx context.Context, body []byte) (response any, isNotification bool) {
	// JSON-RPC batch requests were removed from MCP in 2025-06-18; rejecting
	// them explicitly is clearer than silently processing the first element.
	if trimmed := strings.TrimSpace(string(body)); strings.HasPrefix(trimmed, "[") {
		return rpcFailure(nil, invalidRequest, "不支持 JSON-RPC 批量请求"), false
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return rpcFailure(nil, parseError, "请求体不是合法 JSON"), false
	}
	if req.Method == "" {
		return rpcFailure(req.ID, invalidRequest, "缺少 method"), false
	}
	// Notifications carry no id and expect no response; MCP uses them for
	// notifications/initialized. Skipping them entirely keeps the transport
	// stateless — no queued messages, no sessions.
	if req.isNotification() {
		return nil, true
	}

	return s.dispatch(ctx, req), false
}

func (s *Server) dispatch(ctx context.Context, req request) response {
	switch req.Method {
	case "initialize":
		return rpcResult(req.ID, s.initialize(req))
	case "ping":
		return rpcResult(req.ID, struct{}{})
	case "tools/list":
		return rpcResult(req.ID, toolListResult{Tools: s.definitions(ctx)})
	case "tools/call":
		return s.callTool(ctx, req)
	default:
		return rpcFailure(req.ID, methodNotFound, "不支持的方法: "+req.Method)
	}
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type capabilities struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
	Instructions    string       `json:"instructions,omitempty"`
}

func (s *Server) initialize(req request) initializeResult {
	var params initializeParams
	// Version negotiation is lenient by design: a malformed params block means
	// the client sent nothing we can honour, so fall back to the default.
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}
	return initializeResult{
		ProtocolVersion: negotiateProtocolVersion(params.ProtocolVersion),
		Capabilities:    capabilities{Tools: &toolsCapability{ListChanged: false}},
		ServerInfo:      serverInfo{Name: s.name, Version: s.version},
		Instructions:    s.instructions,
	}
}

// negotiateProtocolVersion echoes the client's version when it is one we speak.
// Echoing an unknown version back would claim fluency we lack, so those fall
// back to the default.
func negotiateProtocolVersion(requested string) string {
	if supportedProtocolVersions[requested] {
		return requested
	}
	return defaultProtocolVersion
}

type toolListResult struct {
	Tools []Definition `json:"tools"`
}

func (s *Server) definitions(ctx context.Context) []Definition {
	definitions := make([]Definition, 0, len(s.tools))
	for _, tool := range s.tools {
		definitions = append(definitions, tool.Definition(ctx))
	}
	return definitions
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) callTool(ctx context.Context, req request) response {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return rpcFailure(req.ID, invalidParams, "params 解析失败: "+err.Error())
	}
	if params.Name == "" {
		return rpcFailure(req.ID, invalidParams, "缺少工具名")
	}
	tool := s.findTool(ctx, params.Name)
	if tool == nil {
		return rpcFailure(req.ID, invalidParams, "未知的工具: "+params.Name)
	}
	// An unparseable arguments block is a protocol error, not a tool failure:
	// the tool only ever sees JSON it can parse (or nil when absent).
	if len(params.Arguments) > 0 && !json.Valid(params.Arguments) {
		return rpcFailure(req.ID, invalidParams, "arguments 不是合法 JSON")
	}
	return rpcResult(req.ID, tool.Call(ctx, params.Arguments))
}

func (s *Server) findTool(ctx context.Context, name string) Tool {
	for _, tool := range s.tools {
		if tool.Definition(ctx).Name == name {
			return tool
		}
	}
	return nil
}
