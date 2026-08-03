package search

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func newTestFirecrawl(options FirecrawlOptions, base string, client *http.Client) *FirecrawlProvider {
	return NewFirecrawl("key", base, options, client)
}

func TestFirecrawlRequestMapping(t *testing.T) {
	tests := []struct {
		name    string
		options FirecrawlOptions
		request Request
		check   func(*testing.T, firecrawlRequest)
	}{
		{
			name:    "缺省来源与结果数",
			request: Request{Query: "go 1.25"},
			check: func(t *testing.T, body firecrawlRequest) {
				if !reflect.DeepEqual(body.Sources, []string{firecrawlSourceWeb}) {
					t.Errorf("Sources = %v, 期望 web", body.Sources)
				}
				if body.Limit != 5 {
					t.Errorf("Limit = %d, 期望缺省 5", body.Limit)
				}
			},
		},
		{
			name:    "不要正文时不带 scrapeOptions",
			request: Request{Query: "x"},
			check: func(t *testing.T, body firecrawlRequest) {
				// 每抓一页多花 1 credit，没人要正文时不该白花
				if body.ScrapeOptions != nil {
					t.Errorf("ScrapeOptions = %+v, 未请求正文时应为空", body.ScrapeOptions)
				}
			},
		},
		{
			name:    "需要正文时按配置的格式抓取",
			request: Request{Query: "x", IncludeRawContent: true},
			check: func(t *testing.T, body firecrawlRequest) {
				if body.ScrapeOptions == nil || len(body.ScrapeOptions.Formats) != 1 {
					t.Fatalf("ScrapeOptions = %+v", body.ScrapeOptions)
				}
				if body.ScrapeOptions.Formats[0].Type != "markdown" {
					t.Errorf("抓取格式 = %q, 期望缺省 markdown", body.ScrapeOptions.Formats[0].Type)
				}
			},
		},
		{
			name:    "自定义抓取格式透传",
			options: FirecrawlOptions{ScrapeFormat: "summary"},
			request: Request{Query: "x", IncludeRawContent: true},
			check: func(t *testing.T, body firecrawlRequest) {
				if body.ScrapeOptions.Formats[0].Type != "summary" {
					t.Errorf("抓取格式 = %q", body.ScrapeOptions.Formats[0].Type)
				}
			},
		},
		{
			name:    "非法抓取格式回落到 markdown",
			options: FirecrawlOptions{ScrapeFormat: "pdf"},
			request: Request{Query: "x", IncludeRawContent: true},
			check: func(t *testing.T, body firecrawlRequest) {
				if body.ScrapeOptions.Formats[0].Type != "markdown" {
					t.Errorf("抓取格式 = %q, 期望回落到 markdown", body.ScrapeOptions.Formats[0].Type)
				}
			},
		},
		{
			name:    "新闻主题切换来源",
			request: Request{Query: "x", Topic: TopicNews},
			check: func(t *testing.T, body firecrawlRequest) {
				if !reflect.DeepEqual(body.Sources, []string{firecrawlSourceNews}) {
					t.Errorf("Sources = %v, 期望 news", body.Sources)
				}
			},
		},
		{
			name:    "域名过滤走原生字段",
			request: Request{Query: "x", IncludeDomains: []string{"https://www.go.dev/doc"}, ExcludeDomains: []string{"CSDN.NET"}},
			check: func(t *testing.T, body firecrawlRequest) {
				if !reflect.DeepEqual(body.IncludeDomains, []string{"go.dev"}) {
					t.Errorf("IncludeDomains = %v", body.IncludeDomains)
				}
				if !reflect.DeepEqual(body.ExcludeDomains, []string{"csdn.net"}) {
					t.Errorf("ExcludeDomains = %v", body.ExcludeDomains)
				}
			},
		},
		{
			name:    "国家代码统一大写",
			request: Request{Query: "x", Country: "cn"},
			check: func(t *testing.T, body firecrawlRequest) {
				if body.Country != "CN" {
					t.Errorf("Country = %q, 期望 CN", body.Country)
				}
			},
		},
		{
			name:    "非法国家代码被丢弃",
			request: Request{Query: "x", Country: "china"},
			check: func(t *testing.T, body firecrawlRequest) {
				if body.Country != "" {
					t.Errorf("Country = %q, 期望丢弃", body.Country)
				}
			},
		},
		{
			name:    "结果数上限被截断",
			request: Request{Query: "x", MaxResults: 500},
			check: func(t *testing.T, body firecrawlRequest) {
				if body.Limit != firecrawlMaxResults {
					t.Errorf("Limit = %d, 期望截断到 %d", body.Limit, firecrawlMaxResults)
				}
			},
		},
		{
			name:    "超长查询按上游上限截断",
			request: Request{Query: strings.Repeat("词", 900)},
			check: func(t *testing.T, body firecrawlRequest) {
				// 上游硬限 500 字符，超了整个请求会被拒，白跑一趟
				if length := len([]rune(body.Query)); length != firecrawlMaxQuery {
					t.Errorf("Query 长度 = %d, 期望截断到 %d", length, firecrawlMaxQuery)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(t, newTestFirecrawl(test.options, "", nil).body(test.request))
		})
	}
}

func TestFirecrawlFreshnessBecomesTimeBasedSearch(t *testing.T) {
	tests := []struct {
		freshness string
		want      string
	}{
		{FreshnessDay, "qdr:d"},
		{FreshnessWeek, "qdr:w"},
		{FreshnessMonth, "qdr:m"},
		{FreshnessYear, "qdr:y"},
		{"2024-12-01to2024-12-31", "cdr:1,cd_min:12/1/2024,cd_max:12/31/2024"},
		{"上周", ""},
		{"", ""},
	}

	provider := newTestFirecrawl(FirecrawlOptions{}, "", nil)
	for _, test := range tests {
		t.Run(test.freshness, func(t *testing.T) {
			if got := provider.body(Request{Query: "x", Freshness: test.freshness}).TBS; got != test.want {
				t.Errorf("TBS = %q, 期望 %q", got, test.want)
			}
		})
	}
}

func TestFirecrawlSearchNormalizesResponse(t *testing.T) {
	const body = `{
	  "success": true,
	  "data": {
	    "web": [
	      {"title":"A","url":"https://a.dev","description":"描述 A","markdown":"# 正文 A"},
	      {"title":"B","url":"https://b.dev","markdown":"正文 B 没有描述"}
	    ]
	  },
	  "creditsUsed": 2
	}`

	var captured firecrawlRequest
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		payload, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(payload, &captured)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	got, err := newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client()).
		Search(context.Background(), Request{Query: "go 1.25", MaxResults: 5, IncludeRawContent: true})
	if err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}

	if capturedAuth != "Bearer key" {
		t.Errorf("鉴权头 = %q, Firecrawl 用 Bearer", capturedAuth)
	}
	if captured.Query != "go 1.25" {
		t.Errorf("上游 query = %q", captured.Query)
	}
	if len(got.Results) != 2 {
		t.Fatalf("结果数 = %d", len(got.Results))
	}
	if got.Results[0].Content != "描述 A" {
		t.Errorf("Content = %q, 期望取 description", got.Results[0].Content)
	}
	if got.Results[1].Content != "正文 B 没有描述" {
		t.Errorf("无 description 时应回落到正文, 得到 %q", got.Results[1].Content)
	}
	if got.Results[0].RawContent != "# 正文 A" {
		t.Errorf("RawContent = %q", got.Results[0].RawContent)
	}
	if got.Results[0].Score <= got.Results[1].Score {
		t.Error("Firecrawl 不返回相关性分数，应按名次递减生成")
	}
	if got.Credits != 2 {
		t.Errorf("Credits = %d, 期望采用响应里的 creditsUsed", got.Credits)
	}
}

