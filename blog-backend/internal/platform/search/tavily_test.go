package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestTavilyRequestMapping(t *testing.T) {
	tests := []struct {
		name    string
		options TavilyOptions
		request Request
		check   func(*testing.T, tavilyRequest)
	}{
		{
			name:    "缺省深度与分块",
			request: Request{Query: "go 1.25"},
			check: func(t *testing.T, body tavilyRequest) {
				if body.SearchDepth != "basic" || body.ChunksPerSource != 3 {
					t.Errorf("深度/分块 = %q/%d, 期望 basic/3", body.SearchDepth, body.ChunksPerSource)
				}
				if body.MaxResults != 5 {
					t.Errorf("MaxResults = %d, 期望缺省 5", body.MaxResults)
				}
				if body.Topic != TopicGeneral {
					t.Errorf("Topic = %q, 期望 general", body.Topic)
				}
				if !body.IncludeUsage {
					t.Error("应请求 usage 以便记账")
				}
			},
		},
		{
			name:    "非法深度回落到 basic",
			options: TavilyOptions{SearchDepth: "turbo", ChunksPerSource: 9},
			request: Request{Query: "x"},
			check: func(t *testing.T, body tavilyRequest) {
				if body.SearchDepth != "basic" || body.ChunksPerSource != 3 {
					t.Errorf("深度/分块 = %q/%d, 期望回落到 basic/3", body.SearchDepth, body.ChunksPerSource)
				}
			},
		},
		{
			name:    "ultra-fast 不发送分块参数",
			options: TavilyOptions{SearchDepth: "ultra-fast"},
			request: Request{Query: "x"},
			check: func(t *testing.T, body tavilyRequest) {
				if body.ChunksPerSource != 0 {
					t.Errorf("ChunksPerSource = %d, ultra-fast 下不应发送", body.ChunksPerSource)
				}
			},
		},
		{
			name:    "结果数上限被截断",
			request: Request{Query: "x", MaxResults: 99},
			check: func(t *testing.T, body tavilyRequest) {
				if body.MaxResults != 20 {
					t.Errorf("MaxResults = %d, 期望截断到 20", body.MaxResults)
				}
			},
		},
		{
			name:    "时效性映射为 time_range",
			request: Request{Query: "x", Freshness: FreshnessMonth},
			check: func(t *testing.T, body tavilyRequest) {
				if body.TimeRange != "month" || body.StartDate != "" {
					t.Errorf("TimeRange/StartDate = %q/%q", body.TimeRange, body.StartDate)
				}
			},
		},
		{
			name:    "自定义区间映射为起止日期",
			request: Request{Query: "x", Freshness: "2026-01-01to2026-06-30"},
			check: func(t *testing.T, body tavilyRequest) {
				if body.StartDate != "2026-01-01" || body.EndDate != "2026-06-30" || body.TimeRange != "" {
					t.Errorf("区间映射错误: %+v", body)
				}
			},
		},
		{
			name:    "国家码翻译为英文名",
			request: Request{Query: "x", Country: "cn"},
			check: func(t *testing.T, body tavilyRequest) {
				if body.Country != "china" {
					t.Errorf("Country = %q, 期望 china", body.Country)
				}
			},
		},
		{
			name:    "未知国家码被丢弃",
			request: Request{Query: "x", Country: "ZZ"},
			check: func(t *testing.T, body tavilyRequest) {
				if body.Country != "" {
					t.Errorf("Country = %q, 未知码应丢弃", body.Country)
				}
			},
		},
		{
			name:    "新闻主题不发送国家",
			request: Request{Query: "x", Country: "cn", Topic: TopicNews},
			check: func(t *testing.T, body tavilyRequest) {
				if body.Country != "" {
					t.Errorf("Country = %q, news 主题下不支持", body.Country)
				}
			},
		},
		{
			name:    "域名过滤走原生字段并归一化",
			request: Request{Query: "x", IncludeDomains: []string{"https://www.go.dev/doc"}, ExcludeDomains: []string{"CSDN.NET"}},
			check: func(t *testing.T, body tavilyRequest) {
				if !reflect.DeepEqual(body.IncludeDomains, []string{"go.dev"}) {
					t.Errorf("IncludeDomains = %v", body.IncludeDomains)
				}
				if !reflect.DeepEqual(body.ExcludeDomains, []string{"csdn.net"}) {
					t.Errorf("ExcludeDomains = %v", body.ExcludeDomains)
				}
			},
		},
		{
			name:    "答案与正文透传",
			request: Request{Query: "x", IncludeAnswer: true, IncludeRawContent: true},
			check: func(t *testing.T, body tavilyRequest) {
				if !body.IncludeAnswer || !body.IncludeRawContent {
					t.Errorf("答案/正文标志未透传: %+v", body)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(t, NewTavily("key", "", test.options, nil).body(test.request))
		})
	}
}

func TestTavilySearchNormalizesResponse(t *testing.T) {
	const body = `{
	  "query": "go 1.25",
	  "answer": "  Go 1.25 引入了……  ",
	  "results": [
	    {"title":"A","url":"https://a.dev","content":"c a","score":0.81,"raw_content":"full a",
	     "published_date":"2026-06-12T00:00:00Z"},
	    {"title":"B","url":"https://b.dev","content":"c b","score":0.42}
	  ],
	  "usage": {"credits": 2}
	}`

	var captured tavilyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Errorf("鉴权头 = %q, 期望 Bearer key", got)
		}
		if r.Method != http.MethodPost {
			t.Errorf("方法 = %q, 期望 POST", r.Method)
		}
		payload, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(payload, &captured)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	provider := NewTavily("key", server.URL, TavilyOptions{}, server.Client())
	got, err := provider.Search(context.Background(), Request{Query: "go 1.25", MaxResults: 5, IncludeAnswer: true})
	if err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}

	if captured.Query != "go 1.25" {
		t.Errorf("上游收到的 query = %q", captured.Query)
	}
	if got.Answer != "Go 1.25 引入了……" {
		t.Errorf("Answer = %q, 期望去掉首尾空白", got.Answer)
	}
	if got.Credits != 2 {
		t.Errorf("Credits = %d, 期望取自 usage", got.Credits)
	}
	if len(got.Results) != 2 {
		t.Fatalf("结果数 = %d", len(got.Results))
	}
	if got.Results[0].Score != 0.81 {
		t.Errorf("Score = %v, 期望取自上游而非名次", got.Results[0].Score)
	}
	if got.Results[0].RawContent != "full a" {
		t.Errorf("RawContent = %q", got.Results[0].RawContent)
	}
	if got.Results[0].PublishedAt == nil {
		t.Error("PublishedAt 应被解析")
	}
}

