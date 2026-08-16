// Package mcp implements the JSON half of the Model Context Protocol over
// Streamable HTTP: the JSON-RPC 2.0 envelope, protocol error codes, version
// negotiation, a tool registry and dispatch. It holds no business logic — tool
// definitions and the HTTP transport belong to the modules that mount a Server.
package mcp

import (
	"encoding/json"
	"strings"
)

// JSON-RPC 2.0 error codes, as fixed by the spec.
const (
	parseError     = -32700
	invalidRequest = -32600
	methodNotFound = -32601
	invalidParams  = -32602
	internalError  = -32603
)

// defaultProtocolVersion is what negotiation falls back to when the client asks
// for a version we do not speak.
const defaultProtocolVersion = "2025-06-18"

// supportedProtocolVersions is the subset of MCP protocol versions this server
// actually speaks. Keep it to versions we have tested against; every entry is
// echoed back verbatim by negotiateProtocolVersion.
var supportedProtocolVersions = map[string]bool{
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

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// nullID is the id a response carries when the request was too broken to have one.
var nullID = json.RawMessage("null")

func rpcResult(id json.RawMessage, result any) response {
	if len(id) == 0 {
		id = nullID
	}
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func rpcFailure(id json.RawMessage, code int, message string) response {
	if len(id) == 0 {
		id = nullID
	}
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

// isNotification reports whether the message expects no response. JSON-RPC
// notifications carry no id; MCP sends `notifications/initialized` this way.
func (r request) isNotification() bool {
	trimmed := strings.TrimSpace(string(r.ID))
	return trimmed == "" || trimmed == "null"
}