func TestFirecrawlReadsNewsResults(t *testing.T) {
	const body = `{
	  "success": true,
	  "data": {"news": [
	    {"title":"N","url":"https://n.dev","snippet":"新闻摘要","date":"2026-07-30"}
	  ]},
	  "creditsUsed": 2
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	got, err := newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client()).
		Search(context.Background(), Request{Query: "x", Topic: TopicNews})
	if err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}
	if len(got.Results) != 1 {
		t.Fatalf("结果数 = %d", len(got.Results))
	}
	if got.Results[0].Content != "新闻摘要" {
		t.Errorf("Content = %q, 新闻结果的摘要在 snippet 里", got.Results[0].Content)
	}
	if got.Results[0].PublishedAt == nil || got.Results[0].PublishedAt.Day() != 30 {
		t.Errorf("PublishedAt = %v", got.Results[0].PublishedAt)
	}
}

func TestFirecrawlTreatsSuccessFalseAsFailure(t *testing.T) {
	// 上游偶尔用 200 报错，当成空结果会既骗了调用方又照样计费
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error":"Request timed out"}`))
	}))
	defer server.Close()

	_, err := newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client()).
		Search(context.Background(), Request{Query: "x"})
	if err == nil {
		t.Fatal("success=false 应作为错误返回")
	}
	if !strings.Contains(err.Error(), "Request timed out") {
		t.Errorf("错误信息 = %q, 应保留上游说明", err.Error())
	}
}

