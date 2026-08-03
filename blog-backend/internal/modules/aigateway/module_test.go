package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// testCache is an in-memory stand-in for the application cache.
type testCache struct {
	mu    sync.RWMutex
	items map[string]any
}

func newTestCache() *testCache { return &testCache{items: make(map[string]any)} }

func (c *testCache) Set(key string, value any, _ ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
	return nil
}

func (c *testCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.items[key]
	return value, ok
}

func (c *testCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; !ok {
		return false
	}
	delete(c.items, key)
	return true
}

func (c *testCache) SetNx(key string, value any, duration ...time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; ok {
		return false
	}
	c.items[key] = value
	return true
}

func (c *testCache) Shutdown() {}

var _ dhcache.Cache = (*testCache)(nil)

// gatewayTestConfig describes the upstreams a test wants to stand up.
type gatewayTestConfig struct {
	Brave     http.HandlerFunc
	Tavily    http.HandlerFunc
	Exa       http.HandlerFunc
	Firecrawl http.HandlerFunc
	Options   *Options
}

func defaultTestOptions() Options {
	return Options{
		CacheTTL:         time.Minute,
		UpstreamTimeout:  5 * time.Second,
		QueueWait:        time.Second,
		LogRetentionDays: 90,
	}
}

// newGatewayTestModule builds a module backed by an in-memory database, with
// the requested upstreams replaced by local test servers.
func newGatewayTestModule(t *testing.T, config gatewayTestConfig) *Module {
	t.Helper()

	// A shared-cache named database, not ":memory:": the module's asynchronous
	// log writer runs on its own pooled connection, and a plain in-memory
	// SQLite database is private to a single connection.
	db, err := gorm.Open(sqlite.Open("file:"+testDBName(t)+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		if raw, err := db.DB(); err == nil {
			_ = raw.Close()
		}
	})
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}

	options := defaultTestOptions()
	if config.Options != nil {
		options = *config.Options
	}

	module, err := New(Dependencies{DB: db, Cache: newTestCache(), Options: options})
	if err != nil {
		t.Fatalf("构建网关模块失败: %v", err)
	}
	t.Cleanup(module.Shutdown)

	enable := func(name string, handler http.HandlerFunc) {
		if handler == nil {
			return
		}
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		updates := map[string]any{"enabled": true, "base_url": server.URL}
		if err := module.service.repo.updateProvider(context.Background(), name, updates); err != nil {
			t.Fatalf("启用供应商 %s 失败: %v", name, err)
		}
		addTestProviderKey(t, module, name, "test-key")
	}
	enable("brave", config.Brave)
	enable("tavily", config.Tavily)
	enable("exa", config.Exa)
	enable("firecrawl", config.Firecrawl)

	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("重新加载供应商失败: %v", err)
	}
	return module
}

// addTestProviderKey stores one upstream credential and returns its ID.
func addTestProviderKey(t *testing.T, module *Module, provider, secret string) int {
	t.Helper()
	key := ProviderKey{Provider: provider, Label: secret, APIKey: secret, Enabled: true, Status: ProviderKeyActive}
	if err := module.service.repo.createProviderKey(context.Background(), &key); err != nil {
		t.Fatalf("创建供应商密钥失败: %v", err)
	}
	return key.ID
}

// testDBName turns a (sub)test name into a database name unique to that test.
func testDBName(t *testing.T) string {
	return "gw_" + strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, t.Name())
}

// issueTestKey creates an enabled API key and returns its plaintext.
func issueTestKey(t *testing.T, module *Module, mutate func(*APIKey)) string {
	t.Helper()
	plain, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey 返回错误: %v", err)
	}
	key := APIKey{Name: "test", KeyPrefix: APIKeyPrefixOf(plain), KeyHash: HashAPIKey(plain), Enabled: true}
	if mutate != nil {
		mutate(&key)
	}
	if err := module.service.repo.createAPIKey(context.Background(), &key); err != nil {
		t.Fatalf("createAPIKey 返回错误: %v", err)
	}
	return plain
}

func newTestEngine(module *Module) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	routes := &router.Routes{Engine: engine, PublicAPI: engine.Group("/api"), AdminAPI: engine.Group("/api/admin")}
	module.RegisterRoutes(routes)
	return engine
}

func doGateway(engine *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func braveOK(results ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		items := make([]map[string]any, 0, len(results))
		for _, title := range results {
			items = append(items, map[string]any{"title": title, "url": "https://" + title + ".dev", "description": "d " + title})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"web": map[string]any{"results": items}})
	}
}

