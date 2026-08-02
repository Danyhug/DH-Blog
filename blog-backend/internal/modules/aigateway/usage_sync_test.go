package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// usageServer answers Tavily's /usage with a payload the test controls, and
// serves a plain search result for anything else.
func usageServer(payload *string, status *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/usage" {
			if status != nil && *status != 0 {
				w.WriteHeader(*status)
			}
			_, _ = w.Write([]byte(*payload))
			return
		}
		_, _ = w.Write([]byte(`{"query":"x","results":[]}`))
	}
}

// keyRow reads one credential straight from the database.
func keyRow(t *testing.T, module *Module, id int) ProviderKey {
	t.Helper()
	stored, err := module.service.repo.providerKeyByID(context.Background(), id)
	if err != nil {
		t.Fatalf("读取供应商密钥失败: %v", err)
	}
	return stored
}

func TestSyncUsageStoresUpstreamNumbers(t *testing.T) {
	payload := `{"key":{"usage":150,"limit":1000},"account":{"plan_usage":150,"plan_limit":1000}}`
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: usageServer(&payload, nil)})

	result := module.service.SyncUsage(context.Background())
	if result.Synced != 1 {
		t.Fatalf("Synced = %d, 期望 1", result.Synced)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, 期望 0", result.Failed)
	}

	keys, err := module.service.repo.listProviderKeys(context.Background())
	if err != nil {
		t.Fatalf("列出供应商密钥失败: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("密钥数 = %d, 期望 1", len(keys))
	}
	stored := keys[0]
	if stored.UpstreamUsed != 150 || stored.UpstreamLimit != 1000 {
		t.Errorf("上游用量 = %d/%d, 期望 150/1000", stored.UpstreamUsed, stored.UpstreamLimit)
	}
	if stored.UpstreamUnit != "credit" {
		t.Errorf("单位 = %q, 期望 credit", stored.UpstreamUnit)
	}
	if stored.UpstreamSyncedAt == nil {
		t.Error("期望记录同步时间，否则页面无法说明数字有多新")
	}
	// 本地计数与上游用量是两套账，同步不该篡改前者
	usage, err := module.service.repo.usageFor(context.Background(),
		currentPeriod(module.service.now()), []string{providerSubject("tavily")})
	if err != nil {
		t.Fatalf("读取本地用量失败: %v", err)
	}
	if usage[providerSubject("tavily")].Count != 0 {
		t.Errorf("本地请求计数 = %d, 期望 0：上游是 credit，两者不能混记",
			usage[providerSubject("tavily")].Count)
	}
}

func TestSyncUsageParksExhaustedCredential(t *testing.T) {
	payload := `{"key":{"usage":1000,"limit":1000},"account":{"plan_usage":1000,"plan_limit":1000}}`
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: usageServer(&payload, nil)})
	runtime := module.service.runtime("tavily")
	if runtime == nil {
		t.Fatal("tavily 运行时缺失")
	}

	result := module.service.SyncUsage(context.Background())
	if len(result.Parked) != 1 {
		t.Fatalf("Parked = %v, 期望停用 1 把密钥", result.Parked)
	}

	stored := keyRow(t, module, runtime.credentials()[0].id)
	if stored.Status != ProviderKeyQuotaExceeded {
		t.Errorf("状态 = %q, 期望 %q", stored.Status, ProviderKeyQuotaExceeded)
	}
	if stored.LastError == "" {
		t.Error("期望写明停用原因，否则页面只显示'不可用'")
	}
	// 停用要立刻生效在内存里，不能等下一次 Reload
	if runtime.usableKeys(module.service.now()) != 0 {
		t.Error("期望密钥立即退出轮换")
	}
}

func TestSyncUsageRevivesRecoveredCredential(t *testing.T) {
	payload := `{"key":{"usage":1000,"limit":1000},"account":{"plan_usage":1000,"plan_limit":1000}}`
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: usageServer(&payload, nil)})
	runtime := module.service.runtime("tavily")

	module.service.SyncUsage(context.Background())
	if runtime.usableKeys(module.service.now()) != 0 {
		t.Fatal("前置条件：密钥应先被停用")
	}

	// 套餐续费后上游自己就报出余量，不必等到下个自然月
	payload = `{"key":{"usage":12,"limit":1000},"account":{"plan_usage":12,"plan_limit":1000}}`
	result := module.service.SyncUsage(context.Background())
	if len(result.Revived) != 1 {
		t.Fatalf("Revived = %v, 期望恢复 1 把密钥", result.Revived)
	}
	if runtime.usableKeys(module.service.now()) != 1 {
		t.Error("期望密钥重新进入轮换")
	}
	stored := keyRow(t, module, runtime.credentials()[0].id)
	if stored.Status != ProviderKeyActive {
		t.Errorf("状态 = %q, 期望 %q", stored.Status, ProviderKeyActive)
	}
}

