package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestExa(options ExaOptions, base string, client *http.Client) *ExaProvider {
	provider := NewExa("key", base, options, client)
	// 固定"现在"，让基于发布日期的时效性映射可断言
	provider.now = func() time.Time { return time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC) }
	return provider
}

func TestExaRequestMapping(t *testing.T) {
	tests := []struct {
		name    string
		options ExaOptions
		request Request
		check   func(*testing.T, exaRequest)
	}{
		{
			name:    "缺省搜索类型与结果数",
			request: Request{Query: "go 1.25"},
			check: func(t *testing.T, body exaRequest) {
				if body.Type != "auto" {
					t.Errorf("Type = %q, 期望 auto", body.Type)
				}
				if body.NumResults != 5 {
					t.Errorf("NumResults = %d, 期望缺省 5", body.NumResults)
				}
			},
		},
		{
			name:    "非法搜索类型回落到 auto",
			options: ExaOptions{SearchType: "turbo"},
			request: Request{Query: "x"},
			check: func(t *testing.T, body exaRequest) {
				if body.Type != "auto" {
					t.Errorf("Type = %q, 期望回落到 auto", body.Type)
				}
			},
		},
		{
			name:    "自定义搜索类型透传",
			options: ExaOptions{SearchType: "fast"},
			request: Request{Query: "x"},
			check: func(t *testing.T, body exaRequest) {
				if body.Type != "fast" {
					t.Errorf("Type = %q", body.Type)
				}
			},
		},
		{
			name:    "始终请求 highlights",
			request: Request{Query: "x"},
			check: func(t *testing.T, body exaRequest) {
				// Exa 不带 contents 就没有任何摘要，结果对 agent 毫无用处
				if body.Contents == nil || !body.Contents.Highlights {
					t.Errorf("Contents = %+v, 应始终请求 highlights", body.Contents)
				}
				if body.Contents.Text {
					t.Error("未请求正文时不应拉取 text")
				}
			},
		},
		{
			name:    "需要正文时附带 text",
			request: Request{Query: "x", IncludeRawContent: true},
			check: func(t *testing.T, body exaRequest) {
				if !body.Contents.Text {
					t.Error("IncludeRawContent 应请求 text")
				}
			},
		},
		{
			name:    "新闻主题映射为 category",
			request: Request{Query: "x", Topic: TopicNews},
			check: func(t *testing.T, body exaRequest) {
				if body.Category != "news" {
					t.Errorf("Category = %q, 期望 news", body.Category)
				}
			},
		},
		{
			name:    "域名过滤走原生字段",
			request: Request{Query: "x", IncludeDomains: []string{"https://www.go.dev/doc"}, ExcludeDomains: []string{"CSDN.NET"}},
			check: func(t *testing.T, body exaRequest) {
				if !reflect.DeepEqual(body.IncludeDomains, []string{"go.dev"}) {
					t.Errorf("IncludeDomains = %v", body.IncludeDomains)
				}
				if !reflect.DeepEqual(body.ExcludeDomains, []string{"csdn.net"}) {
					t.Errorf("ExcludeDomains = %v", body.ExcludeDomains)
				}
			},
		},
		{
			name:    "结果数上限被截断",
			request: Request{Query: "x", MaxResults: 500},
			check: func(t *testing.T, body exaRequest) {
				if body.NumResults != exaMaxResults {
					t.Errorf("NumResults = %d, 期望截断到 %d", body.NumResults, exaMaxResults)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(t, newTestExa(test.options, "", nil).body(test.request))
		})
	}
}

func TestExaFreshnessBecomesPublishDates(t *testing.T) {
	// Exa 没有相对时效参数，只有绝对发布日期区间
	tests := []struct {
		freshness string
		wantStart string
		wantEnd   string
	}{
		{FreshnessDay, "2026-07-31T12:00:00.000Z", ""},
		{FreshnessWeek, "2026-07-25T12:00:00.000Z", ""},
		{FreshnessMonth, "2026-07-02T12:00:00.000Z", ""},
		{FreshnessYear, "2025-08-01T12:00:00.000Z", ""},
		{"2026-01-01to2026-06-30", "2026-01-01T00:00:00.000Z", "2026-06-30T23:59:59.999Z"},
		{"yesterday", "", ""},
		{"", "", ""},
	}

	provider := newTestExa(ExaOptions{}, "", nil)
	for _, test := range tests {
		t.Run(test.freshness, func(t *testing.T) {
			body := provider.body(Request{Query: "x", Freshness: test.freshness})
			if body.StartPublishedDate != test.wantStart {
				t.Errorf("StartPublishedDate = %q, 期望 %q", body.StartPublishedDate, test.wantStart)
			}
			if body.EndPublishedDate != test.wantEnd {
				t.Errorf("EndPublishedDate = %q, 期望 %q", body.EndPublishedDate, test.wantEnd)
			}
		})
	}
}

func TestExaSearchNormalizesResponse(t *testing.T) {
	const body = `{
	  "requestId": "abc",
	  "results": [
	    {"title":"A","url":"https://a.dev","publishedDate":"2026-06-12T01:36:32.547Z",
	     "highlights":["片段一","片段二"],"text":"完整正文 A"},
	    {"title":"B","url":"https://b.dev","summary":"只有摘要"}
	  ],
	  "costDollars": {"total": 0.007}
	}`

	var captured exaRequest
	var capturedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("x-api-key")
		payload, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(payload, &captured)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	provider := newTestExa(ExaOptions{}, server.URL, server.Client())
	got, err := provider.Search(context.Background(), Request{Query: "go 1.25", MaxResults: 5, IncludeRawContent: true})
	if err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}

	if capturedKey != "key" {
		t.Errorf("鉴权头 = %q, Exa 用 x-api-key", capturedKey)
	}
	if captured.Query != "go 1.25" {
		t.Errorf("上游 query = %q", captured.Query)
	}
	if len(got.Results) != 2 {
		t.Fatalf("结果数 = %d", len(got.Results))
	}
	if got.Results[0].Content != "片段一 … 片段二" {
		t.Errorf("Content = %q, 期望由 highlights 拼接", got.Results[0].Content)
	}
	if got.Results[1].Content != "只有摘要" {
		t.Errorf("无 highlights 时应回落到 summary, 得到 %q", got.Results[1].Content)
	}
	if got.Results[0].RawContent != "完整正文 A" {
		t.Errorf("RawContent = %q", got.Results[0].RawContent)
	}
	if got.Results[0].Score <= got.Results[1].Score {
		t.Error("Exa 不返回相关性分数，应按名次递减生成")
	}
	if got.Results[0].PublishedAt == nil || got.Results[0].PublishedAt.Year() != 2026 {
		t.Errorf("PublishedAt = %v", got.Results[0].PublishedAt)
	}
	if got.Credits != 1 {
		t.Errorf("Credits = %d, 期望按次记 1", got.Credits)
	}
	// Exa 按美元计价，网关用微美元保留精度
	if got.CostMicroUSD != 7000 {
		t.Errorf("CostMicroUSD = %d, 期望 $0.007 → 7000", got.CostMicroUSD)
	}
}

