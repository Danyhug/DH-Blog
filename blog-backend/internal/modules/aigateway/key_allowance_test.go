package aigateway

import (
	"context"
	"net/http"
	"testing"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/platform/search"
)

// runtimeWithKeys builds a provider runtime carrying nothing but credential
// configuration, which is all the allowance and upstream helpers read.
func runtimeWithKeys(config Provider, keys ...ProviderKey) *providerRuntime {
	runtime := &providerRuntime{config: config}
	for index := range keys {
		runtime.keys = append(runtime.keys, &providerKeyRuntime{config: keys[index]})
	}
	return runtime
}

func activeKey(id int, quota int) ProviderKey {
	return ProviderKey{
		BaseModel:    model.BaseModel{ID: id},
		Enabled:      true,
		Status:       ProviderKeyActive,
		MonthlyQuota: quota,
	}
}

func TestAllowanceSumsCredentialsInRotation(t *testing.T) {
	// 两个账号各 1000 就是 2000。供应商级只有一个数字，说不出这件事
	runtime := runtimeWithKeys(Provider{Name: "tavily", MonthlyQuota: 1000},
		activeKey(1, 1000), activeKey(2, 1000))
	usage := map[string]Usage{
		providerKeySubject(1):     {Count: 100},
		providerKeySubject(2):     {Count: 50},
		providerSubject("tavily"): {Count: 150},
	}

	spend := allowanceOf(runtime, time.Now(), usage)
	if spend.Quota != 2000 {
		t.Fatalf("配额 = %d, 两个账号各 1000 应汇总成 2000", spend.Quota)
	}
	if spend.Used != 150 {
		t.Fatalf("已用 = %d, 期望两把密钥之和 150", spend.Used)
	}
}

func TestAllowanceShrinksWhenACredentialIsParked(t *testing.T) {
	// 掉一把密钥就等于少一个账号的额度，容量必须立刻跟着腰斩
	now := time.Now()
	parked := activeKey(2, 1000)
	parked.Status = ProviderKeyQuotaExceeded
	parked.DisabledAt = &now

	runtime := runtimeWithKeys(Provider{Name: "tavily", MonthlyQuota: 1000}, activeKey(1, 1000), parked)
	usage := map[string]Usage{
		providerKeySubject(1): {Count: 100},
		providerKeySubject(2): {Count: 900},
	}

	spend := allowanceOf(runtime, now, usage)
	if spend.Quota != 1000 {
		t.Fatalf("配额 = %d, 停用一把后应只剩 1000", spend.Quota)
	}
	// 停掉那把已经花掉的 900 不该压在还在轮换的账号头上
	if spend.Used != 100 {
		t.Fatalf("已用 = %d, 期望只算还在轮换的密钥", spend.Used)
	}
}

func TestAllowanceFallsBackToProviderLevel(t *testing.T) {
	// 没配密钥级额度的存量安装，行为必须和改动前一模一样
	runtime := runtimeWithKeys(Provider{Name: "tavily", MonthlyQuota: 1000, MonthlyCostLimit: 500},
		activeKey(1, 0), activeKey(2, 0))
	usage := map[string]Usage{providerSubject("tavily"): {Count: 42, CostMicroUSD: 7}}

	spend := allowanceOf(runtime, time.Now(), usage)
	if spend.Quota != 1000 || spend.Used != 42 {
		t.Fatalf("配额/已用 = %d/%d, 期望回退到供应商级的 1000/42", spend.Quota, spend.Used)
	}
	if spend.CostLimit != 500 || spend.CostUsed != 7 {
		t.Fatalf("费用上限/已花 = %d/%d, 期望回退到供应商级的 500/7", spend.CostLimit, spend.CostUsed)
	}
}

