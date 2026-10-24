package jev

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, status int, body string) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/systemone" || r.Method != http.MethodPost {
			t.Errorf("请求 = %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return New("key", server.URL+"/", "", server.Client())
}

var twoOptions = Choice{Instructions: "which?", Criteria: map[string]string{"a": "first", "b": "second"}}

func TestChooseParsesAnswer(t *testing.T) {
	client := serve(t, http.StatusOK, `{"model":"jev-1.14.0","answers":{"pick":{"type":"choice","choice":"b",
		"confidence":0.42,"probabilities":{"a":0.35,"b":0.65}}},"usage":{"input_tokens":120,"output_tokens":12}}`)

	decision, err := client.Choose(context.Background(), map[string]any{"query": "x"}, twoOptions)
	if err != nil {
		t.Fatalf("Choose 返回错误: %v", err)
	}
	if decision.Choice != "b" || decision.Confidence != 0.42 || decision.Model != "jev-1.14.0" ||
		decision.Probabilities["a"] != 0.35 || decision.InputTokens != 120 {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestChooseRejectsUnofferedOption(t *testing.T) {
	client := serve(t, http.StatusOK, `{"answers":{"pick":{"type":"choice","choice":"z","confidence":1}}}`)
	if _, err := client.Choose(context.Background(), "x", twoOptions); err == nil {
		t.Fatal("返回未提供的选项时应报错")
	}
}

func TestChooseReportsErrorBody(t *testing.T) {
	client := serve(t, http.StatusUnprocessableEntity, `{"detail":[{"loc":["body","questions"],"msg":"field required"}]}`)
	_, err := client.Choose(context.Background(), "x", twoOptions)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnprocessableEntity || !strings.Contains(apiErr.Message, "field required") {
		t.Fatalf("err = %v", err)
	}
}

func TestChooseRequiresKey(t *testing.T) {
	if _, err := New("", "", "", nil).Choose(context.Background(), "x", twoOptions); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, 期望 ErrNotConfigured", err)
	}
}

func TestChooseExplainsRegionBlock(t *testing.T) {
	// 真实响应：从中国大陆出口访问时 Cloudflare 层直接返回 451，Key 都还没校验
	client := serve(t, http.StatusUnavailableForLegalReasons, `{"title":"Typesafe is not available in your region.","status":451}`)
	_, err := client.Choose(context.Background(), "x", twoOptions)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnavailableForLegalReasons ||
		!strings.Contains(apiErr.Message, "not available in your region") || !strings.Contains(apiErr.Message, "境外") {
		t.Fatalf("err = %v", err)
	}
}
