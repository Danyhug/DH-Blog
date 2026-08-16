package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dh-blog/internal/platform/mcp"
	"dh-blog/internal/platform/search"

	"github.com/gin-gonic/gin"
)

func doMCP(engine *gin.Engine, token, body string) *httptest.ResponseRecorder {
	return doGateway(engine, http.MethodPost, "/api/gateway/v1/mcp", token, body)
}

func decodeRPC(t *testing.T, recorder *httptest.ResponseRecorder) mcp.Response {
	t.Helper()
	var response mcp.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析 JSON-RPC 响应失败: %v (body=%s)", err, recorder.Body.String())
	}
	if response.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, 期望 2.0", response.JSONRPC)
	}
	return response
}

// rpcResultAs re-marshals the loosely typed result into the shape the caller expects.
func rpcResultAs[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	response := decodeRPC(t, recorder)
	if response.Error != nil {
		t.Fatalf("期望成功但收到错误: %+v", response.Error)
	}
	raw, err := json.Marshal(response.Result)
	if err != nil {
		t.Fatalf("序列化 result 失败: %v", err)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("解析 result 失败: %v (result=%s)", err, raw)
	}
	return value
}

func rpcErrorCode(t *testing.T, recorder *httptest.ResponseRecorder) int {
	t.Helper()
	response := decodeRPC(t, recorder)
	if response.Error == nil {
		t.Fatalf("期望 JSON-RPC 错误, 实际 body=%s", recorder.Body.String())
	}
	return response.Error.Code
}

func TestMCPInitializeNegotiatesProtocolVersion(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	tests := []struct {
		requested string
		want      string
	}{
		{"2025-06-18", "2025-06-18"},
		{"2025-03-26", "2025-03-26"},
		{"2024-11-05", "2024-11-05"},
		// 认不出来的版本不能照抄回去，否则等于谎称自己会说这套协议
		{"1999-01-01", mcp.DefaultProtocolVersion},
		{"", mcp.DefaultProtocolVersion},
	}
	for _, test := range tests {
		t.Run(test.requested, func(t *testing.T) {
			body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` +
				test.requested + `","capabilities":{},"clientInfo":{"name":"claude-code","version":"1"}}}`
			result := rpcResultAs[mcp.InitializeResult](t, doMCP(engine, token, body))
			if result.ProtocolVersion != test.want {
				t.Errorf("protocolVersion = %q, 期望 %q", result.ProtocolVersion, test.want)
			}
			if result.ServerInfo.Name != mcpServerName {
				t.Errorf("serverInfo.name = %q, 期望 %q", result.ServerInfo.Name, mcpServerName)
			}
			if result.Capabilities.Tools == nil {
				t.Error("未声明 tools 能力，客户端不会去拉工具列表")
			}
			// instructions 是客户端直接交给模型的，必须说清这个 server 能干什么
			if !strings.Contains(result.Instructions, "搜索") || !strings.Contains(result.Instructions, "写作") {
				t.Errorf("instructions 应同时涵盖搜索与写作: %q", result.Instructions)
			}
		})
	}
}

func TestMCPRequiresAPIKey(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)

	recorder := doMCP(engine, "", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, 期望 401", recorder.Code)
	}
}

func TestMCPNotificationGetsNoResponseBody(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doMCP(engine, token, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d, 期望 202", recorder.Code)
	}
	if body := strings.TrimSpace(recorder.Body.String()); body != "" {
		t.Fatalf("通知不应有响应体, 实际 = %q", body)
	}
}