func tavilyOK(answer string, results ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		items := make([]map[string]any, 0, len(results))
		for _, title := range results {
			items = append(items, map[string]any{"title": title, "url": "https://" + title + ".dev", "content": "c " + title, "score": 0.5})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"query": "q", "answer": answer, "results": items, "usage": map[string]any{"credits": 1},
		})
	}
}

func failing(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func decodeSearch(t *testing.T, recorder *httptest.ResponseRecorder) SearchResult {
	t.Helper()
	var result SearchResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v (body=%s)", err, recorder.Body.String())
	}
	return result
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) errorDetail {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析错误响应失败: %v (body=%s)", err, recorder.Body.String())
	}
	return body.Error
}

func TestGatewayRejectsMissingAPIKey(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	engine := newTestEngine(module)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", "", `{"query":"go"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, 期望 401", recorder.Code)
	}
	if detail := decodeError(t, recorder); detail.Type != "invalid_api_key" {
		t.Fatalf("错误类型 = %q", detail.Type)
	}
}

func TestGatewaySearchReturnsBareJSON(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a", "b")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go 1.25","max_results":5}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	// The gateway contract is a bare object, not the blog's {code,msg,data}.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if _, wrapped := envelope["code"]; wrapped {
		t.Fatal("网关响应不应使用 {code,msg,data} 包裹")
	}

	result := decodeSearch(t, recorder)
	if result.Provider != "brave" {
		t.Errorf("Provider = %q", result.Provider)
	}
	if len(result.Results) != 2 {
		t.Fatalf("结果数 = %d", len(result.Results))
	}
	if result.Meta.RequestID == "" {
		t.Error("meta.request_id 不应为空")
	}
	if result.Meta.Cached {
		t.Error("首次请求不应命中缓存")
	}
}

func TestGatewaySearchViaGET(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodGet, "/api/gateway/v1/search?q=go&max_results=1", token, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decodeSearch(t, recorder); len(result.Results) != 1 {
		t.Fatalf("结果数 = %d", len(result.Results))
	}
}

func TestGatewayServesSecondRequestFromCache(t *testing.T) {
	var upstreamCalls int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamCalls, 1)
		braveOK("a")(w, r)
	}
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: handler})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	for i := 0; i < 2; i++ {
		if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go"}`); recorder.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求状态码 = %d", i+1, recorder.Code)
		}
	}
	if calls := atomic.LoadInt32(&upstreamCalls); calls != 1 {
		t.Fatalf("上游调用次数 = %d, 期望第二次命中缓存", calls)
	}

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go"}`)
	result := decodeSearch(t, recorder)
	if !result.Meta.Cached {
		t.Error("meta.cached 应为 true")
	}
	if result.Meta.Credits != 0 {
		t.Errorf("缓存命中不应计费, credits = %d", result.Meta.Credits)
	}

	// no_cache forces a fresh upstream call.
	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go","no_cache":true}`); recorder.Code != http.StatusOK {
		t.Fatalf("no_cache 请求状态码 = %d", recorder.Code)
	}
	if calls := atomic.LoadInt32(&upstreamCalls); calls != 2 {
		t.Fatalf("上游调用次数 = %d, no_cache 应绕过缓存", calls)
	}
}

func TestGatewayFallsBackToSecondProvider(t *testing.T) {
	var tavilyCalls int32
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave: failing(http.StatusInternalServerError, "boom"),
		Tavily: func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&tavilyCalls, 1)
			tavilyOK("", "t1")(w, r)
		},
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	// Brave wins the tie on name ordering only when scores match; force it
	// first with an explicit request plus fallback.
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go","provider":"brave","allow_fallback":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	result := decodeSearch(t, recorder)
	if result.Provider != "tavily" {
		t.Fatalf("Provider = %q, 期望回退到 tavily", result.Provider)
	}
	if result.Meta.FallbackFrom != "brave" {
		t.Errorf("meta.fallback_from = %q, 期望 brave", result.Meta.FallbackFrom)
	}
	if atomic.LoadInt32(&tavilyCalls) != 1 {
		t.Error("备用供应商应被调用一次")
	}
}

