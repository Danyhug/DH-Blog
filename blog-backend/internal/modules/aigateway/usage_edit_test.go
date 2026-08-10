package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"dh-blog/internal/platform/search"
)

// adminPut sends an admin patch and fails the test on an unexpected status.
func adminPut(t *testing.T, module *Module, path, body string, wantStatus int) string {
	t.Helper()
	engine := newTestEngine(module)
	recorder := doGateway(engine, http.MethodPut, path, "", body)
	if recorder.Code != wantStatus {
		t.Fatalf("PUT %s 状态码 = %d, 期望 %d, body=%s", path, recorder.Code, wantStatus, recorder.Body.String())
	}
	return recorder.Body.String()
}

func providerUsage(t *testing.T, module *Module, name string) Usage {
	t.Helper()
	subject := providerSubject(name)
	usage, err := module.service.repo.usageFor(context.Background(),
		currentPeriod(module.service.now()), []string{subject})
	if err != nil {
		t.Fatalf("读取用量失败: %v", err)
	}
	return usage[subject]
}

func TestAdminOverwritesLocalUsage(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0.007, "a")})

	// 先攒一点本地用量，再按官网真实账单改写
	if err := module.service.repo.addUsage(context.Background(), providerSubject(search.ProviderExa),
		currentPeriod(module.service.now()), 3, 3, 21_000); err != nil {
		t.Fatalf("attribute usage: %v", err)
	}

	adminPut(t, module, "/api/admin/gateway/providers/exa/usage",
		`{"count":120,"costMicroUsd":4560000}`, http.StatusOK)

	got := providerUsage(t, module, search.ProviderExa)
	if got.Count != 120 || got.CostMicroUSD != 4_560_000 {
		t.Fatalf("改写后用量 = %+v, 期望 count=120 cost=4560000", got)
	}
	// 没有出现在请求里的字段必须保持原值，而不是被清零
	if got.Credits != 3 {
		t.Fatalf("credits = %d, 未提交的计数器不应被改动", got.Credits)
	}
}

func TestAdminUsageEditIsAbsoluteNotAdditive(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0.007, "a")})
	if err := module.service.repo.addUsage(context.Background(), providerSubject(search.ProviderExa),
		currentPeriod(module.service.now()), 10, 0, 70_000); err != nil {
		t.Fatalf("attribute usage: %v", err)
	}

	adminPut(t, module, "/api/admin/gateway/providers/exa/usage", `{"count":2}`, http.StatusOK)

	if got := providerUsage(t, module, search.ProviderExa); got.Count != 2 {
		t.Fatalf("count = %d, 手动校准应是覆盖而不是累加", got.Count)
	}
}

func TestAdminUsageEditRejectsBadInput(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0.007, "a")})

	adminPut(t, module, "/api/admin/gateway/providers/exa/usage", `{"count":-1}`, http.StatusBadRequest)
	adminPut(t, module, "/api/admin/gateway/providers/exa/usage", `{"costMicroUsd":-5}`, http.StatusBadRequest)
	adminPut(t, module, "/api/admin/gateway/providers/nope/usage", `{"count":1}`, http.StatusNotFound)
}