func TestSyncUsageParksRejectedCredential(t *testing.T) {
	payload := `{"detail":{"error":"Invalid API key"}}`
	status := http.StatusUnauthorized
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: usageServer(&payload, &status)})
	runtime := module.service.runtime("tavily")

	result := module.service.SyncUsage(context.Background())
	if result.Failed != 1 {
		t.Fatalf("Failed = %d, 期望 1", result.Failed)
	}
	if len(result.Parked) != 1 {
		t.Fatalf("Parked = %v, 期望被拒的密钥直接停用", result.Parked)
	}

	stored := keyRow(t, module, runtime.credentials()[0].id)
	if stored.Status != ProviderKeyAuthFailed {
		t.Errorf("状态 = %q, 期望 %q", stored.Status, ProviderKeyAuthFailed)
	}
	if stored.UpstreamError == "" {
		t.Error("期望记下同步失败的原因")
	}
}

func TestSyncUsageKeepsLastNumbersWhenRefreshFails(t *testing.T) {
	payload := `{"key":{"usage":150,"limit":1000},"account":{"plan_usage":150,"plan_limit":1000}}`
	status := 0
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: usageServer(&payload, &status)})
	runtime := module.service.runtime("tavily")
	module.service.SyncUsage(context.Background())

	// 上游暂时 500：旧数字比空白有用，不该被抹掉
	payload, status = `{"error":"boom"}`, http.StatusInternalServerError
	module.service.SyncUsage(context.Background())

	stored := keyRow(t, module, runtime.credentials()[0].id)
	if stored.UpstreamUsed != 150 || stored.UpstreamLimit != 1000 {
		t.Errorf("上游用量 = %d/%d, 期望保留上次的 150/1000", stored.UpstreamUsed, stored.UpstreamLimit)
	}
	if stored.UpstreamError == "" {
		t.Error("期望记下这次同步失败")
	}
	if stored.Status != ProviderKeyActive {
		t.Errorf("状态 = %q, 期望仍为 %q：读不到用量不等于密钥有问题", stored.Status, ProviderKeyActive)
	}
}

func TestSyncUsageSkipsProvidersWithoutUsageAPI(t *testing.T) {
	// Exa 查用量要另一把 service key，网关只能继续用本地计数
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/usage" {
			t.Errorf("不应向 Exa 请求 %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}})

	result := module.service.SyncUsage(context.Background())
	if result.Synced != 0 || result.Failed != 0 {
		t.Errorf("Synced = %d, Failed = %d, 期望都为 0", result.Synced, result.Failed)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, 期望 1", result.Skipped)
	}
}

func TestSyncUsageSkipsDisabledCredential(t *testing.T) {
	payload := `{"key":{"usage":1,"limit":1000},"account":{"plan_usage":1,"plan_limit":1000}}`
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: usageServer(&payload, nil)})
	runtime := module.service.runtime("tavily")
	id := runtime.credentials()[0].id

	if err := module.service.repo.updateProviderKey(context.Background(), id,
		map[string]any{"enabled": false}); err != nil {
		t.Fatalf("停用密钥失败: %v", err)
	}
	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}

	result := module.service.SyncUsage(context.Background())
	if result.Synced != 0 {
		t.Errorf("Synced = %d, 期望 0：管理员关掉的密钥不该再去打上游", result.Synced)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, 期望 1", result.Skipped)
	}
}

func TestUpstreamExhaustedOverridesMonthlySelfHealing(t *testing.T) {
	parked := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	nextMonth := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	fresh := nextMonth.Add(-30 * time.Minute)
	key := ProviderKey{
		Enabled: true, Status: ProviderKeyQuotaExceeded, DisabledAt: &parked,
		UpstreamUsed: 15000, UpstreamLimit: 15000, UpstreamSyncedAt: &fresh,
	}
	// 上个月停的、这个月按理自愈，但半小时前的同步说还是满的（Brave 是滚动 30 天）
	if key.Usable(nextMonth) {
		t.Error("上游报告仍为用尽时，不该因为跨月就放回轮换")
	}

	// 同步本身坏了很久，就别再拿老数字压着按月自愈
	stale := nextMonth.Add(-72 * time.Hour)
	key.UpstreamSyncedAt = &stale
	if !key.Usable(nextMonth) {
		t.Error("上游数字过期后应回到按月自愈，避免同步故障永久停用密钥")
	}
}

func TestAdminUsageSyncEndpoint(t *testing.T) {
	payload := `{"key":{"usage":9,"limit":1000},"account":{"plan_usage":9,"plan_limit":1000}}`
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: usageServer(&payload, nil)})
	engine := newTestEngine(module)

	recorder := doAdmin(engine, http.MethodPost, "/api/admin/gateway/usage/sync", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data UsageSyncResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if response.Data.Synced != 1 {
		t.Errorf("Synced = %d, 期望 1", response.Data.Synced)
	}
}
