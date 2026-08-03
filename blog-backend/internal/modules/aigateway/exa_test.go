package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func exaOK(cost float64, titles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		items := make([]map[string]any, 0, len(titles))
		for _, title := range titles {
			items = append(items, map[string]any{
				"title": title, "url": "https://" + title + ".dev",
				"highlights": []string{"片段 " + title},
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"requestId": "exa-1", "results": items,
			"costDollars": map[string]any{"total": cost},
		})
	}
}

func TestGatewayRoutesToExaWhenSelected(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave: braveOK("b"),
		Exa:   exaOK(0.007, "e1", "e2"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"语义检索","provider":"exa","max_results":2}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	result := decodeSearch(t, recorder)
	if result.Provider != "exa" {
		t.Fatalf("Provider = %q", result.Provider)
	}
	if len(result.Results) != 2 || result.Results[0].Content != "片段 e1" {
		t.Fatalf("结果异常: %+v", result.Results)
	}
	// Exa 按美元计价，统一响应里以微美元暴露
	if result.Meta.CostMicroUSD != 7000 {
		t.Errorf("meta.cost_micro_usd = %d, 期望 7000", result.Meta.CostMicroUSD)
	}
	if result.Meta.Credits != 1 {
		t.Errorf("meta.credits = %d", result.Meta.Credits)
	}
}

func TestGatewayKeepsAnswerRequestsAwayFromExa(t *testing.T) {
	// Exa 的普通搜索不产出答案，能力过滤必须把它排除
	module := newGatewayTestModule(t, gatewayTestConfig{
		Exa:    exaOK(0.007, "e1"),
		Tavily: tavilyOK("这是答案", "t1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"什么是 Zettelkasten","include_answer":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decodeSearch(t, recorder); result.Provider != "tavily" {
		t.Fatalf("Provider = %q, 需要答案时不应路由到 exa", result.Provider)
	}
}

func TestGatewayPrefersBraveForOperatorQueriesOverExa(t *testing.T) {
	// Exa 明确要求用 includeDomains 而不是 site: 操作符
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave: braveOK("b1"),
		Exa:   exaOK(0.007, "e1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"site:github.com rust 教程"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decodeSearch(t, recorder); result.Provider != "brave" {
		t.Fatalf("Provider = %q, 含操作符的查询应偏向 brave", result.Provider)
	}
}

func TestGatewayFallsBackFromExaToOtherProvider(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Exa:    failing(http.StatusInternalServerError, `{"error":"boom"}`),
		Tavily: tavilyOK("", "t1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go","provider":"exa","allow_fallback":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	result := decodeSearch(t, recorder)
	if result.Provider != "tavily" || result.Meta.FallbackFrom != "exa" {
		t.Fatalf("回退异常: provider=%q fallbackFrom=%q", result.Provider, result.Meta.FallbackFrom)
	}
}

func TestExaPassthroughReturnsNativeShape(t *testing.T) {
	const native = `{"requestId":"exa-native","results":[{"title":"A","url":"https://a.dev","id":"https://a.dev","favicon":"https://a.dev/favicon.ico"}],"costDollars":{"total":0.005}}`

	var seen atomic.Value
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: echoUpstream(native, &seen, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/exa/search", token,
		`{"query":"go","type":"fast","contents":{"highlights":true}}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != native {
		t.Fatalf("响应体被改写: %s", recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Gateway-Provider"); got != "exa" {
		t.Errorf("X-Gateway-Provider = %q", got)
	}

	upstream := lastSeen(t, &seen)
	if upstream["path"] != "/search" {
		t.Errorf("上游路径 = %q", upstream["path"])
	}
	if !strings.Contains(upstream["body"], `"type":"fast"`) {
		t.Errorf("原生字段应保留: %s", upstream["body"])
	}
}

func TestExaPassthroughStripsCallerCredential(t *testing.T) {
	var seen atomic.Value
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: echoUpstream(`{"results":[]}`, &seen, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/exa/search", token,
		`{"query":"go","apiKey":"exa-CALLER-SECRET","x-api-key":"another-SECRET"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	if body := lastSeen(t, &seen)["body"]; strings.Contains(body, "SECRET") {
		t.Fatalf("调用方自带的凭证不应被转发: %s", body)
	}
}

func TestExaPassthroughAccountsCost(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Exa: echoUpstream(`{"results":[],"costDollars":{"total":0.0125}}`, nil, nil),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/exa/search", token,
		`{"query":"go"}`); recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	module.Shutdown()

	ctx := context.Background()
	usage, err := module.service.repo.usageFor(ctx, currentPeriod(time.Now()), []string{providerSubject("exa")})
	if err != nil {
		t.Fatalf("usageFor 返回错误: %v", err)
	}
	if got := usage[providerSubject("exa")]; got.Count != 1 || got.CostMicroUSD != 12500 {
		t.Fatalf("用量 = %+v, 期望 1 次 / 12500 微美元", got)
	}

	logs, _, err := module.service.repo.listLogs(ctx, logFilter{})
	if err != nil {
		t.Fatalf("listLogs 返回错误: %v", err)
	}
	if logs[0].Endpoint != "exa/search" || logs[0].CostMicroUSD != 12500 {
		t.Fatalf("日志内容异常: %+v", logs[0])
	}
}

func TestProvidersEndpointExposesMetadata(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0.007, "e1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

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
	if len(body.Providers) != 1 {
		t.Fatalf("providers = %+v", body.Providers)
	}
	provider := body.Providers[0]
	if provider.Name != "exa" || provider.HomeURL == "" || provider.DocsURL == "" {
		t.Fatalf("元信息缺失: %+v", provider)
	}
	if provider.SupportsAnswer {
		t.Error("exa 不应声明支持答案")
	}
}

func TestAdminProviderViewCarriesMetadataAndCost(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0.007, "e1")})
	ctx := context.Background()
	if err := module.service.repo.addUsage(ctx, providerSubject("exa"), currentPeriod(time.Now()), 2, 2, 14000); err != nil {
		t.Fatalf("addUsage 返回错误: %v", err)
	}

	views, err := module.service.providerViews(ctx)
	if err != nil {
		t.Fatalf("providerViews 返回错误: %v", err)
	}

	byName := map[string]providerView{}
	for _, view := range views {
		byName[view.Name] = view
	}
	// 每家供应商都应有种子行
	for _, name := range []string{"brave", "tavily", "exa", "firecrawl"} {
		view, ok := byName[name]
		if !ok {
			t.Fatalf("缺少供应商 %s", name)
		}
		if view.HomeURL == "" || view.DocsURL == "" || view.ConsoleURL == "" || view.Billing == "" {
			t.Errorf("%s 的展示信息不完整: %+v", name, view)
		}
	}
	if got := byName["exa"].MonthlyCost; got != 14000 {
		t.Errorf("exa 本月花费 = %d, 期望 14000", got)
	}
	if got := byName["exa"].MonthlyUsed; got != 2 {
		t.Errorf("exa 本月调用 = %d, 期望 2", got)
	}
}
