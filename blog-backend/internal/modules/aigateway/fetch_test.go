package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"dh-blog/internal/platform/mcp"
	"dh-blog/internal/platform/search"
)

// firecrawlScrape serves /scrape with markdown for every URL and counts the
// calls; the credit endpoint gets an empty allowance as in firecrawlOK.
func firecrawlScrape(markdown string, statusCode int, calls *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/team/credit-usage") {
			_, _ = w.Write([]byte(`{"success":true,"data":{"remainingCredits":100}}`))
			return
		}
		if calls != nil {
			calls.Add(1)
		}
		var body struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"markdown": markdown,
				"metadata": map[string]any{"title": "页面标题", "url": body.URL + "#final", "statusCode": statusCode},
			},
		})
	}
}

func decodeFetch(t *testing.T, body []byte) FetchResult {
	t.Helper()
	var result FetchResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("解析抓取结果失败: %v (body=%s)", err, body)
	}
	return result
}

func callWebFetch(t *testing.T, module *Module, token, arguments string) mcp.Result {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"web_fetch","arguments":` + arguments + `}}`
	return rpcResultAs[mcp.Result](t, doMCP(newTestEngine(module), token, body))
}

func TestFetchEndpointReadsPageAndBills(t *testing.T) {
	var calls atomic.Int32
	module := newGatewayTestModule(t, gatewayTestConfig{Firecrawl: firecrawlScrape("# 正文\n\n你好", 200, &calls)})
	token := issueTestKey(t, module, nil)

	recorder := doGateway(newTestEngine(module), http.MethodPost, "/api/gateway/v1/fetch", token,
		`{"url":"https://example.com/post#section"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	result := decodeFetch(t, recorder.Body.Bytes())
	// 片段在发给上游前去掉；最终地址以上游报告的为准
	if result.Provider != "firecrawl" || result.Content != "# 正文\n\n你好" || result.URL != "https://example.com/post#final" ||
		result.Title != "页面标题" || result.TotalLength != 8 || result.NextStartIndex != 0 || result.Meta.Credits != 1 {
		t.Fatalf("result = %+v", result)
	}

	// GET 形式同样可用，且命中缓存不再计费
	recorder = doGateway(newTestEngine(module), http.MethodGet, "/api/gateway/v1/fetch?url=https://example.com/post", token, "")
	if recorder.Code != http.StatusOK || !decodeFetch(t, recorder.Body.Bytes()).Meta.Cached {
		t.Fatalf("GET 状态码 = %d, body=%s, 期望缓存命中", recorder.Code, recorder.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("上游调用次数 = %d, 期望 1", calls.Load())
	}

	ctx := context.Background()
	usage, err := module.service.repo.usageFor(ctx, currentPeriod(module.service.now()),
		[]string{providerSubject("firecrawl")})
	if err != nil {
		t.Fatalf("读取用量失败: %v", err)
	}
	if got := usage[providerSubject("firecrawl")]; got.Count != 1 || got.Credits != 1 {
		t.Errorf("firecrawl 用量 = %+v, 期望 1 次 1 credit", got)
	}

	module.Shutdown()
	logs, _, err := module.service.repo.listLogs(ctx, logFilter{})
	if err != nil {
		t.Fatalf("listLogs 返回错误: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("日志条数 = %d, 期望 2", len(logs))
	}
	for _, entry := range logs {
		if entry.Endpoint != "fetch" || entry.Provider != "firecrawl" || entry.Status != StatusOK ||
			!strings.HasPrefix(entry.Query, "https://example.com/post") {
			t.Errorf("日志 = %+v", entry)
		}
	}
}

func TestWebFetchToolReadsInPiecesFromCache(t *testing.T) {
	var calls atomic.Int32
	module := newGatewayTestModule(t, gatewayTestConfig{Firecrawl: firecrawlScrape("0123456789abcdefghij", 200, &calls)})
	token := issueTestKey(t, module, nil)

	first := callWebFetch(t, module, token, `{"url":"https://example.com/long","max_length":8}`)
	if first.IsError {
		t.Fatalf("工具调用失败: %+v", first.Content)
	}
	text := first.Content[0].Text
	for _, want := range []string{"来源 firecrawl", "标题：页面标题", "第 0–8 字符，共 20 字符", "01234567", "start_index 设为 8"} {
		if !strings.Contains(text, want) {
			t.Errorf("第一段缺少 %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "89abc") {
		t.Errorf("第一段不该超出 max_length:\n%s", text)
	}

	second := callWebFetch(t, module, token, `{"url":"https://example.com/long","max_length":8,"start_index":16}`)
	text = second.Content[0].Text
	if !strings.Contains(text, "ghij") || !strings.Contains(text, "缓存命中") || strings.Contains(text, "未完") {
		t.Errorf("最后一段 = \n%s", text)
	}
	if calls.Load() != 1 {
		t.Fatalf("上游调用次数 = %d, 后续分段应走缓存", calls.Load())
	}

	module.Shutdown()
	logs, _, _ := module.service.repo.listLogs(context.Background(), logFilter{})
	if len(logs) == 0 || logs[0].Endpoint != "mcp/fetch" {
		t.Errorf("日志 = %+v, 期望 endpoint 为 mcp/fetch", logs)
	}
}

func TestWebFetchWarnsAboutTargetErrorPages(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Firecrawl: firecrawlScrape("Not Found", 404, nil)})
	result := callWebFetch(t, module, issueTestKey(t, module, nil), `{"url":"https://example.com/missing"}`)
	if result.IsError || !strings.Contains(result.Content[0].Text, "HTTP 404") {
		t.Fatalf("result = %+v, 目标页 404 时应照常返回并提醒", result)
	}
}

func TestFetchRejectsUnsafeURLsBeforeCallingUpstream(t *testing.T) {
	var calls atomic.Int32
	module := newGatewayTestModule(t, gatewayTestConfig{Firecrawl: firecrawlScrape("x", 200, &calls)})
	token := issueTestKey(t, module, nil)

	for _, target := range []string{
		"", "ftp://example.com/file", "example.com", "http://localhost:8080/admin", "http://app.localhost/",
		"http://127.0.0.1/", "http://192.168.1.1/", "http://10.0.0.8/", "http://[::1]/", "http://169.254.169.254/latest/meta-data",
		"https://user:pass@example.com/", "http://0.0.0.0/",
	} {
		body, _ := json.Marshal(map[string]string{"url": target})
		recorder := doGateway(newTestEngine(module), http.MethodPost, "/api/gateway/v1/fetch", token, string(body))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%q: 状态码 = %d, 期望 400 (body=%s)", target, recorder.Code, recorder.Body.String())
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("上游调用次数 = %d, 非法地址不该发出去", calls.Load())
	}

	result := callWebFetch(t, module, token, `{"url":"http://127.0.0.1/"}`)
	if !result.IsError {
		t.Error("MCP 也应拒绝内网地址")
	}
	result = callWebFetch(t, module, token, `{"url":"https://example.com/","max_length":1000000}`)
	if !result.IsError {
		t.Error("MCP 的 max_length 不能超过上限")
	}
}

func TestFetchWithoutFetchProvider(t *testing.T) {
	// 只有 brave：能搜不能抓
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1")})
	token := issueTestKey(t, module, nil)

	recorder := doGateway(newTestEngine(module), http.MethodPost, "/api/gateway/v1/fetch", token, `{"url":"https://example.com/"}`)
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(decodeError(t, recorder).Message, "Firecrawl") {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	// 工具定义要如实说明现在用不了
	list := rpcResultAs[mcp.ToolListResult](t, doMCP(newTestEngine(module), token, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	for _, tool := range list.Tools {
		if tool.Name == mcpToolWebFetch && !strings.Contains(tool.Description, "没有可用的网页抓取供应商") {
			t.Errorf("web_fetch 说明 = %q", tool.Description)
		}
	}
}

func TestFetchHonoursKeyProviderWhitelist(t *testing.T) {
	var calls atomic.Int32
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1"), Firecrawl: firecrawlScrape("x", 200, &calls)})
	token := issueTestKey(t, module, func(key *APIKey) { key.AllowedProviders = "brave" })

	recorder := doGateway(newTestEngine(module), http.MethodPost, "/api/gateway/v1/fetch", token, `{"url":"https://example.com/"}`)
	if recorder.Code != http.StatusServiceUnavailable || calls.Load() != 0 {
		t.Fatalf("状态码 = %d, 调用次数 = %d; 白名单不含 firecrawl 时不能抓取", recorder.Code, calls.Load())
	}
}

func TestFetchPageFailuresDoNotTripBreaker(t *testing.T) {
	var calls atomic.Int32
	module := newGatewayTestModule(t, gatewayTestConfig{Firecrawl: func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/team/credit-usage") {
			_, _ = w.Write([]byte(`{"success":true,"data":{"remainingCredits":100}}`))
			return
		}
		calls.Add(1)
		_, _ = w.Write([]byte(`{"success":false,"error":"All scraping engines failed"}`))
	}})
	token := issueTestKey(t, module, func(key *APIKey) { key.RateLimitPerMin = 0 })

	for i := 0; i < 8; i++ {
		body, _ := json.Marshal(map[string]string{"url": "https://dead.example/" + string(rune('a'+i))})
		recorder := doGateway(newTestEngine(module), http.MethodPost, "/api/gateway/v1/fetch", token, string(body))
		if recorder.Code == http.StatusOK {
			t.Fatalf("抓取失败的页面不应返回 200")
		}
	}
	// 一连串死链说明的是页面，不是 Firecrawl：熔断器必须保持关闭，否则它的搜索也会被一起停掉
	if state := module.service.runtime("firecrawl").breaker.State(); state != search.BreakerClosed {
		t.Fatalf("熔断器状态 = %v, 期望保持关闭", state)
	}
	if calls.Load() != 8 {
		t.Fatalf("上游调用次数 = %d, 期望每次都真的发出去", calls.Load())
	}
}

func TestProvidersReportFetchSupport(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("b1"), Firecrawl: firecrawlScrape("x", 200, nil)})
	statuses, err := module.service.ProviderStatuses(context.Background(), nil)
	if err != nil {
		t.Fatalf("ProviderStatuses 返回错误: %v", err)
	}
	for _, status := range statuses {
		if want := status.Name == "firecrawl"; status.SupportsFetch != want {
			t.Errorf("%s supports_fetch = %v, 期望 %v", status.Name, status.SupportsFetch, want)
		}
	}
}
