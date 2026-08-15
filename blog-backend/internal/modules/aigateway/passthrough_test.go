package aigateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"dh-blog/internal/platform/search"
)

// echoUpstream returns a fixed body and records what the upstream received.
func echoUpstream(body string, seen *atomic.Value, calls *int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			atomic.AddInt32(calls, 1)
		}
		if seen != nil {
			payload, _ := io.ReadAll(r.Body)
			seen.Store(map[string]string{"path": r.URL.Path, "query": r.URL.RawQuery, "body": string(payload)})
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func lastSeen(t *testing.T, seen *atomic.Value) map[string]string {
	t.Helper()
	value, ok := seen.Load().(map[string]string)
	if !ok {
		t.Fatal("上游未收到任何请求")
	}
	return value
}

func TestTavilyPassthroughReturnsNativeShape(t *testing.T) {
	const native = `{"query":"go","answer":"native answer","results":[{"title":"A","url":"https://a.dev","content":"c","score":0.9}],"response_time":1.2,"usage":{"credits":1}}`

	var seen atomic.Value
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: echoUpstream(native, &seen, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token,
		`{"query":"go","search_depth":"basic","include_answer":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	// 透传的价值在于 SDK 不用改代码，所以响应体必须与上游逐字节一致
	if recorder.Body.String() != native {
		t.Fatalf("响应体被改写:\n得到 %s\n期望 %s", recorder.Body.String(), native)
	}
	// 明确不是统一格式：不应出现网关自己的 meta 字段
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if _, wrapped := envelope["meta"]; wrapped {
		t.Error("原生透传不应附加网关的 meta 字段")
	}
	if got := recorder.Header().Get("X-Gateway-Provider"); got != "tavily" {
		t.Errorf("X-Gateway-Provider = %q", got)
	}

	upstream := lastSeen(t, &seen)
	if upstream["path"] != "/search" {
		t.Errorf("上游路径 = %q", upstream["path"])
	}
	if !strings.Contains(upstream["body"], `"search_depth":"basic"`) {
		t.Errorf("上游请求体应保留原生字段: %s", upstream["body"])
	}
}

func TestTavilyPassthroughStripsCallerCredential(t *testing.T) {
	var seen atomic.Value
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: echoUpstream(`{"results":[]}`, &seen, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token,
		`{"query":"go","api_key":"tvly-CALLER-SECRET"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}

	upstream := lastSeen(t, &seen)
	if strings.Contains(upstream["body"], "CALLER-SECRET") {
		t.Fatalf("调用方自带的凭证不应被转发: %s", upstream["body"])
	}
	if !strings.Contains(upstream["body"], `"query":"go"`) {
		t.Errorf("其余字段应保留: %s", upstream["body"])
	}
}

func TestBravePassthroughForwardsAllowlistedParamsOnly(t *testing.T) {
	var seen atomic.Value
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: echoUpstream(`{"web":{"results":[]}}`, &seen, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	path := "/api/gateway/v1/brave/web/search?q=rust&count=5&extra_snippets=true" +
		"&unexpected=1&X-Subscription-Token=leak"
	recorder := doGateway(engine, http.MethodGet, path, token, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	upstream := lastSeen(t, &seen)
	forwarded, err := url.ParseQuery(upstream["query"])
	if err != nil {
		t.Fatalf("解析上游 query 失败: %v", err)
	}
	if forwarded.Get("q") != "rust" || forwarded.Get("count") != "5" || forwarded.Get("extra_snippets") != "true" {
		t.Errorf("允许的参数未透传: %v", forwarded)
	}
	if forwarded.Has("unexpected") || forwarded.Has("X-Subscription-Token") {
		t.Errorf("白名单外的参数应被丢弃: %v", forwarded)
	}
	if upstream["path"] != "/web/search" {
		t.Errorf("上游路径 = %q", upstream["path"])
	}
}

func TestPassthroughRequiresAPIKey(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: echoUpstream(`{}`, nil, nil)})
	engine := newTestEngine(module)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", "", `{"query":"go"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, 期望 401", recorder.Code)
	}
}

