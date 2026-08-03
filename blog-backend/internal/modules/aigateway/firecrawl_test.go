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

// firecrawlOK answers a search with the v2 envelope; anything else (the credit
// endpoint) gets an empty allowance so the usage sync stays out of the way.
func firecrawlOK(creditsUsed int, titles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/team/credit-usage") {
			_, _ = w.Write([]byte(`{"success":true,"data":{"remainingCredits":100}}`))
			return
		}
		items := make([]map[string]any, 0, len(titles))
		for _, title := range titles {
			items = append(items, map[string]any{
				"title": title, "url": "https://" + title + ".dev", "description": "描述 " + title,
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":     true,
			"data":        map[string]any{"web": items},
			"creditsUsed": creditsUsed,
		})
	}
}

func TestGatewayRoutesToFirecrawlWhenSelected(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:     braveOK("b"),
		Firecrawl: firecrawlOK(2, "f1", "f2"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"抓正文","provider":"firecrawl","max_results":2}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	result := decodeSearch(t, recorder)
	if result.Provider != "firecrawl" {
		t.Fatalf("Provider = %q", result.Provider)
	}
	if len(result.Results) != 2 || result.Results[0].Content != "描述 f1" {
		t.Fatalf("结果异常: %+v", result.Results)
	}
	if result.Meta.Credits != 2 {
		t.Errorf("meta.credits = %d, 期望采用上游回报的 2", result.Meta.Credits)
	}
}

func TestGatewayAcceptsFirecrawlForRawContent(t *testing.T) {
	// Brave 没有正文能力，要正文时必须被能力过滤剔除
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:     braveOK("b1"),
		Firecrawl: firecrawlOK(3, "f1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go 泛型","include_raw_content":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decodeSearch(t, recorder); result.Provider != "firecrawl" {
		t.Fatalf("Provider = %q, 需要正文时不应路由到 brave", result.Provider)
	}
}

func TestGatewayKeepsAnswerRequestsAwayFromFirecrawl(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Firecrawl: firecrawlOK(2, "f1"),
		Tavily:    tavilyOK("这是答案", "t1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"什么是 RAG","include_answer":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if result := decodeSearch(t, recorder); result.Provider != "tavily" {
		t.Fatalf("Provider = %q, 需要答案时不应路由到 firecrawl", result.Provider)
	}
}

func TestGatewayFallsBackFromFirecrawlToOtherProvider(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Firecrawl: failing(http.StatusBadGateway, `{"success":false,"error":"boom"}`),
		Tavily:    tavilyOK("", "t1"),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go","provider":"firecrawl","allow_fallback":true}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	result := decodeSearch(t, recorder)
	if result.Provider != "tavily" || result.Meta.FallbackFrom != "firecrawl" {
		t.Fatalf("回退异常: provider=%q fallbackFrom=%q", result.Provider, result.Meta.FallbackFrom)
	}
}

func TestFirecrawlPassthroughReturnsNativeShape(t *testing.T) {
	const native = `{"success":true,"data":{"web":[{"title":"A","url":"https://a.dev","description":"d","markdown":"# A"}]},"creditsUsed":4}`

	var seen atomic.Value
	module := newGatewayTestModule(t, gatewayTestConfig{Firecrawl: echoUpstream(native, &seen, nil)})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/firecrawl/search", token,
		`{"query":"go","sources":["web"],"scrapeOptions":{"formats":[{"type":"markdown"}]}}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != native {
		t.Fatalf("响应体被改写: %s", recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Gateway-Provider"); got != "firecrawl" {
		t.Errorf("X-Gateway-Provider = %q", got)
	}

	upstream := lastSeen(t, &seen)
	if upstream["path"] != "/search" {
		t.Errorf("上游路径 = %q", upstream["path"])
	}
	if !strings.Contains(upstream["body"], `"scrapeOptions"`) {
		t.Errorf("原生字段应保留: %s", upstream["body"])
	}
}

func TestFirecrawlPassthroughStripsCallerCredential(t *testing.T) {
	var seen atomic.Value
	module := newGatewayTestModule(t, gatewayTestConfig{
		Firecrawl: echoUpstream(`{"success":true,"data":{"web":[]}}`, &seen, nil),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/firecrawl/search", token,
		`{"query":"go","apiKey":"fc-CALLER-SECRET","api_key":"another-SECRET"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	if body := lastSeen(t, &seen)["body"]; strings.Contains(body, "SECRET") {
		t.Fatalf("调用方自带的凭证不应被转发: %s", body)
	}
}

func TestFirecrawlPassthroughAccountsCredits(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Firecrawl: echoUpstream(`{"success":true,"data":{"web":[]},"creditsUsed":8}`, nil, nil),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/firecrawl/search", token,
		`{"query":"go"}`); recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	module.Shutdown()

	ctx := context.Background()
	usage, err := module.service.repo.usageFor(ctx, currentPeriod(time.Now()), []string{providerSubject("firecrawl")})
	if err != nil {
		t.Fatalf("usageFor 返回错误: %v", err)
	}
	if got := usage[providerSubject("firecrawl")]; got.Count != 1 || got.Credits != 8 {
		t.Fatalf("用量 = %+v, 期望 1 次 / 8 credit", got)
	}

	logs, _, err := module.service.repo.listLogs(ctx, logFilter{})
	if err != nil {
		t.Fatalf("listLogs 返回错误: %v", err)
	}
	if logs[0].Endpoint != "firecrawl/search" || logs[0].Credits != 8 {
		t.Fatalf("日志内容异常: %+v", logs[0])
	}
}

func TestSyncUsageParksExhaustedFirecrawlKey(t *testing.T) {
	// 额度是 team 级的：上游说没了，这把密钥就该立刻退出轮换
	module := newGatewayTestModule(t, gatewayTestConfig{
		Firecrawl: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(r.URL.Path, "/team/credit-usage") {
				_, _ = w.Write([]byte(`{"success":true,"data":{"remainingCredits":0,"planCredits":3000}}`))
				return
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"web":[]}}`))
		},
	})
	runtime := module.service.runtime("firecrawl")
	if runtime == nil {
		t.Fatal("firecrawl 运行时缺失")
	}

	result := module.service.SyncUsage(context.Background())
	if result.Synced != 1 {
		t.Fatalf("Synced = %d, 期望 1", result.Synced)
	}
	if len(result.Parked) != 1 {
		t.Fatalf("Parked = %v, 期望停用 1 把密钥", result.Parked)
	}

	stored := keyRow(t, module, runtime.credentials()[0].id)
	if stored.UpstreamUsed != 3000 || stored.UpstreamLimit != 3000 {
		t.Errorf("上游用量 = %d/%d, 期望 3000/3000", stored.UpstreamUsed, stored.UpstreamLimit)
	}
	if stored.UpstreamScope != "account" {
		t.Errorf("Scope = %q, credit 挂在 team 上应记为 account", stored.UpstreamScope)
	}
	if runtime.usableKeys(module.service.now()) != 0 {
		t.Error("期望密钥立即退出轮换")
	}
}

func TestSearchRejectsUnknownProviderButAcceptsFirecrawl(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Firecrawl: firecrawlOK(2, "f1")})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"go","provider":"serper"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, 期望 400", recorder.Code)
	}
	if detail := decodeError(t, recorder); !strings.Contains(detail.Message, "firecrawl") {
		t.Errorf("错误信息未列出 firecrawl: %q", detail.Message)
	}
}