func TestGatewayDoesNotFallBackWhenDisallowed(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  failing(http.StatusInternalServerError, "boom"),
		Tavily: tavilyOK("", "t1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go","provider":"brave","allow_fallback":false}`)
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("状态码 = %d, 期望 502", recorder.Code)
	}
	if detail := decodeError(t, recorder); detail.Type != "provider_error" || detail.Provider != "brave" {
		t.Fatalf("错误 = %+v", detail)
	}
}

func TestGatewayDoesNotFallBackOnNonRetryableError(t *testing.T) {
	var tavilyCalls int32
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave: failing(http.StatusBadRequest, `{"error":{"detail":"missing q"}}`),
		Tavily: func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&tavilyCalls, 1)
			tavilyOK("", "t1")(w, r)
		},
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go","provider":"brave","allow_fallback":true}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, 期望 400", recorder.Code)
	}
	if atomic.LoadInt32(&tavilyCalls) != 0 {
		t.Error("参数错误不应触发回退，否则只是再烧一份配额")
	}
}

func TestGatewayRoutesAnswerRequestToTavily(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  braveOK("a"),
		Tavily: tavilyOK("这是答案", "t1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go","include_answer":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	result := decodeSearch(t, recorder)
	if result.Provider != "tavily" {
		t.Fatalf("Provider = %q, 需要答案时必须路由到 tavily", result.Provider)
	}
	if result.Answer != "这是答案" {
		t.Errorf("Answer = %q", result.Answer)
	}
}

func TestGatewayRejectsProviderOutsideKeyWhitelist(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a"), Tavily: tavilyOK("", "t")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.AllowedProviders = "brave" })

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go","provider":"tavily"}`)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("状态码 = %d, 期望 403", recorder.Code)
	}
	if detail := decodeError(t, recorder); detail.Type != "provider_not_allowed" {
		t.Fatalf("错误类型 = %q", detail.Type)
	}
}

func TestGatewayEnforcesPerMinuteRateLimit(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.RateLimitPerMin = 1 })

	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"one"}`); recorder.Code != http.StatusOK {
		t.Fatalf("首次请求状态码 = %d", recorder.Code)
	}
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"two"}`)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("状态码 = %d, 期望 429", recorder.Code)
	}
	if detail := decodeError(t, recorder); detail.Type != "rate_limit_exceeded" {
		t.Fatalf("错误类型 = %q", detail.Type)
	}
}

func TestGatewayEnforcesMonthlyKeyQuota(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.MonthlyQuota = 1 })

	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"one"}`); recorder.Code != http.StatusOK {
		t.Fatalf("首次请求状态码 = %d", recorder.Code)
	}
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"two"}`)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("状态码 = %d, 期望 429", recorder.Code)
	}
}

func TestGatewaySkipsProviderWithExhaustedQuota(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  braveOK("a"),
		Tavily: tavilyOK("", "t"),
	})
	ctx := context.Background()
	if err := module.service.repo.updateProvider(ctx, "brave", map[string]any{"monthly_quota": 1}); err != nil {
		t.Fatalf("更新配额失败: %v", err)
	}
	if err := module.service.repo.addUsage(ctx, providerSubject("brave"), currentPeriod(time.Now()), 1, 1, 0); err != nil {
		t.Fatalf("写入用量失败: %v", err)
	}
	if err := module.service.Reload(ctx); err != nil {
		t.Fatalf("Reload 失败: %v", err)
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decodeSearch(t, recorder); result.Provider != "tavily" {
		t.Fatalf("Provider = %q, 配额耗尽的供应商应被跳过", result.Provider)
	}
}

func TestGatewayRecordsUsageAndLog(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a", "b")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go"}`); recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	// The log writer is asynchronous; drain it before asserting.
	module.Shutdown()

	ctx := context.Background()
	usage, err := module.service.repo.usageFor(ctx, currentPeriod(time.Now()), []string{providerSubject("brave")})
	if err != nil {
		t.Fatalf("usageFor 返回错误: %v", err)
	}
	if usage[providerSubject("brave")].Count != 1 {
		t.Fatalf("供应商用量 = %d, 期望 1", usage[providerSubject("brave")].Count)
	}

	logs, total, err := module.service.repo.listLogs(ctx, logFilter{})
	if err != nil {
		t.Fatalf("listLogs 返回错误: %v", err)
	}
	if total != 1 {
		t.Fatalf("日志条数 = %d, 期望 1", total)
	}
	entry := logs[0]
	if entry.Status != StatusOK || entry.Provider != "brave" || entry.ResultCount != 2 {
		t.Fatalf("日志内容异常: %+v", entry)
	}
	if entry.Query != "go" {
		t.Errorf("日志 query = %q", entry.Query)
	}
}

func TestGatewayValidatesRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"query 为空", `{"query":"   "}`},
		{"provider 非法", `{"query":"go","provider":"google"}`},
		{"max_results 越界", `{"query":"go","max_results":50}`},
		{"topic 非法", `{"query":"go","topic":"video"}`},
		{"freshness 非法", `{"query":"go","freshness":"yesterday"}`},
		{"country 长度错误", `{"query":"go","country":"china"}`},
	}

	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, test.body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("状态码 = %d, 期望 400", recorder.Code)
			}
			if detail := decodeError(t, recorder); detail.Type != "invalid_request" {
				t.Fatalf("错误类型 = %q", detail.Type)
			}
		})
	}
}

func TestGatewayReportsNoProviderWhenAllDisabled(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go"}`)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d, 期望 503", recorder.Code)
	}
	if detail := decodeError(t, recorder); detail.Type != "no_provider_available" {
		t.Fatalf("错误类型 = %q", detail.Type)
	}
}