func TestTavilyCreditsFallBackToDepthPrice(t *testing.T) {
	tests := []struct {
		depth string
		want  int
	}{
		{"basic", 1},
		{"advanced", 2},
	}
	for _, test := range tests {
		t.Run(test.depth, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"query":"x","results":[]}`))
			}))
			defer server.Close()

			provider := NewTavily("key", server.URL, TavilyOptions{SearchDepth: test.depth}, server.Client())
			got, err := provider.Search(context.Background(), Request{Query: "x"})
			if err != nil {
				t.Fatalf("Search 返回错误: %v", err)
			}
			if got.Credits != test.want {
				t.Errorf("Credits = %d, 期望 %d", got.Credits, test.want)
			}
		})
	}
}

func TestTavilySearchErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantKind  ErrorKind
		wantMsg   string
		retryable bool
	}{
		{"鉴权失败", http.StatusUnauthorized, `{"detail":{"error":"missing key"}}`, KindAuthFailed, "missing key", true},
		{"参数错误", http.StatusBadRequest, `{"detail":{"error":"invalid topic"}}`, KindBadRequest, "invalid topic", false},
		{"限速", http.StatusTooManyRequests, `{"detail":{"error":"slow down"}}`, KindRateLimited, "slow down", true},
		{"套餐额度耗尽", 432, `{"detail":{"error":"plan limit"}}`, KindQuotaExceeded, "plan limit", true},
		{"按量额度耗尽", 433, `{"detail":{"error":"paygo limit"}}`, KindQuotaExceeded, "paygo limit", true},
		{"上游故障", http.StatusInternalServerError, `boom`, KindUnavailable, "boom", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			provider := NewTavily("key", server.URL, TavilyOptions{}, server.Client())
			_, err := provider.Search(context.Background(), Request{Query: "x"})

			searchErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("错误类型 = %T, 期望 *search.Error", err)
			}
			if searchErr.Kind != test.wantKind {
				t.Errorf("Kind = %q, 期望 %q", searchErr.Kind, test.wantKind)
			}
			if searchErr.Message != test.wantMsg {
				t.Errorf("Message = %q, 期望 %q", searchErr.Message, test.wantMsg)
			}
			if searchErr.Retryable() != test.retryable {
				t.Errorf("Retryable = %v, 期望 %v", searchErr.Retryable(), test.retryable)
			}
		})
	}
}

func TestTavilySearchRequiresAPIKey(t *testing.T) {
	_, err := NewTavily("", "", TavilyOptions{}, nil).Search(context.Background(), Request{Query: "x"})
	searchErr, ok := err.(*Error)
	if !ok || searchErr.Kind != KindAuthFailed {
		t.Fatalf("未配置 Key 时应返回 auth 错误, 得到 %v", err)
	}
}

func TestCapabilitiesDifferBetweenProviders(t *testing.T) {
	brave := NewBrave("k", "", nil).Capabilities()
	tavily := NewTavily("k", "", TavilyOptions{}, nil).Capabilities()

	if brave.Answer || brave.RawContent || brave.DomainFilter {
		t.Errorf("Brave 不具备答案/正文/域名过滤能力: %+v", brave)
	}
	if !brave.SearchOperators || !brave.Pagination {
		t.Errorf("Brave 应支持搜索操作符与分页: %+v", brave)
	}
	if !tavily.Answer || !tavily.RawContent || !tavily.DomainFilter {
		t.Errorf("Tavily 应具备答案/正文/域名过滤能力: %+v", tavily)
	}
	if tavily.SearchOperators || tavily.Pagination {
		t.Errorf("Tavily 不支持搜索操作符与分页: %+v", tavily)
	}
}
