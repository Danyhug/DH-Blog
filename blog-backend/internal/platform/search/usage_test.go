package search

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// tavilyUsageServer serves one canned /usage payload and records the token it
// was called with.
func tavilyUsageServer(t *testing.T, status int, body string, token *string) *TavilyProvider {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usage" {
			t.Errorf("请求路径 = %s, 期望 /usage", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("请求方法 = %s, 期望 GET", r.Method)
		}
		if token != nil {
			*token = r.Header.Get("Authorization")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return NewTavily("tvly-test", server.URL, TavilyOptions{}, nil)
}

func TestTavilyUsageUsesKeyLimitWhenSet(t *testing.T) {
	var token string
	provider := tavilyUsageServer(t, http.StatusOK, `{
		"key": {"usage": 150, "limit": 1000, "search_usage": 150},
		"account": {"current_plan": "Bootstrap", "plan_usage": 500, "plan_limit": 15000}
	}`, &token)

	report, err := provider.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage 返回错误: %v", err)
	}
	if token != "Bearer tvly-test" {
		t.Errorf("Authorization = %q, 期望复用搜索用的同一把密钥", token)
	}
	if report.Used != 150 || report.Limit != 1000 {
		t.Errorf("用量 = %d/%d, 期望 150/1000", report.Used, report.Limit)
	}
	if report.Scope != UsageScopeKey {
		t.Errorf("Scope = %q, 期望 %q", report.Scope, UsageScopeKey)
	}
	if report.Unit != UsageUnitCredit {
		t.Errorf("Unit = %q, 期望 %q（Tavily 按 credit 计费，不是按请求数）", report.Unit, UsageUnitCredit)
	}
	if report.Exhausted() {
		t.Error("150/1000 不应判为用尽")
	}
}

func TestTavilyUsageFallsBackToPlanWhenKeyHasNoLimit(t *testing.T) {
	// limit 为 null 表示这把密钥自己没有上限，真正拦住它的是所属套餐
	provider := tavilyUsageServer(t, http.StatusOK, `{
		"key": {"usage": 7, "limit": null},
		"account": {"current_plan": "Researcher", "plan_usage": 7, "plan_limit": 1000}
	}`, nil)

	report, err := provider.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage 返回错误: %v", err)
	}
	if report.Used != 7 || report.Limit != 1000 {
		t.Errorf("用量 = %d/%d, 期望回落到套餐额度 7/1000", report.Used, report.Limit)
	}
	if report.Scope != UsageScopeAccount {
		t.Errorf("Scope = %q, 期望 %q，好让页面说清这是账户共享额度", report.Scope, UsageScopeAccount)
	}
}

func TestTavilyUsageReportsUnlimitedAsNoCeiling(t *testing.T) {
	provider := tavilyUsageServer(t, http.StatusOK, `{
		"key": {"usage": 40, "limit": null},
		"account": {"plan_usage": 40, "plan_limit": null}
	}`, nil)

	report, err := provider.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage 返回错误: %v", err)
	}
	if report.Limit != 0 {
		t.Errorf("Limit = %d, 期望 0（无上限）", report.Limit)
	}
	if report.Exhausted() {
		t.Error("没有上限时不该判为用尽，否则会把好密钥停掉")
	}
}

func TestTavilyUsageClassifiesRejection(t *testing.T) {
	provider := tavilyUsageServer(t, http.StatusUnauthorized, `{"detail":{"error":"Invalid API key"}}`, nil)

	_, err := provider.Usage(context.Background())
	if err == nil {
		t.Fatal("期望返回错误")
	}
	var providerErr *Error
	if !errors.As(err, &providerErr) {
		t.Fatalf("错误类型 = %T, 期望 *search.Error", err)
	}
	if providerErr.Kind != KindAuthFailed {
		t.Errorf("Kind = %q, 期望 %q，同步时才能据此停掉被吊销的密钥", providerErr.Kind, KindAuthFailed)
	}
}