func TestMCPToolListReflectsKeyAllowlist(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  braveOK("b1"),
		Tavily: tavilyOK("", "t1"),
	})
	engine := newTestEngine(module)
	// 这把 key 只被允许用 tavily，工具 schema 就不该把 brave 摆出来给模型选
	token := issueTestKey(t, module, func(key *APIKey) { key.AllowedProviders = "tavily" })

	result := rpcResultAs[mcp.ToolListResult](t, doMCP(engine, token, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	if len(result.Tools) != 1 || result.Tools[0].Name != mcpToolWebSearch {
		t.Fatalf("工具列表 = %+v", result.Tools)
	}

	enum := providerEnumOf(t, result.Tools[0])
	if len(enum) != 2 || enum[0] != providerAuto || enum[1] != "tavily" {
		t.Fatalf("provider 可选值 = %v, 期望 [auto tavily]", enum)
	}
	if strings.Contains(result.Tools[0].Description, "brave") {
		t.Error("工具说明里出现了这把 key 用不了的供应商")
	}
}

func providerEnumOf(t *testing.T, tool mcp.Definition) []string {
	t.Helper()
	schema, ok := tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("inputSchema 类型异常: %T", tool.InputSchema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("inputSchema.properties 缺失: %+v", schema)
	}
	provider, ok := properties["provider"].(map[string]any)
	if !ok {
		t.Fatalf("provider 参数缺失: %+v", properties)
	}
	raw, ok := provider["enum"].([]any)
	if !ok {
		t.Fatalf("provider.enum 缺失: %+v", provider)
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		values = append(values, item.(string))
	}
	return values
}

func TestMCPToolCallSearchesAndLabelsLog(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1", "b2")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"web_search",
		"arguments":{"query":"go 1.25","max_results":2}}}`
	result := rpcResultAs[mcp.Result](t, doMCP(engine, token, body))

	if result.IsError {
		t.Fatalf("工具调用失败: %+v", result.Content)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "text" {
		t.Fatalf("content = %+v", result.Content)
	}
	text := result.Content[0].Text
	for _, want := range []string{"来源 brave", "b1", "b2", "https://b1.dev"} {
		if !strings.Contains(text, want) {
			t.Errorf("返回文本缺少 %q:\n%s", want, text)
		}
	}

	module.Shutdown() // 日志是异步写的，先排空

	logs, total, err := module.service.repo.listLogs(context.Background(), logFilter{})
	if err != nil {
		t.Fatalf("listLogs 返回错误: %v", err)
	}
	if total != 1 {
		t.Fatalf("日志条数 = %d", total)
	}
	// MCP 与统一接口共用一条搜索路径，只有 endpoint 能区分流量来自哪个入口
	if logs[0].Endpoint != "mcp/search" {
		t.Errorf("日志 endpoint = %q, 期望 mcp/search", logs[0].Endpoint)
	}
	if logs[0].Status != StatusOK || logs[0].Provider != "brave" {
		t.Errorf("日志内容异常: %+v", logs[0])
	}
}

func TestMCPToolCallCountsAgainstKeyQuota(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.MonthlyQuota = 1 })

	call := func() mcp.Result {
		body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"web_search",
			"arguments":{"query":"配额测试"}}}`
		return rpcResultAs[mcp.Result](t, doMCP(engine, token, body))
	}

	if first := call(); first.IsError {
		t.Fatalf("首次调用不应失败: %+v", first.Content)
	}
	second := call()
	if !second.IsError {
		t.Fatal("超出配额后仍然放行")
	}
	if !strings.Contains(second.Content[0].Text, "搜索失败") {
		t.Errorf("失败文本不易读: %q", second.Content[0].Text)
	}
}

func TestMCPToolCallUpstreamFailureIsToolError(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: failing(http.StatusInternalServerError, "boom")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	body := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"web_search","arguments":{"query":"go"}}}`
	response := decodeRPC(t, doMCP(engine, token, body))
	// 上游挂掉属于执行期失败，要放进 result.isError 让模型看得见，而不是变成协议错误
	if response.Error != nil {
		t.Fatalf("上游失败被报成了 JSON-RPC 错误: %+v", response.Error)
	}
	result := rpcResultAs[mcp.Result](t, doMCP(engine, token, body))
	if !result.IsError {
		t.Fatal("上游失败未标记 isError")
	}
}

func TestMCPProtocolErrors(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	tests := []struct {
		name string
		body string
		want int
	}{
		{
			name: "未知方法",
			body: `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`,
			want: mcp.MethodNotFound,
		},
		{
			name: "未知工具",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"delete_everything","arguments":{}}}`,
			want: mcp.InvalidParams,
		},
		{
			name: "批量请求",
			body: `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`,
			want: mcp.InvalidRequest,
		},
		{
			name: "缺少 method",
			body: `{"jsonrpc":"2.0","id":1}`,
			want: mcp.InvalidRequest,
		},
		{
			name: "body 不是 JSON",
			body: `not json`,
			want: mcp.ParseError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := doMCP(engine, token, test.body)
			// 协议错误也走 200 + JSON-RPC error，这是 JSON-RPC over HTTP 的惯例
			if recorder.Code != http.StatusOK {
				t.Fatalf("状态码 = %d, 期望 200", recorder.Code)
			}
			if got := rpcErrorCode(t, recorder); got != test.want {
				t.Errorf("错误码 = %d, 期望 %d", got, test.want)
			}
		})
	}
}

// TestMCPSemanticValidationFailsAsToolError covers arguments the tool can parse
// but must reject: the Tool interface only returns a Result, so semantic
// failures surface as isError=true text the model can read and correct, instead
// of a JSON-RPC error the client would just surface as a transport failure.
func TestMCPSemanticValidationFailsAsToolError(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "参数非法",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"web_search","arguments":{"query":"  "}}}`,
			want: "query 不能为空",
		},
		{
			name: "max_results 越界",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"web_search","arguments":{"query":"go","max_results":99}}}`,
			want: "max_results",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := doMCP(engine, token, test.body)
			if recorder.Code != http.StatusOK {
				t.Fatalf("状态码 = %d, 期望 200", recorder.Code)
			}
			response := decodeRPC(t, recorder)
			if response.Error != nil {
				t.Fatalf("语义校验失败不应成为 JSON-RPC 错误: %+v", response.Error)
			}
			result := rpcResultAs[mcp.Result](t, recorder)
			if !result.IsError {
				t.Fatal("语义校验失败未标记 isError")
			}
			if !strings.Contains(result.Content[0].Text, test.want) {
				t.Errorf("错误文本不易读: %q", result.Content[0].Text)
			}
		})
	}
}

