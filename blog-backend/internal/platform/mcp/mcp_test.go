package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// stubTool is a fixed tool for protocol tests. It records how it was called so
// tests can assert the arguments pass through untouched.
type stubTool struct {
	name        string
	description string
	result      Result
	defCtx      context.Context
	callCtx     context.Context
	gotArgs     json.RawMessage
}

func (t *stubTool) Name() string { return t.name }

func (t *stubTool) Definition(ctx context.Context) Definition {
	t.defCtx = ctx
	return Definition{
		Name:        t.name,
		Description: t.description,
		InputSchema: map[string]any{"type": "object"},
	}
}

func (t *stubTool) Call(ctx context.Context, args json.RawMessage) Result {
	t.callCtx = ctx
	t.gotArgs = args
	return t.result
}

func newTestServer(tools ...*stubTool) (*Server, []*stubTool) {
	server := New("test-server", "1.0.0", "测试用 MCP 服务")
	interfaceTools := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		interfaceTools = append(interfaceTools, tool)
	}
	server.Register(interfaceTools...)
	return server, tools
}

// handleRPC runs Handle and decodes the JSON-RPC response as the wire format
// would, so tests exercise the actual serialized shape.
func handleRPC(t *testing.T, server *Server, body string) (Response, bool) {
	t.Helper()
	got, isNotification := server.Handle(context.Background(), []byte(body))
	if isNotification {
		return Response{}, true
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("序列化响应失败: %v", err)
	}
	var decoded Response
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("解析响应失败: %v (body=%s)", err, raw)
	}
	if decoded.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, 期望 2.0", decoded.JSONRPC)
	}
	return decoded, false
}

func rpcResultAs[T any](t *testing.T, decoded Response) T {
	t.Helper()
	if decoded.Error != nil {
		t.Fatalf("期望成功但收到错误: %+v", decoded.Error)
	}
	raw, err := json.Marshal(decoded.Result)
	if err != nil {
		t.Fatalf("序列化 result 失败: %v", err)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("解析 result 失败: %v (result=%s)", err, raw)
	}
	return value
}

func TestInitializeNegotiatesProtocolVersion(t *testing.T) {
	server, _ := newTestServer()

	tests := []struct {
		name      string
		requested string
		want      string
	}{
		{"支持的最新版本", "2025-06-18", "2025-06-18"},
		{"支持的中间版本", "2025-03-26", "2025-03-26"},
		{"支持的最早版本", "2024-11-05", "2024-11-05"},
		// 认不出来的版本不能照抄回去，否则等于谎称自己会说这套协议
		{"未知版本退回默认", "1999-01-01", "2025-06-18"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` +
				test.requested + `","capabilities":{},"clientInfo":{"name":"claude-code","version":"1"}}}`
			decoded, _ := handleRPC(t, server, body)
			result := rpcResultAs[InitializeResult](t, decoded)
			if result.ProtocolVersion != test.want {
				t.Errorf("protocolVersion = %q, 期望 %q", result.ProtocolVersion, test.want)
			}
			if result.ServerInfo.Name != "test-server" || result.ServerInfo.Version != "1.0.0" {
				t.Errorf("serverInfo = %+v, 期望构造参数传入的 name/version", result.ServerInfo)
			}
			if result.Instructions != "测试用 MCP 服务" {
				t.Errorf("instructions = %q", result.Instructions)
			}
			if result.Capabilities.Tools == nil {
				t.Error("未声明 tools 能力，客户端不会去拉工具列表")
			}
		})
	}
}

func TestInitializeWithoutParamsFallsBackToDefault(t *testing.T) {
	server, _ := newTestServer()

	decoded, _ := handleRPC(t, server, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	result := rpcResultAs[InitializeResult](t, decoded)
	if result.ProtocolVersion != "2025-06-18" {
		t.Errorf("protocolVersion = %q, 期望默认 2025-06-18", result.ProtocolVersion)
	}
}

func TestPing(t *testing.T) {
	server, _ := newTestServer()

	decoded, _ := handleRPC(t, server, `{"jsonrpc":"2.0","id":9,"method":"ping"}`)
	if decoded.Error != nil {
		t.Fatalf("ping 返回错误: %+v", decoded.Error)
	}
	if string(decoded.ID) != "9" {
		t.Errorf("响应 id = %s, 期望回显请求的 9", decoded.ID)
	}
}

func TestToolListAdvertisesRegisteredToolsInOrder(t *testing.T) {
	server, tools := newTestServer(
		&stubTool{name: "first", description: "第一个工具"},
		&stubTool{name: "second", description: "第二个工具"},
	)

	decoded, _ := handleRPC(t, server, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	list := rpcResultAs[ToolListResult](t, decoded)
	if len(list.Tools) != 2 {
		t.Fatalf("工具数量 = %d, 期望 2", len(list.Tools))
	}
	if list.Tools[0].Name != "first" || list.Tools[1].Name != "second" {
		t.Errorf("工具顺序 = %+v, 期望按注册顺序", list.Tools)
	}
	if list.Tools[0].Description != "第一个工具" {
		t.Errorf("description = %q", list.Tools[0].Description)
	}
	if tools[0].defCtx == nil {
		t.Error("Definition 没有收到 ctx")
	}
}

func TestToolCallPassesArgumentsAndReturnsResult(t *testing.T) {
	server, tools := newTestServer(
		&stubTool{name: "echo", result: Text("好的")},
	)

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"q":"hi"}}}`
	decoded, _ := handleRPC(t, server, body)
	result := rpcResultAs[Result](t, decoded)
	if result.IsError {
		t.Fatalf("工具调用失败: %+v", result.Content)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "text" || result.Content[0].Text != "好的" {
		t.Errorf("content = %+v", result.Content)
	}
	if string(tools[0].gotArgs) != `{"q":"hi"}` {
		t.Errorf("工具收到的 arguments = %s, 期望原样透传", tools[0].gotArgs)
	}
	if tools[0].callCtx == nil {
		t.Error("Call 没有收到 ctx")
	}
}

func TestToolCallFindsToolByNameWithoutCallingDefinition(t *testing.T) {
	server, tools := newTestServer(
		&stubTool{name: "echo", result: Text("好的")},
	)

	decoded, _ := handleRPC(t, server, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`)
	if decoded.Error != nil {
		t.Fatalf("按名字查找工具失败: %+v", decoded.Error)
	}
	if tools[0].defCtx != nil {
		t.Error("tools/call 不该为解析工具名调用 Definition，名字查找应当走 Name()")
	}
}