// braveWithHeaders answers one search carrying the given rate-limit headers.
func braveWithHeaders(t *testing.T, headers map[string]string) *BraveProvider {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for name, value := range headers {
			w.Header().Set(name, value)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"web":{"results":[]}}`))
	}))
	t.Cleanup(server.Close)
	return NewBrave("brave-test", server.URL, nil)
}

func TestBraveUsageComesFromResponseHeaders(t *testing.T) {
	provider := braveWithHeaders(t, map[string]string{
		"X-RateLimit-Limit":     "1, 15000",
		"X-RateLimit-Remaining": "1, 12000",
		"X-RateLimit-Policy":    "1;w=1, 15000;w=2592000",
	})

	if _, err := provider.Usage(context.Background()); !errors.Is(err, ErrUsageUnavailable) {
		t.Fatalf("首次调用前 Usage 错误 = %v, 期望 ErrUsageUnavailable", err)
	}

	if _, err := provider.Search(context.Background(), Request{Query: "x"}); err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}

	report, err := provider.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage 返回错误: %v", err)
	}
	// Brave 只在响应头里报额度，用量得由 limit 减 remaining 反推
	if report.Used != 3000 || report.Limit != 15000 {
		t.Errorf("用量 = %d/%d, 期望 3000/15000", report.Used, report.Limit)
	}
	if report.Unit != UsageUnitRequest {
		t.Errorf("Unit = %q, 期望 %q", report.Unit, UsageUnitRequest)
	}
	if report.Window != braveQuotaWindow {
		t.Errorf("Window = %q, 期望 %q（滚动 30 天，不是自然月）", report.Window, braveQuotaWindow)
	}
}

func TestBraveUsageIgnoresUnlimitedPlan(t *testing.T) {
	// 0 表示不限量；把它当成上限会让密钥立刻显示为用尽
	provider := braveWithHeaders(t, map[string]string{
		"X-RateLimit-Limit":     "1, 0",
		"X-RateLimit-Remaining": "1, 0",
	})
	if _, err := provider.Search(context.Background(), Request{Query: "x"}); err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}
	if _, err := provider.Usage(context.Background()); !errors.Is(err, ErrUsageUnavailable) {
		t.Fatalf("Usage 错误 = %v, 期望 ErrUsageUnavailable", err)
	}
}

func TestBraveUsageSurvivesRejectedSearch(t *testing.T) {
	// 配额耗尽时上游返回 429，但额度头还在——正是最该记下来的一次
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "1, 2000")
		w.Header().Set("X-RateLimit-Remaining", "0, 0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED"}}`))
	}))
	t.Cleanup(server.Close)
	provider := NewBrave("brave-test", server.URL, nil)

	if _, err := provider.Search(context.Background(), Request{Query: "x"}); err == nil {
		t.Fatal("期望搜索失败")
	}
	report, err := provider.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage 返回错误: %v", err)
	}
	if !report.Exhausted() {
		t.Errorf("用量 = %d/%d, 期望判为用尽", report.Used, report.Limit)
	}
}

func TestLastRateLimitValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
		ok    bool
	}{
		{name: "两个窗口取最长的", value: "1, 15000", want: 15000, ok: true},
		{name: "带窗口长度", value: "1;w=1, 15000;w=2592000", want: 15000, ok: true},
		{name: "单个值", value: "500", want: 500, ok: true},
		{name: "空值", value: "", ok: false},
		{name: "非数字", value: "1, abc", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lastRateLimitValue(tt.value)
			if ok != tt.ok {
				t.Fatalf("ok = %v, 期望 %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("值 = %d, 期望 %d", got, tt.want)
			}
		})
	}
}

func TestExaDoesNotReportUsage(t *testing.T) {
	// Exa 的用量要另一把 service key 才能查，这里明确它不实现该接口，
	// 免得以后有人以为同步覆盖了三家
	var provider Provider = NewExa("exa-test", "", ExaOptions{}, nil)
	if _, ok := provider.(UsageReporter); ok {
		t.Error("Exa 不应实现 UsageReporter")
	}
}
