package aigateway

import (
	"reflect"
	"testing"

	"dh-blog/internal/platform/search"
)

var (
	braveCapability  = search.Capability{SearchOperators: true, Pagination: true}
	tavilyCapability = search.Capability{Answer: true, RawContent: true, DomainFilter: true}
)

func healthyCandidate(name string, capability search.Capability) candidate {
	return candidate{Name: name, Capability: capability, Priority: 100, Weight: 1, Healthy: true}
}

func TestRouteHonoursExplicitProvider(t *testing.T) {
	in := routeInput{
		Requested:  search.ProviderBrave,
		Candidates: []candidate{healthyCandidate("brave", braveCapability), healthyCandidate("tavily", tavilyCapability)},
	}
	order, err := route(in)
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "brave" {
		t.Fatalf("首选 = %q, 期望尊重显式指定", order[0])
	}
}

func TestRouteExplicitProviderOverridesCapabilityFilter(t *testing.T) {
	// Brave cannot produce an answer, but an explicit choice wins: the caller
	// gets Brave and an empty answer rather than a silent redirect.
	in := routeInput{
		Requested:  search.ProviderBrave,
		NeedAnswer: true,
		Candidates: []candidate{healthyCandidate("brave", braveCapability), healthyCandidate("tavily", tavilyCapability)},
	}
	order, err := route(in)
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "brave" {
		t.Fatalf("首选 = %q, 显式指定应优先于能力过滤", order[0])
	}
}

func TestRouteExplicitUnknownProvider(t *testing.T) {
	_, err := route(routeInput{
		Requested:  "google",
		Candidates: []candidate{healthyCandidate("brave", braveCapability)},
	})
	if err != ErrProviderNotFound {
		t.Fatalf("错误 = %v, 期望 ErrProviderNotFound", err)
	}
}

func TestRouteExplicitProviderOutsideKeyWhitelist(t *testing.T) {
	_, err := route(routeInput{
		Requested:  "tavily",
		Allowed:    []string{"brave"},
		Candidates: []candidate{healthyCandidate("brave", braveCapability), healthyCandidate("tavily", tavilyCapability)},
	})
	if err != ErrProviderNotAllowed {
		t.Fatalf("错误 = %v, 期望 ErrProviderNotAllowed", err)
	}
}

func TestRouteExplicitUnavailableWithoutFallback(t *testing.T) {
	brave := healthyCandidate("brave", braveCapability)
	brave.Healthy = false

	_, err := route(routeInput{
		Requested:     "brave",
		AllowFallback: false,
		Candidates:    []candidate{brave, healthyCandidate("tavily", tavilyCapability)},
	})
	if err != ErrNoProviderAvailable {
		t.Fatalf("错误 = %v, 期望禁止回退时直接失败", err)
	}
}