func TestAllowanceIsOpenWhenOneCredentialIsUncapped(t *testing.T) {
	// 只加已知的那部分会低估容量，把还有余量的供应商提前挡在门外
	runtime := runtimeWithKeys(Provider{Name: "tavily", MonthlyQuota: 1000},
		activeKey(1, 1000), activeKey(2, 0))
	usage := map[string]Usage{providerKeySubject(1): {Count: 10}, providerKeySubject(2): {Count: 5}}

	spend := allowanceOf(runtime, time.Now(), usage)
	if spend.Quota != 0 {
		t.Fatalf("配额 = %d, 有一把不限量时整体应视为不限", spend.Quota)
	}
	if spend.Used != 15 {
		t.Fatalf("已用 = %d, 期望 15", spend.Used)
	}
}

func syncedKey(id int, used, limit int, unit, scope string, at time.Time) ProviderKey {
	key := activeKey(id, 0)
	key.UpstreamUsed, key.UpstreamLimit = used, limit
	key.UpstreamUnit, key.UpstreamScope = unit, scope
	key.UpstreamSyncedAt = &at
	return key
}

func TestUpstreamCostDoesNotDoubleCountOneAccount(t *testing.T) {
	// 同一个账户下的两把密钥各自报的是同一份账单，加起来就是把同一笔钱算两次
	now := time.Now()
	runtime := runtimeWithKeys(Provider{Name: "exa"},
		syncedKey(1, 12_000_000, 0, search.UsageUnitMicroUSD, search.UsageScopeAccount, now),
		syncedKey(2, 12_000_000, 0, search.UsageUnitMicroUSD, search.UsageScopeAccount, now))

	cost, ok := runtime.upstreamCostMicroUSD(now)
	if !ok {
		t.Fatal("上游已同步，应报出花费")
	}
	if cost != 12_000_000 {
		t.Fatalf("花费 = %d, 同账户不该累加成 24000000", cost)
	}
}

func TestUpstreamCostSumsIndependentKeys(t *testing.T) {
	// key 口径下两把密钥是各花各的，这时候求和才是对的
	now := time.Now()
	runtime := runtimeWithKeys(Provider{Name: "exa"},
		syncedKey(1, 4_000_000, 0, search.UsageUnitMicroUSD, search.UsageScopeKey, now),
		syncedKey(2, 3_000_000, 0, search.UsageUnitMicroUSD, search.UsageScopeKey, now))

	cost, _ := runtime.upstreamCostMicroUSD(now)
	if cost != 7_000_000 {
		t.Fatalf("花费 = %d, 期望两把独立密钥之和 7000000", cost)
	}
}

func TestUpstreamHeadroomSumsKeyScopedRemainders(t *testing.T) {
	now := time.Now()
	runtime := runtimeWithKeys(Provider{Name: "tavily"},
		syncedKey(1, 900, 1000, search.UsageUnitCredit, search.UsageScopeKey, now),
		syncedKey(2, 100, 1000, search.UsageUnitCredit, search.UsageScopeKey, now))

	headroom, ok := runtime.upstreamHeadroom(now)
	if !ok {
		t.Fatal("上游报了额度，应算得出余量")
	}
	if headroom != 0.5 {
		t.Fatalf("余量 = %v, 期望 (100+900)/2000 = 0.5", headroom)
	}
}

func TestUpstreamHeadroomTakesTightestAccountScoped(t *testing.T) {
	// account 口径共用一个池子，加起来会凭空多出一份额度
	now := time.Now()
	runtime := runtimeWithKeys(Provider{Name: "firecrawl"},
		syncedKey(1, 900, 1000, search.UsageUnitCredit, search.UsageScopeAccount, now),
		syncedKey(2, 100, 1000, search.UsageUnitCredit, search.UsageScopeAccount, now))

	headroom, _ := runtime.upstreamHeadroom(now)
	if headroom != 0.1 {
		t.Fatalf("余量 = %v, 期望取最紧的一把 0.1", headroom)
	}
}