func TestMCPPing(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	response := decodeRPC(t, doMCP(engine, token, `{"jsonrpc":"2.0","id":9,"method":"ping"}`))
	if response.Error != nil {
		t.Fatalf("ping 返回错误: %+v", response.Error)
	}
}

func TestMCPStreamVerbsAreRejected(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	// 网关不做服务端推送，按传输规范这两个动作要明确回 405，客户端才会退回纯 POST
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		recorder := doGateway(engine, method, "/api/gateway/v1/mcp", token, "")
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s 状态码 = %d, 期望 405", method, recorder.Code)
		}
	}
}

func TestRenderSearchResult(t *testing.T) {
	published := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	text := renderSearchResult(SearchResult{
		Provider: "tavily",
		Answer:   "答案在这里",
		Results: []search.Result{
			{Title: "标题一", URL: "https://a.dev", Content: "摘要\n带换行", PublishedAt: &published},
			{Title: "标题二", URL: "https://b.dev"},
		},
		Meta: SearchMeta{Cached: true, LatencyMS: 12, FallbackFrom: "brave"},
	})

	for _, want := range []string{
		"来源 tavily", "由 brave 回退", "缓存命中", "答案在这里",
		"1. 标题一", "https://a.dev", "发布于 2026-03-04", "摘要 带换行", "2. 标题二",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("渲染结果缺少 %q:\n%s", want, text)
		}
	}
}

func TestRenderSearchResultWithoutHits(t *testing.T) {
	text := renderSearchResult(SearchResult{Provider: "brave"})
	if !strings.Contains(text, "没有找到结果") {
		t.Errorf("空结果应说明，实际 = %q", text)
	}
}

// schemaPropertiesOf pulls the advertised parameter map out of a tool definition.
func schemaPropertiesOf(t *testing.T, tool mcp.Definition) map[string]any {
	t.Helper()
	schema, ok := tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("inputSchema 类型异常: %T", tool.InputSchema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("inputSchema.properties 缺失: %+v", schema)
	}
	return properties
}

func TestMCPToolAdvertisesBuiltInWebSearchParameterNames(t *testing.T) {
	// 目标是取代 Claude Code 内置的 WebSearch，所以它那套域名参数名必须认，
	// 否则模型照着旧习惯传 allowed_domains，会被静默忽略、搜索照跑但没过滤
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	}})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doMCP(engine, token, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	list := rpcResultAs[mcp.ToolListResult](t, recorder)
	properties := schemaPropertiesOf(t, list.Tools[0])

	for _, name := range []string{"allowed_domains", "blocked_domains", "include_domains", "exclude_domains"} {
		if _, ok := properties[name]; !ok {
			t.Errorf("schema 缺少参数 %q", name)
		}
	}
}

func TestMCPSearchAcceptsBuiltInDomainFilterNames(t *testing.T) {
	var received struct {
		IncludeDomains []string `json:"include_domains"`
		ExcludeDomains []string `json:"exclude_domains"`
	}
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":"x","results":[]}`))
	}})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"web_search",` +
		`"arguments":{"query":"go release","provider":"tavily",` +
		`"allowed_domains":["go.dev"],"blocked_domains":["example.com"]}}}`
	if recorder := doMCP(engine, token, body); recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d: %s", recorder.Code, recorder.Body.String())
	}

	if len(received.IncludeDomains) != 1 || received.IncludeDomains[0] != "go.dev" {
		t.Errorf("include_domains = %v, 期望 allowed_domains 被映射过去", received.IncludeDomains)
	}
	if len(received.ExcludeDomains) != 1 || received.ExcludeDomains[0] != "example.com" {
		t.Errorf("exclude_domains = %v, 期望 blocked_domains 被映射过去", received.ExcludeDomains)
	}
}

func TestMCPSearchStillAcceptsGatewayDomainFilterNames(t *testing.T) {
	var received struct {
		IncludeDomains []string `json:"include_domains"`
	}
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":"x","results":[]}`))
	}})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"web_search",` +
		`"arguments":{"query":"go release","provider":"tavily","include_domains":["pkg.go.dev"]}}}`
	if recorder := doMCP(engine, token, body); recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(received.IncludeDomains) != 1 || received.IncludeDomains[0] != "pkg.go.dev" {
		t.Errorf("include_domains = %v, 期望网关自己的参数名仍然可用", received.IncludeDomains)
	}
}

func TestMCPToolDescriptionPresentsItselfAsTheWebSearch(t *testing.T) {
	// 描述若把自己写成"博客自建网关的一个检索入口"，模型多半会绕开它去用内置搜索
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	}})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	list := rpcResultAs[mcp.ToolListResult](t,
		doMCP(engine, token, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	description := list.Tools[0].Description

	if !strings.Contains(description, "联网搜索") {
		t.Errorf("描述里没说明这是联网搜索工具: %s", description)
	}
	if strings.Contains(description, "博客") {
		t.Errorf("描述不该把自己限定成博客的附属功能: %s", description)
	}
}
