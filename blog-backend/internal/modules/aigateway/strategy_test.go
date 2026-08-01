package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func adminRequest(engine *http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	(*engine).ServeHTTP(recorder, request)
	return recorder
}

func decodeAdmin[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应失败: %v (body=%s)", err, recorder.Body.String())
	}
	if envelope.Code != 1 {
		t.Fatalf("接口返回失败: %s", envelope.Msg)
	}
	var data T
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	return data
}

func TestGatewayDefaultsToBalancedStrategy(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	if got := module.service.Strategy(); got != StrategyBalanced {
		t.Fatalf("默认调度方式 = %q, 期望 balanced", got)
	}
}

func TestAdminSettingsExposeStrategyOptions(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	var handler http.Handler = newTestEngine(module)

	view := decodeAdmin[settingsView](t, adminRequest(&handler, http.MethodGet, "/api/admin/gateway/settings", ""))
	if view.RoutingStrategy != string(StrategyBalanced) {
		t.Errorf("当前调度方式 = %q", view.RoutingStrategy)
	}
	if len(view.Strategies) != 3 {
		t.Fatalf("调度方式数量 = %d, 期望 3", len(view.Strategies))
	}

	byValue := map[string]strategyOption{}
	for _, option := range view.Strategies {
		if option.Label == "" || option.Description == "" {
			t.Errorf("调度方式 %q 缺少文案: %+v", option.Value, option)
		}
		byValue[option.Value] = option
	}
	for _, value := range []string{string(StrategyBalanced), string(StrategyPriority), string(StrategyModel)} {
		if _, ok := byValue[value]; !ok {
			t.Errorf("缺少调度方式 %q", value)
		}
	}
	// 模型调度必须被标记为未接入，否则界面会谎称它已经生效
	if byValue[string(StrategyModel)].Implemented {
		t.Error("模型调度尚未接入，不应标记为已实现")
	}
	if !byValue[string(StrategyBalanced)].Implemented || !byValue[string(StrategyPriority)].Implemented {
		t.Error("负载均衡与按优先级应标记为已实现")
	}
}

func TestAdminUpdateStrategyTakesEffectImmediately(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	var handler http.Handler = newTestEngine(module)

	recorder := adminRequest(&handler, http.MethodPut, "/api/admin/gateway/settings", `{"routingStrategy":"priority"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := module.service.Strategy(); got != StrategyPriority {
		t.Fatalf("调度方式 = %q, 修改应立即生效", got)
	}

	view := decodeAdmin[settingsView](t, adminRequest(&handler, http.MethodGet, "/api/admin/gateway/settings", ""))
	if view.RoutingStrategy != string(StrategyPriority) {
		t.Errorf("回读的调度方式 = %q", view.RoutingStrategy)
	}
}

func TestAdminRejectsUnknownStrategy(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	var handler http.Handler = newTestEngine(module)

	recorder := adminRequest(&handler, http.MethodPut, "/api/admin/gateway/settings", `{"routingStrategy":"random"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, 期望 400", recorder.Code)
	}
	if got := module.service.Strategy(); got != StrategyBalanced {
		t.Errorf("非法值不应改变现有配置, 当前 = %q", got)
	}
}

func TestStrategySurvivesReload(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	ctx := context.Background()

	if err := module.service.SetStrategy(ctx, StrategyPriority); err != nil {
		t.Fatalf("SetStrategy 返回错误: %v", err)
	}
	// 供应商配置变更会触发 Reload，调度方式不能被顺手重置回默认值
	if err := module.service.Reload(ctx); err != nil {
		t.Fatalf("Reload 返回错误: %v", err)
	}
	if got := module.service.Strategy(); got != StrategyPriority {
		t.Fatalf("Reload 后调度方式 = %q, 期望保持 priority", got)
	}
}

func TestStrategyChangesLiveRouting(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  braveOK("b1"),
		Tavily: tavilyOK("", "t1"),
	})
	ctx := context.Background()

	// brave 优先级更高但配额几乎耗尽；两种策略应给出相反的选择
	if err := module.service.repo.updateProvider(ctx, "brave", map[string]any{"priority": 1, "monthly_quota": 2000}); err != nil {
		t.Fatalf("更新 brave 失败: %v", err)
	}
	if err := module.service.repo.updateProvider(ctx, "tavily", map[string]any{"priority": 2, "monthly_quota": 1000}); err != nil {
		t.Fatalf("更新 tavily 失败: %v", err)
	}
	if err := module.service.repo.addUsage(ctx, providerSubject("brave"), currentPeriod(module.service.now()), 1990, 1990, 0); err != nil {
		t.Fatalf("写入用量失败: %v", err)
	}
	if err := module.service.Reload(ctx); err != nil {
		t.Fatalf("Reload 失败: %v", err)
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	search := func(query string) string {
		recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"`+query+`"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
		}
		return decodeSearch(t, recorder).Provider
	}

	if got := search("balanced 模式"); got != "tavily" {
		t.Errorf("负载均衡下 provider = %q, 期望剩余配额多的 tavily", got)
	}

	if err := module.service.SetStrategy(ctx, StrategyPriority); err != nil {
		t.Fatalf("SetStrategy 返回错误: %v", err)
	}
	if got := search("priority 模式"); got != "brave" {
		t.Errorf("按优先级下 provider = %q, 期望优先级更高的 brave", got)
	}

	// 模型调度未接入，行为应与负载均衡一致
	if err := module.service.SetStrategy(ctx, StrategyModel); err != nil {
		t.Fatalf("SetStrategy 返回错误: %v", err)
	}
	if got := search("model 模式"); got != "tavily" {
		t.Errorf("模型调度未接入时 provider = %q, 应回落到负载均衡的结果", got)
	}
}