func TestExaSnippetFallsBackToTruncatedText(t *testing.T) {
	long := strings.Repeat("字", 900)
	got := exaSnippet(nil, "", long)
	if len([]rune(got)) != 500 {
		t.Fatalf("回落到正文时应截断到 500 字符, 得到 %d", len([]rune(got)))
	}
}

func TestExaSearchErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantKind  ErrorKind
		retryable bool
	}{
		{"鉴权失败", http.StatusUnauthorized, `{"error":"bad key"}`, KindAuthFailed, true},
		{"余额不足", exaPaymentRequired, `{"error":"insufficient credit"}`, KindQuotaExceeded, true},
		{"参数错误", http.StatusBadRequest, `{"error":"bad query"}`, KindBadRequest, false},
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

			_, err := newTestExa(ExaOptions{}, server.URL, server.Client()).
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

func TestExaCapabilities(t *testing.T) {
	capability := NewExa("k", "", ExaOptions{}, nil).Capabilities()
	// Exa 是语义检索：不支持搜索操作符，也没有结果分页
	if capability.SearchOperators || capability.Pagination {
		t.Errorf("Exa 不支持操作符与分页: %+v", capability)
	}
	if !capability.RawContent || !capability.DomainFilter {
		t.Errorf("Exa 支持正文与域名过滤: %+v", capability)
	}
	// 只有 deep 系列模式才综合答案，网关不承诺
	if capability.Answer {
		t.Error("Exa 的普通搜索不返回答案，不应声明该能力")
	}
}

func TestExaForwardReadsCost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "key" {
			t.Errorf("鉴权头 = %q", got)
		}
		_, _ = w.Write([]byte(`{"results":[],"costDollars":{"total":0.025}}`))
	}))
	defer server.Close()

	response, err := newTestExa(ExaOptions{}, server.URL, server.Client()).
		Forward(context.Background(), PassthroughRequest{Method: http.MethodPost, Path: "/search", Body: []byte(`{"query":"x"}`)})
	if err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}
	if response.CostMicroUSD != 25000 {
		t.Errorf("CostMicroUSD = %d, 期望 $0.025 → 25000", response.CostMicroUSD)
	}
	if response.Credits != 1 {
		t.Errorf("Credits = %d", response.Credits)
	}
}

func TestMetaForKnownAndUnknownProviders(t *testing.T) {
	for _, name := range []string{ProviderBrave, ProviderTavily, ProviderExa, ProviderFirecrawl} {
		meta := MetaFor(name)
		if meta.HomeURL == "" || meta.DocsURL == "" || meta.ConsoleURL == "" || meta.Billing == "" {
			t.Errorf("%s 的元信息不完整: %+v", name, meta)
		}
		for _, link := range []string{meta.HomeURL, meta.DocsURL, meta.ConsoleURL} {
			if !strings.HasPrefix(link, "https://") {
				t.Errorf("%s 的链接必须是 https: %q", name, link)
			}
		}
	}
	if meta := MetaFor("unknown"); meta.Name != "unknown" || meta.DisplayName != "unknown" {
		t.Errorf("未知供应商应回落到名称本身: %+v", meta)
	}
}