func TestToolCallWithoutArgumentsPassesNil(t *testing.T) {
	server, tools := newTestServer(
		&stubTool{name: "noop", result: Text("ok")},
	)

	decoded, _ := handleRPC(t, server, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"noop"}}`)
	if decoded.Error != nil {
		t.Fatalf("无 arguments 的调用不应失败: %+v", decoded.Error)
	}
	if tools[0].gotArgs != nil {
		t.Errorf("工具收到的 arguments = %s, 期望 nil", tools[0].gotArgs)
	}
}

func TestToolCallExecutionFailureIsResultErrorNotProtocolError(t *testing.T) {
	server, _ := newTestServer(
		&stubTool{name: "flaky", result: ToolError("上游挂掉了")},
	)

	body := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"flaky","arguments":{}}}`
	decoded, _ := handleRPC(t, server, body)
	// 执行期失败要放进 result.isError 让模型看得见，而不是变成协议错误
	if decoded.Error != nil {
		t.Fatalf("工具失败被报成了 JSON-RPC 错误: %+v", decoded.Error)
	}
	result := rpcResultAs[Result](t, decoded)
	if !result.IsError {
		t.Fatal("工具失败未标记 isError")
	}
	if result.Content[0].Text != "上游挂掉了" {
		t.Errorf("失败文本 = %q", result.Content[0].Text)
	}
}

func TestProtocolErrors(t *testing.T) {
	server, _ := newTestServer(
		&stubTool{name: "echo", result: Text("ok")},
	)

	tests := []struct {
		name string
		body string
		want int
	}{
		{
			name: "未知方法",
			body: `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`,
			want: MethodNotFound,
		},
		{
			name: "未知工具",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"delete_everything","arguments":{}}}`,
			want: InvalidParams,
		},
		{
			name: "缺少工具名",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"arguments":{}}}`,
			want: InvalidParams,
		},
		{
			name: "params 不是合法 JSON",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":"not json"}`,
			want: InvalidParams,
		},
		{
			name: "批量请求",
			body: `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`,
			want: InvalidRequest,
		},
		{
			name: "缺少 method",
			body: `{"jsonrpc":"2.0","id":1}`,
			want: InvalidRequest,
		},
		{
			name: "body 不是 JSON",
			body: `not json`,
			want: ParseError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoded, isNotification := handleRPC(t, server, test.body)
			if isNotification {
				t.Fatalf("协议错误不应被当成通知")
			}
			if decoded.Error == nil {
				t.Fatalf("期望 JSON-RPC 错误, 实际 result=%+v", decoded.Result)
			}
			if decoded.Error.Code != test.want {
				t.Errorf("错误码 = %d, 期望 %d", decoded.Error.Code, test.want)
			}
		})
	}
}

func TestNotificationReturnsNoResponse(t *testing.T) {
	server, _ := newTestServer()

	tests := []struct {
		name string
		body string
	}{
		{"无 id", `{"jsonrpc":"2.0","method":"notifications/initialized"}`},
		{"id 为 JSON null", `{"jsonrpc":"2.0","id":null,"method":"ping"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, isNotification := server.Handle(context.Background(), []byte(test.body))
			if !isNotification {
				t.Fatalf("期望按通知处理, 实际返回 %+v", got)
			}
			if got != nil {
				t.Errorf("通知不应有响应体, 实际 = %+v", got)
			}
		})
	}
}

func TestBrokenRequestsGetNullID(t *testing.T) {
	server, _ := newTestServer()

	tests := []struct {
		name string
		body string
	}{
		{"非法 JSON 没有 id 可用", `not json`},
		{"批量请求没有 id", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoded, _ := handleRPC(t, server, test.body)
			if strings.TrimSpace(string(decoded.ID)) != "null" {
				t.Errorf("响应 id = %s, 期望 null", decoded.ID)
			}
		})
	}
}