func TestPassthroughHonoursKeyWhitelist(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  echoUpstream(`{"web":{"results":[]}}`, nil, nil),
		Tavily: echoUpstream(`{"results":[]}`, nil, nil),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.AllowedProviders = "brave" })

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token, `{"query":"go"}`)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("状态码 = %d, 期望 403", recorder.Code)
	}
	if detail := decodeError(t, recorder); detail.Type != "provider_not_allowed" {
		t.Fatalf("错误类型 = %q", detail.Type)
	}
}

func TestPassthroughRejectsDisabledProvider(t *testing.T) {
	// 只启用 brave，tavily 仍是默认的未启用状态
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: echoUpstream(`{"web":{"results":[]}}`, nil, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token, `{"query":"go"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, 期望 404", recorder.Code)
	}
	if detail := decodeError(t, recorder); detail.Type != "provider_not_found" {
		t.Fatalf("错误类型 = %q", detail.Type)
	}
}

func TestPassthroughNeverFallsBackToAnotherProvider(t *testing.T) {
	var tavilyCalls int32
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave: failing(http.StatusInternalServerError, `{"error":{"detail":"boom"}}`),
		Tavily: func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&tavilyCalls, 1)
			echoUpstream(`{"results":[]}`, nil, nil)(w, r)
		},
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodGet, "/api/gateway/v1/brave/web/search?q=go", token, "")

	// 上游的错误状态与错误体都原样返回，供 SDK 自行解析
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("状态码 = %d, 期望原样返回上游的 500", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "boom") {
		t.Errorf("上游错误体应原样返回: %s", recorder.Body.String())
	}
	// 换一家的响应结构就不再是调用方要的 schema，所以透传绝不回退
	if atomic.LoadInt32(&tavilyCalls) != 0 {
		t.Error("原生透传不应回退到其他供应商")
	}
}

func TestPassthroughServesSecondRequestFromCache(t *testing.T) {
	var calls int32
	module := newGatewayTestModule(t, gatewayTestConfig{
		Tavily: echoUpstream(`{"query":"go","results":[],"usage":{"credits":1}}`, nil, &calls),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	for i := 0; i < 2; i++ {
		if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token,
			`{"query":"go","max_results":5}`); recorder.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求状态码 = %d", i+1, recorder.Code)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("上游调用次数 = %d, 期望第二次命中缓存", got)
	}

	// 字段顺序不同但语义相同的请求体应命中同一个缓存条目
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token,
		`{"max_results":5,"query":"go"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("上游调用次数 = %d, 字段顺序不同不应造成缓存未命中", got)
	}
	if recorder.Header().Get("X-Gateway-Cached") != "1" {
		t.Error("缓存命中应带 X-Gateway-Cached 头")
	}
}

func TestPassthroughDoesNotCacheUpstreamFailure(t *testing.T) {
	var calls int32
	module := newGatewayTestModule(t, gatewayTestConfig{
		Tavily: func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":{"error":"bad"}}`))
		},
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	for i := 0; i < 2; i++ {
		doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token, `{"query":"go"}`)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("上游调用次数 = %d, 失败响应不应进缓存", got)
	}
}

func TestPassthroughAccountsUsageAndLogsEndpoint(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Tavily: echoUpstream(`{"query":"go","results":[],"usage":{"credits":2}}`, nil, nil),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token,
		`{"query":"go"}`); recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	module.Shutdown() // 日志是异步写的，先排空

	ctx := context.Background()
	usage, err := module.service.repo.usageFor(ctx, currentPeriod(time.Now()), []string{providerSubject("tavily")})
	if err != nil {
		t.Fatalf("usageFor 返回错误: %v", err)
	}
	if got := usage[providerSubject("tavily")]; got.Count != 1 || got.Credits != 2 {
		t.Fatalf("用量 = %+v, 期望 1 次 / 2 额度", got)
	}

	logs, total, err := module.service.repo.listLogs(ctx, logFilter{})
	if err != nil {
		t.Fatalf("listLogs 返回错误: %v", err)
	}
	if total != 1 {
		t.Fatalf("日志条数 = %d", total)
	}
	if logs[0].Endpoint != "tavily/search" {
		t.Errorf("日志 endpoint = %q, 应能与统一接口区分", logs[0].Endpoint)
	}
	if logs[0].Status != StatusOK || logs[0].Provider != "tavily" || logs[0].Credits != 2 {
		t.Errorf("日志内容异常: %+v", logs[0])
	}
}

