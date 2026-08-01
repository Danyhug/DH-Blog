package search

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBraveForwardReturnsUpstreamBodyVerbatim(t *testing.T) {
	const raw = `{"web":{"results":[{"title":"A","url":"https://a.dev"}]},"extra":"保持原样"}`

	var captured *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	provider := NewBrave("token", server.URL, server.Client())
	response, err := provider.Forward(context.Background(), PassthroughRequest{
		Method: http.MethodGet,
		Path:   "/web/search",
		Query:  url.Values{"q": []string{"go 1.25"}, "count": []string{"3"}},
	})
	if err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}

	if captured.URL.Path != "/web/search" {
		t.Errorf("上游路径 = %q", captured.URL.Path)
	}
	if got := captured.URL.Query().Get("q"); got != "go 1.25" {
		t.Errorf("上游 q = %q", got)
	}
	if got := captured.Header.Get("X-Subscription-Token"); got != "token" {
		t.Errorf("鉴权头 = %q, 网关应注入自己的凭证", got)
	}
	if string(response.Body) != raw {
		t.Errorf("响应体被改写: %s", response.Body)
	}
	if !response.OK() || response.Status != http.StatusOK {
		t.Errorf("Status = %d", response.Status)
	}
	if response.Credits != 1 {
		t.Errorf("Credits = %d, Brave 应恒为 1", response.Credits)
	}
	if response.ContentType != "application/json; charset=utf-8" {
		t.Errorf("ContentType = %q", response.ContentType)
	}
}

func TestTavilyForwardSendsBodyAndReadsCredits(t *testing.T) {
	const raw = `{"query":"go","results":[],"usage":{"credits":2}}`

	var capturedBody []byte
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		capturedAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	provider := NewTavily("key", server.URL, TavilyOptions{}, server.Client())
	body := []byte(`{"query":"go","search_depth":"advanced"}`)
	response, err := provider.Forward(context.Background(), PassthroughRequest{
		Method: http.MethodPost,
		Path:   "/search",
		Body:   body,
	})
	if err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}

	if string(capturedBody) != string(body) {
		t.Errorf("上游收到的请求体被改写: %s", capturedBody)
	}
	if capturedAuth != "Bearer key" {
		t.Errorf("鉴权头 = %q", capturedAuth)
	}
	if string(response.Body) != raw {
		t.Errorf("响应体被改写: %s", response.Body)
	}
	if response.Credits != 2 {
		t.Errorf("Credits = %d, 期望取自 usage", response.Credits)
	}
}

func TestTavilyForwardCreditsFallBackToDepth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// 调用方没要 usage，网关只能按配置的深度计价
		_, _ = w.Write([]byte(`{"query":"go","results":[]}`))
	}))
	defer server.Close()

	provider := NewTavily("key", server.URL, TavilyOptions{SearchDepth: "advanced"}, server.Client())
	response, err := provider.Forward(context.Background(), PassthroughRequest{
		Method: http.MethodPost, Path: "/search", Body: []byte(`{"query":"go"}`),
	})
	if err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}
	if response.Credits != 2 {
		t.Errorf("Credits = %d, advanced 深度应记 2", response.Credits)
	}
}

func TestForwardReportsUpstreamFailureWithoutSwallowingBody(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		provider func(base string, client *http.Client) Forwarder
		wantKind ErrorKind
	}{
		{
			name: "brave 限速", status: http.StatusTooManyRequests, body: `{"error":{"detail":"slow"}}`,
			provider: func(base string, client *http.Client) Forwarder { return NewBrave("k", base, client) },
			wantKind: KindRateLimited,
		},
		{
			name: "tavily 配额耗尽", status: 432, body: `{"detail":{"error":"plan limit"}}`,
			provider: func(base string, client *http.Client) Forwarder {
				return NewTavily("k", base, TavilyOptions{}, client)
			},
			wantKind: KindQuotaExceeded,
		},
		{
			name: "tavily 参数错误", status: http.StatusBadRequest, body: `{"detail":{"error":"bad topic"}}`,
			provider: func(base string, client *http.Client) Forwarder {
				return NewTavily("k", base, TavilyOptions{}, client)
			},
			wantKind: KindBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			response, err := test.provider(server.URL, server.Client()).
				Forward(context.Background(), PassthroughRequest{Method: http.MethodPost, Path: "/x"})
			// A failing upstream is not a transport error: the caller still
			// needs the provider's own error body.
			if err != nil {
				t.Fatalf("Forward 不应把上游错误转成 error: %v", err)
			}
			if response.Status != test.status {
				t.Errorf("Status = %d, 期望 %d", response.Status, test.status)
			}
			if string(response.Body) != test.body {
				t.Errorf("错误体被改写: %s", response.Body)
			}
			if response.Kind != test.wantKind {
				t.Errorf("Kind = %q, 期望 %q", response.Kind, test.wantKind)
			}
			if response.Credits != 0 {
				t.Errorf("Credits = %d, 失败不应计费", response.Credits)
			}
		})
	}
}

func TestForwardRequiresAPIKey(t *testing.T) {
	forwarders := map[string]Forwarder{
		"brave":  NewBrave("", "", nil),
		"tavily": NewTavily("", "", TavilyOptions{}, nil),
	}
	for name, forwarder := range forwarders {
		t.Run(name, func(t *testing.T) {
			_, err := forwarder.Forward(context.Background(), PassthroughRequest{Method: http.MethodGet, Path: "/x"})
			searchErr, ok := err.(*Error)
			if !ok || searchErr.Kind != KindAuthFailed {
				t.Fatalf("未配置 Key 时应返回 auth 错误, 得到 %v", err)
			}
		})
	}
}
