package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"dh-blog/internal/platform/search"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// exaCapability mirrors what the Exa adapter reports, so the spend-capped
// candidate below is filtered on the same terms the real one would be.
var exaCapability = search.Capability{Answer: true, RawContent: true, DomainFilter: true}

func TestRouteExcludesCandidateOverSpendCap(t *testing.T) {
	// 按次数算 Exa 还早得很（根本没设次数上限），但按金额已经到顶
	exa := healthyCandidate("exa", exaCapability)
	exa.MonthlyCostLimit = 10_000_000
	exa.CostUsed = 10_000_000

	order, err := route(routeInput{Candidates: []candidate{exa, healthyCandidate("brave", braveCapability)}})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if len(order) != 1 || order[0] != "brave" {
		t.Errorf("候选顺序 = %v, 期望只剩 brave", order)
	}
}

func TestRouteExplicitSpendCappedProviderIsRejected(t *testing.T) {
	// 显式指定能越过能力过滤，但越不过额度：钱花完了就是花完了
	exa := healthyCandidate("exa", exaCapability)
	exa.MonthlyCostLimit = 5_000_000
	exa.CostUsed = 6_200_000

	_, err := route(routeInput{Requested: "exa", Candidates: []candidate{exa}})
	if err != ErrNoProviderAvailable {
		t.Errorf("err = %v, 期望 %v", err, ErrNoProviderAvailable)
	}
}

func TestRoutePrefersProviderWithMoreSpendLeft(t *testing.T) {
	poor := healthyCandidate("exa", exaCapability)
	poor.MonthlyCostLimit = 10_000_000
	poor.CostUsed = 9_000_000 // 剩 10%

	rich := healthyCandidate("tavily", tavilyCapability)
	rich.MonthlyQuota = 1000
	rich.Used = 100 // 剩 90%

	order, err := route(routeInput{Candidates: []candidate{poor, rich}})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "tavily" {
		t.Errorf("首选 = %q, 期望余量更充裕的 tavily", order[0])
	}
}

func TestScoreUsesTighterOfTheTwoAllowances(t *testing.T) {
	// 两种上限都设了：次数还剩九成、金额只剩一成，排序要按更紧的那个来，
	// 否则会一直往一个马上要断的供应商上送
	both := healthyCandidate("exa", exaCapability)
	both.MonthlyQuota, both.Used = 1000, 100
	both.MonthlyCostLimit, both.CostUsed = 10_000_000, 9_000_000

	onlyCalls := healthyCandidate("exa", exaCapability)
	onlyCalls.MonthlyQuota, onlyCalls.Used = 1000, 100

	if score(both, balanceTotals{}) >= score(onlyCalls, balanceTotals{}) {
		t.Error("金额快用完时得分不应与只看次数时相同")
	}
}

func TestUncappedProviderScoresAsFullyAvailable(t *testing.T) {
	// 不设上限不等于余量为 0，否则没配额的供应商会被永远排在最后
	free := healthyCandidate("brave", braveCapability)
	full := healthyCandidate("tavily", tavilyCapability)
	full.MonthlyQuota, full.Used = 1000, 0

	if score(free, balanceTotals{}) != score(full, balanceTotals{}) {
		t.Errorf("无上限得分 = %v, 满额度得分 = %v, 期望相等", score(free, balanceTotals{}), score(full, balanceTotals{}))
	}
}

func TestSearchStopsAtSpendCap(t *testing.T) {
	// Exa 每次调用自己报花了多少钱，网关按这个累加；到顶后不该再放行
	var calls int
	module := newGatewayTestModule(t, gatewayTestConfig{Exa: func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"t","url":"https://example.com"}],"costDollars":{"total":0.005}}`))
	}})
	if err := module.service.repo.updateProvider(context.Background(), "exa",
		map[string]any{"monthly_cost_limit": 10_000}); err != nil { // 0.01 美元，够两次
		t.Fatalf("设置费用上限失败: %v", err)
	}
	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)
	for index := 0; index < 3; index++ {
		body := `{"query":"probe ` + string(rune('a'+index)) + `","provider":"exa","no_cache":true}`
		recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, body)
		if index < 2 {
			if recorder.Code != http.StatusOK {
				t.Fatalf("第 %d 次状态码 = %d, 期望 200: %s", index+1, recorder.Code, recorder.Body.String())
			}
			continue
		}
		// 闸门看的是"已经花到顶没有"，而不是"这一次会不会超"——
		// Exa 的单次价格要等响应回来才知道，所以最多超出一次调用的钱
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("第 3 次状态码 = %d, 期望 503（费用已到顶）: %s", recorder.Code, recorder.Body.String())
		}
	}
	if calls != 2 {
		t.Errorf("上游调用次数 = %d, 期望 2：到顶后不该再打上游", calls)
	}
}

func TestAdminRejectsNegativeSpendCap(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	recorder := doAdmin(engine, http.MethodPut, "/api/admin/gateway/providers/exa",
		`{"monthlyCostLimitMicroUsd":-1}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, 期望 400", recorder.Code)
	}
}

func TestAdminReportsSpendCap(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	if recorder := doAdmin(engine, http.MethodPut, "/api/admin/gateway/providers/exa",
		`{"monthlyCostLimitMicroUsd":10000000}`); recorder.Code != http.StatusOK {
		t.Fatalf("保存费用上限失败: %s", recorder.Body.String())
	}

	recorder := doAdmin(engine, http.MethodGet, "/api/admin/gateway/providers", "")
	var response struct {
		Data []providerView `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, view := range response.Data {
		if view.Name != "exa" {
			continue
		}
		if view.MonthlyCostLimit != 10_000_000 {
			t.Errorf("费用上限 = %d, 期望 10000000", view.MonthlyCostLimit)
		}
		return
	}
	t.Fatal("响应中没有 exa")
}

func TestExaSeedCarriesFreeTierSpendCap(t *testing.T) {
	// 新装默认按 Exa 免费额度的每月 10 美元封顶，而不是敞着口子
	module := newGatewayTestModule(t, gatewayTestConfig{})
	provider, err := module.service.repo.providerByName(context.Background(), "exa")
	if err != nil {
		t.Fatalf("读取 exa 失败: %v", err)
	}
	if provider.MonthlyCostLimit != 10_000_000 {
		t.Errorf("默认费用上限 = %d, 期望 10000000（$10）", provider.MonthlyCostLimit)
	}
	if provider.MonthlyQuota != 0 {
		t.Errorf("默认次数上限 = %d, 期望 0：Exa 按金额计费，次数上限说明不了预算", provider.MonthlyQuota)
	}
}

func TestUpgradeDoesNotImposeSpendCapOnExistingProvider(t *testing.T) {
	// 老库里已有 exa 行，升级只会补出一个值为 0 的新列。
	// 这里刻意不回填默认上限：升级后突然开始拦流量，比敞着口子更糟
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
	existing := Provider{Name: "exa", DisplayName: "Exa", Enabled: true, Priority: 100, Weight: 1, RPS: 5}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("写入老数据失败: %v", err)
	}

	module, err := New(Dependencies{DB: db, Cache: newTestCache(), Options: defaultTestOptions()})
	if err != nil {
		t.Fatalf("构建网关模块失败: %v", err)
	}
	t.Cleanup(module.Shutdown)

	provider, err := module.service.repo.providerByName(context.Background(), "exa")
	if err != nil {
		t.Fatalf("读取 exa 失败: %v", err)
	}
	if provider.MonthlyCostLimit != 0 {
		t.Errorf("升级后费用上限 = %d, 期望保持 0", provider.MonthlyCostLimit)
	}
}