func TestPassthroughEnforcesRateLimitAndQuota(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*APIKey)
	}{
		{"每分钟限速", func(key *APIKey) { key.RateLimitPerMin = 1 }},
		{"月配额", func(key *APIKey) { key.MonthlyQuota = 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			module := newGatewayTestModule(t, gatewayTestConfig{
				Tavily: echoUpstream(`{"results":[],"usage":{"credits":1}}`, nil, nil),
			})
			engine := newTestEngine(module)
			token := issueTestKey(t, module, test.mutate)

			if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token,
				`{"query":"one"}`); recorder.Code != http.StatusOK {
				t.Fatalf("首次请求状态码 = %d", recorder.Code)
			}
			// 换一个查询，避免命中缓存导致不消耗额度
			recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token, `{"query":"two"}`)
			if recorder.Code != http.StatusTooManyRequests {
				t.Fatalf("状态码 = %d, 期望 429", recorder.Code)
			}
		})
	}
}

func TestPassthroughSkipsProviderWithExhaustedQuota(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Tavily: echoUpstream(`{"results":[]}`, nil, nil),
	})
	ctx := context.Background()
	if err := module.service.repo.updateProvider(ctx, "tavily", map[string]any{"monthly_quota": 1}); err != nil {
		t.Fatalf("更新配额失败: %v", err)
	}
	if err := module.service.repo.addUsage(ctx, providerSubject("tavily"), currentPeriod(time.Now()), 1, 1, 0); err != nil {
		t.Fatalf("写入用量失败: %v", err)
	}
	if err := module.service.Reload(ctx); err != nil {
		t.Fatalf("Reload 失败: %v", err)
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token, `{"query":"go"}`)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d, 期望 503", recorder.Code)
	}
}

func TestPassthroughRejectsOversizedBody(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: echoUpstream(`{}`, nil, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	oversized := `{"query":"` + strings.Repeat("x", maxPassthroughRequestBody) + `"}`
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/tavily/search", token, oversized)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码 = %d, 期望 413", recorder.Code)
	}
}

func TestPassthroughRoutesDisabledWithGateway(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: echoUpstream(`{}`, nil, nil)})
	module.SetEnabled(false)
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	for _, target := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/gateway/v1/tavily/search"},
		{http.MethodGet, "/api/gateway/v1/brave/web/search?q=go"},
	} {
		recorder := doGateway(engine, target.method, target.path, token, `{"query":"go"}`)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s %s 状态码 = %d, 关闭网关后不应注册", target.method, target.path, recorder.Code)
		}
	}
}

func TestSanitizeTavilyBodyLeavesNonObjectUntouched(t *testing.T) {
	// 非法或非对象的请求体交给上游去拒绝，网关不擅自改写
	for _, body := range []string{`[1,2,3]`, `not json`, ``} {
		if got := string(sanitizeTavilyBody([]byte(body))); got != body {
			t.Errorf("sanitizeTavilyBody(%q) = %q, 应原样返回", body, got)
		}
	}
}

func TestPassthroughCacheKeySeparatesProvidersAndRoutes(t *testing.T) {
	key := func(provider, path, body string) string {
		return passthroughCacheKey(provider, search.PassthroughRequest{
			Method: http.MethodPost, Path: path, Body: []byte(body),
		})
	}
	if key("brave", "/web/search", "") == key("tavily", "/web/search", "") {
		t.Error("不同供应商必须使用不同的缓存键")
	}
	if key("tavily", "/search", `{"a":1}`) == key("tavily", "/search", `{"a":2}`) {
		t.Error("不同请求体必须使用不同的缓存键")
	}
	if once := key("tavily", "/search", `{"a":1}`); once != key("tavily", "/search", `{"a":1}`) {
		t.Error("相同请求必须命中同一个缓存键")
	}
}
