package search

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func scrapeServer(t *testing.T, status int, body string, inspect func(*http.Request, map[string]any)) *FirecrawlProvider {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if inspect != nil {
			inspect(r, payload)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return newTestFirecrawl(FirecrawlOptions{}, server.URL, server.Client())
}

func TestFirecrawlFetchRequestAndResponse(t *testing.T) {
	provider := scrapeServer(t, http.StatusOK, `{"success":true,"data":{"markdown":"  # Hello\n\nbody  ",
		"metadata":{"title":["", "Hello Page"],"description":"desc","sourceURL":"https://a.dev/x",
		"url":"https://a.dev/final","statusCode":200,"contentType":"text/html"}}}`,
		func(r *http.Request, payload map[string]any) {
			if r.URL.Path != "/scrape" || r.Header.Get("Authorization") != "Bearer key" {
				t.Errorf("请求 = %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
			}
			if payload["url"] != "https://a.dev/x" || payload["onlyMainContent"] != true {
				t.Errorf("payload = %v", payload)
			}
			if formats, _ := payload["formats"].([]any); len(formats) != 1 || formats[0] != "markdown" {
				t.Errorf("formats = %v", payload["formats"])
			}
			// 上游超时要比网关自己的客户端先到
			if payload["timeout"] != float64(13000) {
				t.Errorf("timeout = %v, 期望 13000", payload["timeout"])
			}
		})

	page, err := provider.Fetch(context.Background(), FetchRequest{URL: "https://a.dev/x", Timeout: 13 * time.Second})
	if err != nil {
		t.Fatalf("Fetch 返回错误: %v", err)
	}
	// title 可能是数组，取第一个非空的；最终地址以重定向后的为准
	if page.Title != "Hello Page" || page.URL != "https://a.dev/final" || page.Content != "# Hello\n\nbody" ||
		page.StatusCode != 200 || page.Credits != 1 || page.Description != "desc" {
		t.Fatalf("page = %+v", page)
	}
}

func TestFirecrawlFetchBillsPDFPerPage(t *testing.T) {
	provider := scrapeServer(t, http.StatusOK, `{"success":true,"data":{"markdown":"pdf",
		"metadata":{"statusCode":200,"contentType":"application/pdf","numPages":7}}}`, nil)
	page, err := provider.Fetch(context.Background(), FetchRequest{URL: "https://a.dev/x.pdf"})
	if err != nil {
		t.Fatalf("Fetch 返回错误: %v", err)
	}
	if page.Credits != 7 || page.URL != "https://a.dev/x.pdf" {
		t.Fatalf("page = %+v, PDF 按页计费", page)
	}
}

func TestFirecrawlFetchKeepsTargetErrorPages(t *testing.T) {
	// 目标站 404 但 Firecrawl 拿回了文档：照样计费，作为页面返回而不是错误
	provider := scrapeServer(t, http.StatusOK, `{"success":true,"data":{"markdown":"Not Found",
		"metadata":{"statusCode":404}}}`, nil)
	page, err := provider.Fetch(context.Background(), FetchRequest{URL: "https://a.dev/missing"})
	if err != nil || page.StatusCode != 404 || page.Credits != 1 {
		t.Fatalf("page = %+v, err = %v", page, err)
	}
}

func TestFirecrawlFetchErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		kind   ErrorKind
	}{
		// 页面本身抓不下来：不是 Firecrawl 坏了，不该重试也不该计入熔断
		{"页面抓取失败", http.StatusOK, `{"success":false,"error":"All scraping engines failed"}`, KindBadRequest},
		{"额度用完", http.StatusPaymentRequired, `{"success":false,"error":"Insufficient credits"}`, KindQuotaExceeded},
		{"密钥无效", http.StatusUnauthorized, `{"success":false,"error":"Unauthorized"}`, KindAuthFailed},
		{"上游故障", http.StatusInternalServerError, `{"success":false,"code":"UNKNOWN_ERROR"}`, KindUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := scrapeServer(t, test.status, test.body, nil)
			_, err := provider.Fetch(context.Background(), FetchRequest{URL: "https://a.dev/x"})
			var providerErr *Error
			if !errors.As(err, &providerErr) || providerErr.Kind != test.kind {
				t.Fatalf("err = %v, 期望 %s", err, test.kind)
			}
		})
	}
}