func TestRouteExplicitUnavailableFallsBack(t *testing.T) {
	brave := healthyCandidate("brave", braveCapability)
	brave.Healthy = false

	order, err := route(routeInput{
		Requested:     "brave",
		AllowFallback: true,
		Candidates:    []candidate{brave, healthyCandidate("tavily", tavilyCapability)},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if len(order) != 1 || order[0] != "tavily" {
		t.Fatalf("顺序 = %v, 期望回退到 tavily", order)
	}
}

func TestRouteExplicitAppendsFallbackTail(t *testing.T) {
	order, err := route(routeInput{
		Requested:     "brave",
		AllowFallback: true,
		Candidates:    []candidate{healthyCandidate("brave", braveCapability), healthyCandidate("tavily", tavilyCapability)},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if len(order) != 2 || order[0] != "brave" || order[1] != "tavily" {
		t.Fatalf("顺序 = %v, 期望 [brave tavily]", order)
	}
}

func TestRouteCapabilityHardFilters(t *testing.T) {
	tests := []struct {
		name  string
		input routeInput
		want  string
	}{
		{
			name:  "需要答案时只能走 tavily",
			input: routeInput{NeedAnswer: true},
			want:  "tavily",
		},
		{
			name:  "需要正文时只能走 tavily",
			input: routeInput{NeedRawContent: true},
			want:  "tavily",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.Candidates = []candidate{
				healthyCandidate("brave", braveCapability),
				healthyCandidate("tavily", tavilyCapability),
			}
			order, err := route(test.input)
			if err != nil {
				t.Fatalf("route 返回错误: %v", err)
			}
			if len(order) != 1 || order[0] != test.want {
				t.Fatalf("顺序 = %v, 期望仅 [%s]", order, test.want)
			}
		})
	}
}

func TestRouteAnswerRequestFailsWhenNoProviderSupportsIt(t *testing.T) {
	_, err := route(routeInput{
		NeedAnswer: true,
		Candidates: []candidate{healthyCandidate("brave", braveCapability)},
	})
	if err != ErrNoProviderAvailable {
		t.Fatalf("错误 = %v, 期望能力硬约束导致无可用供应商", err)
	}
}

func TestRouteOperatorPreferenceIsSoft(t *testing.T) {
	// Operators steer towards Brave...
	order, err := route(routeInput{
		PrefersOperators: true,
		Candidates:       []candidate{healthyCandidate("brave", braveCapability), healthyCandidate("tavily", tavilyCapability)},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "brave" {
		t.Fatalf("首选 = %q, 期望操作符查询偏向 brave", order[0])
	}

	// ...but never leave the caller with nothing when Brave is unavailable.
	order, err = route(routeInput{
		PrefersOperators: true,
		Candidates:       []candidate{healthyCandidate("tavily", tavilyCapability)},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if len(order) != 1 || order[0] != "tavily" {
		t.Fatalf("顺序 = %v, 操作符偏好应是软约束", order)
	}
}

func TestRouteSkipsUnhealthyAndExhaustedCandidates(t *testing.T) {
	unhealthy := healthyCandidate("brave", braveCapability)
	unhealthy.Healthy = false

	exhausted := healthyCandidate("tavily", tavilyCapability)
	exhausted.MonthlyQuota = 100
	exhausted.Used = 100

	_, err := route(routeInput{Candidates: []candidate{unhealthy, exhausted}})
	if err != ErrNoProviderAvailable {
		t.Fatalf("错误 = %v, 期望熔断与配额耗尽的候选都被剔除", err)
	}
}

func TestRoutePrefersHigherRemainingQuota(t *testing.T) {
	// Brave has burnt 95% of its free tier; Tavily is barely touched.
	brave := healthyCandidate("brave", braveCapability)
	brave.MonthlyQuota, brave.Used = 2000, 1900

	tavily := healthyCandidate("tavily", tavilyCapability)
	tavily.MonthlyQuota, tavily.Used = 1000, 100

	order, err := route(routeInput{Candidates: []candidate{brave, tavily}})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "tavily" {
		t.Fatalf("首选 = %q, 期望剩余配额比例高者优先", order[0])
	}
}

func TestRouteWeightBreaksQuotaTie(t *testing.T) {
	light := healthyCandidate("brave", braveCapability)
	light.Weight = 1

	heavy := healthyCandidate("tavily", tavilyCapability)
	heavy.Weight = 5

	order, err := route(routeInput{Candidates: []candidate{light, heavy}})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "tavily" {
		t.Fatalf("首选 = %q, 配额相同时应由权重决定", order[0])
	}
}

func TestRouteBalancedIgnoresPriority(t *testing.T) {
	// 负载均衡下优先级不参与排序，否则"均衡"就被固定顺序架空了
	primary := healthyCandidate("brave", braveCapability)
	primary.Priority, primary.MonthlyQuota, primary.Used = 1, 2000, 1990

	standby := healthyCandidate("tavily", tavilyCapability)
	standby.Priority, standby.MonthlyQuota, standby.Used = 2, 1000, 0

	order, err := route(routeInput{
		Strategy:   StrategyBalanced,
		Candidates: []candidate{primary, standby},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "tavily" {
		t.Fatalf("首选 = %q, 负载均衡应只看剩余配额与权重", order[0])
	}
}

func TestRoutePriorityStrategyFollowsConfiguredOrder(t *testing.T) {
	// 按优先级模式下，主供应商只要还可用就一直用，剩余配额再少也不让位
	primary := healthyCandidate("brave", braveCapability)
	primary.Priority, primary.MonthlyQuota, primary.Used = 1, 2000, 1990

	standby := healthyCandidate("tavily", tavilyCapability)
	standby.Priority, standby.MonthlyQuota, standby.Used = 2, 1000, 0

	order, err := route(routeInput{
		Strategy:   StrategyPriority,
		Candidates: []candidate{standby, primary}, // 故意乱序传入
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if len(order) != 2 || order[0] != "brave" || order[1] != "tavily" {
		t.Fatalf("顺序 = %v, 期望 [brave tavily]", order)
	}
}

func TestRoutePriorityStrategyStillExcludesExhausted(t *testing.T) {
	// 配额是过滤条件而非排序因子：用尽的主供应商必须让位
	primary := healthyCandidate("brave", braveCapability)
	primary.Priority, primary.MonthlyQuota, primary.Used = 1, 2000, 2000

	standby := healthyCandidate("tavily", tavilyCapability)
	standby.Priority = 2

	order, err := route(routeInput{
		Strategy:   StrategyPriority,
		Candidates: []candidate{primary, standby},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if len(order) != 1 || order[0] != "tavily" {
		t.Fatalf("顺序 = %v, 配额用尽的主供应商应被剔除", order)
	}
}

func TestRoutePriorityStrategyBreaksTiesByWeight(t *testing.T) {
	light := healthyCandidate("brave", braveCapability)
	light.Priority, light.Weight = 5, 1

	heavy := healthyCandidate("tavily", tavilyCapability)
	heavy.Priority, heavy.Weight = 5, 9

	order, err := route(routeInput{
		Strategy:   StrategyPriority,
		Candidates: []candidate{light, heavy},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "tavily" {
		t.Fatalf("首选 = %q, 同优先级应由权重决定", order[0])
	}
}

func TestRouteModelStrategyFallsBackToBalanced(t *testing.T) {
	// 模型调度尚未接入，必须退回一个说得清的顺序，而不是自造一个
	rich := healthyCandidate("tavily", tavilyCapability)
	rich.MonthlyQuota, rich.Used = 1000, 10

	poor := healthyCandidate("brave", braveCapability)
	poor.MonthlyQuota, poor.Used = 2000, 1990

	model, err := route(routeInput{Strategy: StrategyModel, Candidates: []candidate{poor, rich}})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	balanced, err := route(routeInput{Strategy: StrategyBalanced, Candidates: []candidate{poor, rich}})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if !reflect.DeepEqual(model, balanced) {
		t.Fatalf("模型调度顺序 %v 应与负载均衡 %v 一致", model, balanced)
	}
}

func TestRoutingStrategyValidity(t *testing.T) {
	tests := []struct {
		strategy    RoutingStrategy
		valid       bool
		implemented bool
	}{
		{StrategyBalanced, true, true},
		{StrategyPriority, true, true},
		{StrategyModel, true, false},
		{RoutingStrategy("random"), false, true},
		{RoutingStrategy(""), false, true},
	}
	for _, test := range tests {
		t.Run(string(test.strategy), func(t *testing.T) {
			if got := test.strategy.Valid(); got != test.valid {
				t.Errorf("Valid() = %v, 期望 %v", got, test.valid)
			}
			if got := test.strategy.Implemented(); got != test.implemented {
				t.Errorf("Implemented() = %v, 期望 %v", got, test.implemented)
			}
		})
	}
}

func TestRouteAppliesKeyWhitelistInAutoMode(t *testing.T) {
	order, err := route(routeInput{
		Allowed:    []string{"tavily"},
		Candidates: []candidate{healthyCandidate("brave", braveCapability), healthyCandidate("tavily", tavilyCapability)},
	})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if len(order) != 1 || order[0] != "tavily" {
		t.Fatalf("顺序 = %v, 期望仅保留白名单内的供应商", order)
	}
}

func TestRouteWithoutCandidates(t *testing.T) {
	if _, err := route(routeInput{}); err != ErrNoProviderAvailable {
		t.Fatalf("错误 = %v, 期望 ErrNoProviderAvailable", err)
	}
}

func TestQueryUsesOperators(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"site:github.com rust", true},
		{"machine learning filetype:pdf", true},
		{"intitle:golang", true},
		{`"climate change solutions"`, true},
		{"go 1.25 新特性", false},
		{`带一个"引号`, false},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			if got := queryUsesOperators(test.query); got != test.want {
				t.Errorf("queryUsesOperators(%q) = %v, 期望 %v", test.query, got, test.want)
			}
		})
	}
}

// drainRoute plays n consecutive routing decisions, feeding each pick back as
// usage so the next decision sees the traffic the previous one created.
func drainRoute(t *testing.T, rounds int, costPerCall map[string]int, pool ...candidate) map[string]int {
	t.Helper()
	picks := make(map[string]int, len(pool))
	for round := 0; round < rounds; round++ {
		order, err := route(routeInput{Candidates: append([]candidate(nil), pool...)})
		if err != nil {
			t.Fatalf("第 %d 轮 route 返回错误: %v", round, err)
		}
		winner := order[0]
		picks[winner]++
		for index := range pool {
			if pool[index].Name != winner {
				continue
			}
			pool[index].Used++
			pool[index].CostUsed += costPerCall[winner]
		}
	}
	return picks
}

func TestBalancedKeepsUsingACostCappedProvider(t *testing.T) {
	// 回归：Exa 是唯一会上报真实费用的供应商，配了费用上限后剩余比例会随消费下降，
	// 而两个上限都不设的 Firecrawl 恒为 1.0。旧打分只看「剩余比例 + 静态权重」，
	// 于是 Firecrawl 永远压过 Exa —— Exa 只在第一次打平时按名字排序被选中一次，
	// 之后再也轮不到。
	exa := healthyCandidate("exa", exaCapability)
	exa.MonthlyCostLimit = 10_000_000 // 免费额度 $10
	firecrawl := healthyCandidate("firecrawl", tavilyCapability)

	picks := drainRoute(t, 20, map[string]int{"exa": 7_000}, exa, firecrawl)

	// 判据必须看份额而不是「有没有被选中过」：旧打分下 Exa 恰好会在第一轮打平时
	// 被选中一次，用 >0 判定是抓不住这个 bug 的。
	if picks["exa"] < 5 {
		t.Fatalf("20 轮选路里 Exa 只拿到 %d 次: %v", picks["exa"], picks)
	}
	if picks["firecrawl"] < 5 {
		t.Fatalf("20 轮选路里 Firecrawl 只拿到 %d 次: %v", picks["firecrawl"], picks)
	}
}

func TestBalancedSpreadsTrafficAccordingToWeight(t *testing.T) {
	// 权重表达的是流量份额：3:1 的配置跑够轮数后应该接近 3:1，而不是让重的那个通吃
	heavy := healthyCandidate("exa", exaCapability)
	heavy.Weight = 3
	light := healthyCandidate("firecrawl", tavilyCapability)
	light.Weight = 1

	picks := drainRoute(t, 40, map[string]int{}, heavy, light)

	if picks["firecrawl"] == 0 {
		t.Fatalf("低权重供应商完全没有拿到流量: %v", picks)
	}
	ratio := float64(picks["exa"]) / float64(picks["firecrawl"])
	if ratio < 2 || ratio > 4 {
		t.Fatalf("流量比例 = %.2f (%v), 期望接近权重比 3:1", ratio, picks)
	}
}

func TestBalancedStillYieldsWhenAllowanceRunsLow(t *testing.T) {
	// 均衡不能盖过余量：快用完的供应商即使最闲，也该排在余量充足的后面
	nearlySpent := healthyCandidate("exa", exaCapability)
	nearlySpent.MonthlyCostLimit, nearlySpent.CostUsed = 10_000_000, 9_500_000

	roomy := healthyCandidate("firecrawl", tavilyCapability)
	roomy.Used = 500 // 用得多得多，但完全没有上限

	order, err := route(routeInput{Candidates: []candidate{nearlySpent, roomy}})
	if err != nil {
		t.Fatalf("route 返回错误: %v", err)
	}
	if order[0] != "firecrawl" {
		t.Fatalf("首选 = %q, 余量只剩 5%% 的供应商不该排在最前", order[0])
	}
}
