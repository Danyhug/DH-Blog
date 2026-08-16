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
	// Name identifies the tool for tools/call lookups. It is kept separate from
	// Definition so dispatch never has to render a full definition just to match
	// a name — a tool's description can be expensive and caller-dependent.
	Name() string
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
		return rpcFailure(nil, InvalidRequest, "不支持 JSON-RPC 批量请求"), false
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return rpcFailure(nil, ParseError, "请求体不是合法 JSON"), false
	}
	if req.Method == "" {
		return rpcFailure(req.ID, InvalidRequest, "缺少 method"), false
	}
	// Notifications carry no id and expect no response; MCP uses them for
	// notifications/initialized. Skipping them entirely keeps the transport
	// stateless — no queued messages, no sessions.
	if req.isNotification() {
		return nil, true
	}

	return s.dispatch(ctx, req), false
}

func (s *Server) dispatch(ctx context.Context, req request) Response {
	switch req.Method {
	case "initialize":
		return rpcResult(req.ID, s.initialize(req))
	case "ping":
		return rpcResult(req.ID, struct{}{})
	case "tools/list":
		return rpcResult(req.ID, ToolListResult{Tools: s.definitions(ctx)})
	case "tools/call":
		return s.callTool(ctx, req)
	default:
		return rpcFailure(req.ID, MethodNotFound, "不支持的方法: "+req.Method)
	}
}

// ServerInfo identifies the server to the client at initialize.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsCapability declares the tools feature in the capabilities block.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// Capabilities is what the server says it can do at initialize.
type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// InitializeParams is the part of the initialize request that version
// negotiation reads.
type InitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// InitializeResult is the initialize reply.
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Instructions    string       `json:"instructions,omitempty"`
}

func (s *Server) initialize(req request) InitializeResult {
	var params InitializeParams
	// Version negotiation is lenient by design: a malformed params block means
	// the client sent nothing we can honour, so fall back to the default.
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}
	return InitializeResult{
		ProtocolVersion: negotiateProtocolVersion(params.ProtocolVersion),
		Capabilities:    Capabilities{Tools: &ToolsCapability{ListChanged: false}},
		ServerInfo:      ServerInfo{Name: s.name, Version: s.version},
		Instructions:    s.instructions,
	}
}

// negotiateProtocolVersion echoes the client's version when it is one we speak.
// Echoing an unknown version back would claim fluency we lack, so those fall
// back to the default.
func negotiateProtocolVersion(requested string) string {
	if SupportedProtocolVersions[requested] {
		return requested
	}
	return DefaultProtocolVersion
}

// ToolListResult is the tools/list reply.
type ToolListResult struct {
	Tools []Definition `json:"tools"`
}

func (s *Server) definitions(ctx context.Context) []Definition {
	definitions := make([]Definition, 0, len(s.tools))
	for _, tool := range s.tools {
		definitions = append(definitions, tool.Definition(ctx))
	}
	return definitions
}

// ToolCallParams is the tools/call request shape.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) callTool(ctx context.Context, req request) Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return rpcFailure(req.ID, InvalidParams, "params 解析失败: "+err.Error())
	}
	if params.Name == "" {
		return rpcFailure(req.ID, InvalidParams, "缺少工具名")
	}
	tool := s.findTool(params.Name)
	if tool == nil {
		return rpcFailure(req.ID, InvalidParams, "未知的工具: "+params.Name)
	}
	// The tool only ever sees arguments it can parse (or nil when absent):
	// arguments is a raw message inside params, so params having parsed means
	// arguments is valid JSON too. A type-level mismatch inside the payload is
	// the tool's own concern and surfaces as an isError result.
	return rpcResult(req.ID, tool.Call(ctx, params.Arguments))
}

func (s *Server) findTool(name string) Tool {
	for _, tool := range s.tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}