func TestGatewayProvidersEndpoint(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a"), Tavily: tavilyOK("", "t")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.AllowedProviders = "tavily" })

	recorder := doGateway(engine, http.MethodGet, "/api/gateway/v1/providers", token, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}

	var body struct {
		Providers []providerStatus `json:"providers"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(body.Providers) != 1 || body.Providers[0].Name != "tavily" {
		t.Fatalf("providers = %+v, 应受 Key 白名单约束", body.Providers)
	}
	if !body.Providers[0].SupportsAnswer {
		t.Error("tavily 应声明支持答案")
	}
}

func TestGatewayEndpointsDisabledByConfig(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	module.SetEnabled(false)
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"go"}`); recorder.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, 关闭网关后不应注册对外路由", recorder.Code)
	}
	// The admin surface stays up so the gateway can still be configured.
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/gateway/providers", nil))
	if recorder.Code == http.StatusNotFound {
		t.Fatal("管理接口应始终可用")
	}
}

func TestAdminProviderUpdateLeavesCredentialsAlone(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	engine := newTestEngine(module)

	// 密钥归 /keys 端点管，供应商补丁即便带上 apiKey 也不该动到凭据
	request := httptest.NewRequest(http.MethodPut, "/api/admin/gateway/providers/brave",
		strings.NewReader(`{"apiKey":"","weight":3}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	keys, err := module.service.repo.listProviderKeys(context.Background())
	if err != nil {
		t.Fatalf("listProviderKeys 返回错误: %v", err)
	}
	if len(keys) != 1 || keys[0].APIKey != "test-key" {
		t.Fatalf("密钥被供应商补丁改动了: %+v", keys)
	}

	provider, err := module.service.repo.providerByName(context.Background(), "brave")
	if err != nil {
		t.Fatalf("providerByName 返回错误: %v", err)
	}
	if provider.Weight != 3 {
		t.Fatalf("Weight = %d, 期望 3", provider.Weight)
	}
}

func TestAdminProviderListMasksCredential(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	views, err := module.service.providerViews(context.Background())
	if err != nil {
		t.Fatalf("providerViews 返回错误: %v", err)
	}
	seen := 0
	for _, view := range views {
		for _, key := range view.Keys {
			seen++
			if strings.Contains(key.Masked, "test-key") {
				t.Fatalf("管理接口不应返回明文密钥: %q", key.Masked)
			}
		}
	}
	if seen == 0 {
		t.Fatal("没有任何密钥被展示，用例失去意义")
	}
}

func TestAdminCreateKeyReturnsPlaintextOnce(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	request := httptest.NewRequest(http.MethodPost, "/api/admin/gateway/keys",
		strings.NewReader(`{"name":"agent","rateLimitPerMin":60}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var created struct {
		Data struct {
			ID     int    `json:"id"`
			APIKey string `json:"apiKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !strings.HasPrefix(created.Data.APIKey, apiKeyLiteral) {
		t.Fatalf("返回的 Key = %q", created.Data.APIKey)
	}

	// The plaintext must not be recoverable afterwards.
	views, err := module.service.apiKeyViews(context.Background())
	if err != nil {
		t.Fatalf("apiKeyViews 返回错误: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("Key 数量 = %d", len(views))
	}
	body, _ := json.Marshal(views)
	if strings.Contains(string(body), created.Data.APIKey) {
		t.Fatal("列表接口不应再返回明文 Key")
	}

	// ...and it must actually authenticate.
	if _, err := module.service.authenticate(context.Background(), created.Data.APIKey); err != nil {
		t.Fatalf("新建的 Key 应可用: %v", err)
	}
}