func TestAdminStoresExaUsageCredentialAndMasksIt(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0.007, "a")})

	adminPut(t, module, "/api/admin/gateway/providers/exa",
		`{"usageServiceKey":"exa-service-key-1234567890","usageKeyId":"550e8400-uuid"}`, http.StatusOK)

	// 真实值要落进 Extra，适配器才拿得到
	provider, err := module.service.repo.providerByName(context.Background(), search.ProviderExa)
	if err != nil {
		t.Fatalf("providerByName: %v", err)
	}
	if got := extraString(provider.Extra, extraKeyUsageServiceKey); got != "exa-service-key-1234567890" {
		t.Fatalf("存储的服务密钥 = %q", got)
	}
	if got := extraString(provider.Extra, extraKeyUsageKeyID); got != "550e8400-uuid" {
		t.Fatalf("存储的 key UUID = %q", got)
	}
	// 已有的其它选项不能被这次合并覆盖掉
	if got := extraString(provider.Extra, "search_type"); got != "auto" {
		t.Fatalf("search_type = %q, 合并不应丢掉原有选项", got)
	}

	// 列表接口不能把服务密钥原样吐回去
	engine := newTestEngine(module)
	recorder := doGateway(engine, http.MethodGet, "/api/admin/gateway/providers", "", "")
	body := recorder.Body.String()
	if strings.Contains(body, "exa-service-key-1234567890") {
		t.Fatalf("管理列表泄露了服务密钥: %s", body)
	}
	if !strings.Contains(body, "550e8400-uuid") {
		t.Fatalf("管理列表应回显 key UUID: %s", body)
	}

	var payload struct {
		Data []struct {
			Name            string `json:"name"`
			UsageServiceKey string `json:"usageServiceKeyMasked"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, item := range payload.Data {
		if item.Name != search.ProviderExa {
			continue
		}
		if item.UsageServiceKey != "exa-****7890" {
			t.Fatalf("掩码 = %q, 期望首尾各留四位", item.UsageServiceKey)
		}
	}
}

func TestAdminClearsExaUsageCredential(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0.007, "a")})
	adminPut(t, module, "/api/admin/gateway/providers/exa",
		`{"usageServiceKey":"svc-key-abcdefgh","usageKeyId":"uuid"}`, http.StatusOK)
	adminPut(t, module, "/api/admin/gateway/providers/exa", `{"usageServiceKey":""}`, http.StatusOK)

	provider, err := module.service.repo.providerByName(context.Background(), search.ProviderExa)
	if err != nil {
		t.Fatalf("providerByName: %v", err)
	}
	if got := extraString(provider.Extra, extraKeyUsageServiceKey); got != "" {
		t.Fatalf("服务密钥 = %q, 传空串应清除", got)
	}
	// 只清了密钥，key UUID 不该跟着消失
	if got := extraString(provider.Extra, extraKeyUsageKeyID); got != "uuid" {
		t.Fatalf("key UUID = %q, 不该被一起清掉", got)
	}
}

func TestRoutingPrefersUpstreamSpendOverLocalCounter(t *testing.T) {
	// 本地计数只看得到网关自己发出的请求。上游报出的花费更权威，
	// 到顶后即使本地还是 0 也不该再放行。
	var calls int
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"t","url":"https://example.com"}],"costDollars":{"total":0}}`))
	}})
	if err := module.service.repo.updateProvider(context.Background(), search.ProviderExa,
		map[string]any{"monthly_cost_limit": 10_000_000}); err != nil {
		t.Fatalf("设置费用上限失败: %v", err)
	}
	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}

	runtime := module.service.runtime(search.ProviderExa)
	credentials := runtime.credentials()
	if len(credentials) == 0 {
		t.Fatal("测试模块没有可用的 Exa 凭据")
	}
	// 本地一次都没记，上游说已经花掉 $12，超过 $10 的上限
	runtime.recordUsage(credentials[0].id, search.UsageReport{
		Used: 12_000_000, Unit: search.UsageUnitMicroUSD, Scope: search.UsageScopeKey,
	}, module.service.now())

	if local := providerUsage(t, module, search.ProviderExa); local.CostMicroUSD != 0 {
		t.Fatalf("前置条件不成立：本地费用 = %d, 期望 0", local.CostMicroUSD)
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"probe","provider":"exa","no_cache":true}`)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d, 上游费用已超上限时应拒绝: %s", recorder.Code, recorder.Body.String())
	}
	if calls != 0 {
		t.Errorf("上游调用次数 = %d, 期望 0", calls)
	}
}

func TestRoutingFallsBackToLocalWhenUpstreamStale(t *testing.T) {
	// 上游数字过期就不该再压过本地统计，否则同步一坏供应商会被永久锁死
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0, "a")})
	if err := module.service.repo.updateProvider(context.Background(), search.ProviderExa,
		map[string]any{"monthly_cost_limit": 10_000_000}); err != nil {
		t.Fatalf("设置费用上限失败: %v", err)
	}
	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}

	runtime := module.service.runtime(search.ProviderExa)
	credentials := runtime.credentials()
	stale := module.service.now().Add(-2 * upstreamUsageTTL)
	runtime.recordUsage(credentials[0].id, search.UsageReport{
		Used: 12_000_000, Unit: search.UsageUnitMicroUSD, Scope: search.UsageScopeKey,
	}, stale)

	if _, ok := runtime.upstreamCostMicroUSD(module.service.now()); ok {
		t.Fatal("过期的上游数字不该再被采用")
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"probe","provider":"exa","no_cache":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 上游数据过期时应回退本地统计放行: %s", recorder.Code, recorder.Body.String())
	}
}