func TestFirecrawlCreditsFallback(t *testing.T) {
	tests := []struct {
		name     string
		reported int
		results  int
		want     int
	}{
		{"以上游回报为准", 7, 3, 7},
		{"缺省按 10 条一档", 0, 3, 2},
		{"11 条算两档", 0, 11, 4},
		{"空结果也要计一次", 0, 0, 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := firecrawlCredits(test.reported, test.results); got != test.want {
				t.Errorf("firecrawlCredits(%d, %d) = %d, 期望 %d", test.reported, test.results, got, test.want)
			}
		})
	}
}

func TestFirecrawlSearchErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantKind  ErrorKind
		retryable bool
	}{
		{"鉴权失败", http.StatusUnauthorized, `{"success":false,"error":"bad token"}`, KindAuthFailed, true},
		{"额度耗尽", firecrawlPaymentRequired, `{"success":false,"error":"out of credits"}`, KindQuotaExceeded, true},
		{"参数错误", http.StatusBadRequest, `{"success":false,"error":"bad query"}`, KindBadRequest, false},
		{"限速", http.StatusTooManyRequests, `{}`, KindRateLimited, true},
		{"上游故障", http.StatusBadGateway, `boom`, KindUnavailable, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			_, err := newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client()).
				Search(context.Background(), Request{Query: "x"})

			searchErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("错误类型 = %T", err)
			}
			if searchErr.Kind != test.wantKind {
				t.Errorf("Kind = %q, 期望 %q", searchErr.Kind, test.wantKind)
			}
			if searchErr.Retryable() != test.retryable {
				t.Errorf("Retryable = %v, 期望 %v", searchErr.Retryable(), test.retryable)
			}
		})
	}
}

func TestFirecrawlCapabilities(t *testing.T) {
	capability := NewFirecrawl("k", "", FirecrawlOptions{}, nil).Capabilities()
	if !capability.SearchOperators {
		t.Error("Firecrawl 支持 site:/filetype: 等操作符")
	}
	if !capability.RawContent || !capability.DomainFilter {
		t.Errorf("Firecrawl 支持正文与域名过滤: %+v", capability)
	}
	if capability.Answer {
		t.Error("Firecrawl 只返回网页，不综合答案")
	}
	if capability.Pagination {
		t.Error("Firecrawl 搜索没有翻页参数")
	}
}

func TestFirecrawlUsageReportsPlanBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/team/credit-usage" {
			t.Errorf("用量路径 = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"remainingCredits":420,"planCredits":3000}}`))
	}))
	defer server.Close()

	report, err := newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client()).
		Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage 返回错误: %v", err)
	}
	// 上游只报剩余，用量得自己算
	if report.Used != 2580 || report.Limit != 3000 {
		t.Errorf("用量 = %d/%d, 期望 2580/3000", report.Used, report.Limit)
	}
	if report.Unit != UsageUnitCredit {
		t.Errorf("Unit = %q", report.Unit)
	}
	// credit 挂在 team 上，换一把同账号的密钥并不会多出余量
	if report.Scope != UsageScopeAccount {
		t.Errorf("Scope = %q, 期望 %q", report.Scope, UsageScopeAccount)
	}
	if report.Exhausted() {
		t.Error("还有余量时不该判定为耗尽")
	}
}

func TestFirecrawlUsageWithoutPlanAllowanceIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"remainingCredits":12}}`))
	}))
	defer server.Close()

	_, err := newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client()).
		Usage(context.Background())
	// 没有额度总量就没有比较基准，编一个只会让选路按假数字行事
	if !errors.Is(err, ErrUsageUnavailable) {
		t.Fatalf("err = %v, 期望 ErrUsageUnavailable", err)
	}
}

func TestFirecrawlForwardReadsCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Errorf("鉴权头 = %q", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"web":[]},"creditsUsed":6}`))
	}))
	defer server.Close()

	response, err := newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client()).
		Forward(context.Background(), PassthroughRequest{Method: http.MethodPost, Path: "/search", Body: []byte(`{"query":"x"}`)})
	if err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}
	if response.Credits != 6 {
		t.Errorf("Credits = %d, 期望采用 creditsUsed", response.Credits)
	}
}
