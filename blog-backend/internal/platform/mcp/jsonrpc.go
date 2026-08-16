// Package mcp implements the JSON half of the Model Context Protocol over
// Streamable HTTP: the JSON-RPC 2.0 envelope, protocol error codes, version
// negotiation, a tool registry and dispatch. It holds no business logic — tool
// definitions and the HTTP transport belong to the modules that mount a Server.
package mcp

import (
	"encoding/json"
	"strings"
)

// JSON-RPC 2.0 error codes, as fixed by the spec. The gateway answers
// transport-level problems (unreadable body, oversized request) with these
// before the protocol layer ever sees the bytes.
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// DefaultProtocolVersion is what negotiation falls back to when the client asks
// for a version we do not speak.
const DefaultProtocolVersion = "2025-06-18"

// SupportedProtocolVersions is the subset of MCP protocol versions this server
// actually speaks. Keep it to versions we have tested against; every entry is
// echoed back verbatim by negotiateProtocolVersion.
var SupportedProtocolVersions = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
}

// request is one inbound JSON-RPC 2.0 message. ID is kept raw so a response
// can echo whatever shape the client used without reinterpretation.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// RPCError is one JSON-RPC error object inside a Response.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Response is a JSON-RPC 2.0 reply. It is exported so the module mounting the
// server can unmarshal what it writes back — tests decode the wire format — and
// can build its own error replies for transport-level problems the protocol
// layer never sees.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// nullID is the id a response carries when the request was too broken to have one.
var nullID = json.RawMessage("null")

func rpcResult(id json.RawMessage, result any) Response {
	if len(id) == 0 {
		id = nullID
	}
	return Response{JSONRPC: "2.0", ID: id, Result: result}
}

func rpcFailure(id json.RawMessage, code int, message string) Response {
	if len(id) == 0 {
		id = nullID
	}
	return Response{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: message}}
}

// isNotification reports whether the message expects no response. JSON-RPC
// notifications carry no id; MCP sends `notifications/initialized` this way.
func (r request) isNotification() bool {
	trimmed := strings.TrimSpace(string(r.ID))
	return trimmed == "" || trimmed == "null"
}