func TestUpstreamHeadroomIgnoresParkedAndStale(t *testing.T) {
	now := time.Now()
	parked := syncedKey(1, 1000, 1000, search.UsageUnitCredit, search.UsageScopeKey, now)
	parked.Status = ProviderKeyQuotaExceeded
	parked.DisabledAt = &now
	stale := syncedKey(2, 900, 1000, search.UsageUnitCredit, search.UsageScopeKey,
		now.Add(-2*upstreamUsageTTL))
	fresh := syncedKey(3, 200, 1000, search.UsageUnitCredit, search.UsageScopeKey, now)

	runtime := runtimeWithKeys(Provider{Name: "firecrawl"}, parked, stale, fresh)
	headroom, ok := runtime.upstreamHeadroom(now)
	if !ok {
		t.Fatal("还有一把是新鲜的，应算得出余量")
	}
	if headroom != 0.8 {
		t.Fatalf("余量 = %v, 停用和过期的都不该参与，期望 0.8", headroom)
	}
}

func TestSearchBillsTheCredentialThatServed(t *testing.T) {
	// 密钥级额度全靠这份计数，落不下来就等于没做
	module := newGatewayTestModule(t, gatewayTestConfig{Tavily: tavilyOK("答案", "a")})
	keyID := providerKeyID(t, module, search.ProviderTavily)

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"probe","provider":"tavily","no_cache":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d: %s", recorder.Code, recorder.Body.String())
	}

	subject := providerKeySubject(keyID)
	usage, err := module.service.repo.usageFor(context.Background(),
		currentPeriod(module.service.now()), []string{subject})
	if err != nil {
		t.Fatalf("读取密钥用量失败: %v", err)
	}
	if usage[subject].Count != 1 {
		t.Fatalf("密钥 %d 的本月计数 = %d, 期望 1", keyID, usage[subject].Count)
	}
}

func TestLegacyUsageKeyIDMovesToTheOldestCredential(t *testing.T) {
	// 存量安装把 id 存在供应商的 Extra 里，整条轮换链共用一个，
	// 于是每把密钥都上报同一把的花费，加起来就是翻倍。迁移要把它落到具体那把上。
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: exaOK(0, "a")})
	db := module.service.repo.db
	if err := db.Model(&Provider{}).Where("name = ?", search.ProviderExa).
		Update("extra", `{"search_type":"auto","usage_key_id":"legacy-uuid"}`).Error; err != nil {
		t.Fatalf("构造存量数据失败: %v", err)
	}
	second := addTestProviderKey(t, module, search.ProviderExa, "second-key")

	if err := migrateProviderUsageKeyIDs(db); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	oldest := providerKeyID(t, module, search.ProviderExa)
	if oldest == second {
		t.Fatal("前置条件不成立：应该拿到更早的那把密钥")
	}
	moved, err := module.service.repo.providerKeyByID(context.Background(), oldest)
	if err != nil {
		t.Fatalf("providerKeyByID: %v", err)
	}
	if moved.UsageKeyID != "legacy-uuid" {
		t.Fatalf("最早的密钥上的 id = %q, 期望迁移过来", moved.UsageKeyID)
	}
	// 后加的那把不该跟着拿到同一个 id，否则翻倍问题原样保留
	later, err := module.service.repo.providerKeyByID(context.Background(), second)
	if err != nil {
		t.Fatalf("providerKeyByID: %v", err)
	}
	if later.UsageKeyID != "" {
		t.Fatalf("第二把密钥的 id = %q, 不该共用", later.UsageKeyID)
	}

	provider, err := module.service.repo.providerByName(context.Background(), search.ProviderExa)
	if err != nil {
		t.Fatalf("providerByName: %v", err)
	}
	if got := extraString(provider.Extra, extraKeyUsageKeyID); got != "" {
		t.Fatalf("Extra 里还留着 id = %q, 两处都存会各自漂移", got)
	}
	if got := extraString(provider.Extra, "search_type"); got != "auto" {
		t.Fatalf("search_type = %q, 迁移不该丢掉其它选项", got)
	}
}
