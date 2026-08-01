package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBraveQueryMapping(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		want    map[string]string
	}{
		{
			name:    "基础参数",
			request: Request{Query: "go 1.25", MaxResults: 3},
			want:    map[string]string{"q": "go 1.25", "count": "3", "text_decorations": "0", "safesearch": "moderate"},
		},
		{
			name:    "结果数上限被截断",
			request: Request{Query: "x", MaxResults: 99},
			want:    map[string]string{"count": "20"},
		},
		{
			name:    "结果数缺省",
			request: Request{Query: "x"},
			want:    map[string]string{"count": "5"},
		},
		{
			name:    "新闻主题",
			request: Request{Query: "x", Topic: TopicNews},
			want:    map[string]string{"result_filter": "news"},
		},
		{
			name:    "时效性映射",
			request: Request{Query: "x", Freshness: FreshnessWeek},
			want:    map[string]string{"freshness": "pw"},
		},
		{
			name:    "自定义时间区间",
			request: Request{Query: "x", Freshness: "2026-01-01to2026-06-30"},
			want:    map[string]string{"freshness": "2026-01-01to2026-06-30"},
		},
		{
			name:    "非法时效性被丢弃",
			request: Request{Query: "x", Freshness: "yesterday"},
			want:    map[string]string{"freshness": ""},
		},
		{
			name:    "国家与语言",
			request: Request{Query: "x", Country: "cn", Language: "ZH"},
			want:    map[string]string{"country": "CN", "search_lang": "zh"},
		},
		{
			name:    "单个包含域名注入 site 操作符",
			request: Request{Query: "泛型", IncludeDomains: []string{"https://go.dev/doc"}},
			want:    map[string]string{"q": "泛型 site:go.dev"},
		},
		{
			name:    "多个包含域名用 OR 组合",
			request: Request{Query: "泛型", IncludeDomains: []string{"go.dev", "www.pkg.go.dev"}},
			want:    map[string]string{"q": "泛型 (site:go.dev OR site:pkg.go.dev)"},
		},
		{
			name:    "排除域名注入负号操作符",
			request: Request{Query: "泛型", ExcludeDomains: []string{"csdn.net"}},
			want:    map[string]string{"q": "泛型 -site:csdn.net"},
		},
		{
			name:    "正文请求降级为 extra_snippets",
			request: Request{Query: "x", IncludeRawContent: true},
			want:    map[string]string{"extra_snippets": "true"},
		},
	}

	provider := NewBrave("token", "", nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := provider.query(test.request)
			for key, want := range test.want {
				if got := values.Get(key); got != want {
					t.Errorf("参数 %s = %q, 期望 %q", key, got, want)
				}
			}
		})
	}
}

func TestBraveSearchNormalizesResponse(t *testing.T) {
	const body = `{
	  "query": {"original": "go 1.25", "more_results_available": true},
	  "web": {"results": [
	    {"title": "A", "url": "https://a.dev", "description": "desc a", "page_age": "2026-06-12T00:00:00",
	     "extra_snippets": ["s1", "s2"]},
	    {"title": "B", "url": "https://b.dev", "description": "desc b"}
	  ]}
	}`

	var captured *url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL
		if got := r.Header.Get("X-Subscription-Token"); got != "token" {
			t.Errorf("鉴权头 = %q, 期望 token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	provider := NewBrave("token", server.URL, server.Client())
	got, err := provider.Search(context.Background(), Request{Query: "go 1.25", MaxResults: 5, IncludeRawContent: true})
	if err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}

	if !strings.HasSuffix(captured.Path, "/web/search") {
		t.Errorf("请求路径 = %q, 期望以 /web/search 结尾", captured.Path)
	}
	if got.Query != "go 1.25" {
		t.Errorf("Query = %q", got.Query)
	}
	if got.Credits != 1 {
		t.Errorf("Credits = %d, Brave 应恒为 1", got.Credits)
	}
	if len(got.Results) != 2 {
		t.Fatalf("结果数 = %d, 期望 2", len(got.Results))
	}
	if got.Results[0].Content != "desc a" {
		t.Errorf("Content = %q, 期望取自 description", got.Results[0].Content)
	}
	if got.Results[0].RawContent != "s1\n\ns2" {
		t.Errorf("RawContent = %q, 期望由 extra_snippets 拼接", got.Results[0].RawContent)
	}
	if got.Results[0].Score <= got.Results[1].Score {
		t.Errorf("名次分应递减, 得到 %v 与 %v", got.Results[0].Score, got.Results[1].Score)
	}
	published := got.Results[0].PublishedAt
	if published == nil || published.Year() != 2026 || published.Month() != time.June {
		t.Errorf("PublishedAt = %v, 期望解析出 2026-06", published)
	}
	if got.Results[1].PublishedAt != nil {
		t.Errorf("缺少 page_age 时 PublishedAt 应为 nil")
	}
}

func TestBraveSearchTruncatesToMaxResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"web":{"results":[
		  {"title":"1","url":"https://1"},{"title":"2","url":"https://2"},{"title":"3","url":"https://3"}]}}`))
	}))
	defer server.Close()

	provider := NewBrave("token", server.URL, server.Client())
	got, err := provider.Search(context.Background(), Request{Query: "x", MaxResults: 2})
	if err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}
	if len(got.Results) != 2 {
		t.Fatalf("结果数 = %d, 期望被截断到 2", len(got.Results))
	}
}

func TestBraveSearchFallsBackToNewsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"web":{"results":[]},"news":{"results":[{"title":"N","url":"https://n"}]}}`))
	}))
	defer server.Close()

	provider := NewBrave("token", server.URL, server.Client())
	got, err := provider.Search(context.Background(), Request{Query: "x", Topic: TopicNews})
	if err != nil {
		t.Fatalf("Search 返回错误: %v", err)
	}
	if len(got.Results) != 1 || got.Results[0].Title != "N" {
		t.Fatalf("期望回落到 news 结果, 得到 %+v", got.Results)
	}
}

func TestBraveSearchErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantKind  ErrorKind
		retryable bool
	}{
		{"鉴权失败", http.StatusUnauthorized, `{"error":{"detail":"bad token"}}`, KindAuthFailed, true},
		{"参数错误", http.StatusBadRequest, `{"error":{"detail":"missing q"}}`, KindBadRequest, false},
		{"限速", http.StatusTooManyRequests, `{}`, KindRateLimited, true},
		{"上游故障", http.StatusBadGateway, `oops`, KindUnavailable, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			provider := NewBrave("token", server.URL, server.Client())
			_, err := provider.Search(context.Background(), Request{Query: "x"})

			searchErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("错误类型 = %T, 期望 *search.Error", err)
			}
			if searchErr.Kind != test.wantKind {
				t.Errorf("Kind = %q, 期望 %q", searchErr.Kind, test.wantKind)
			}
			if searchErr.Retryable() != test.retryable {
				t.Errorf("Retryable = %v, 期望 %v", searchErr.Retryable(), test.retryable)
			}
			if searchErr.Status != test.status {
				t.Errorf("Status = %d, 期望 %d", searchErr.Status, test.status)
			}
		})
	}
}

func TestBraveSearchRequiresAPIKey(t *testing.T) {
	_, err := NewBrave("", "", nil).Search(context.Background(), Request{Query: "x"})
	searchErr, ok := err.(*Error)
	if !ok || searchErr.Kind != KindAuthFailed {
		t.Fatalf("未配置 Key 时应返回 auth 错误, 得到 %v", err)
	}
}
