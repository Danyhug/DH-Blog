package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dh-blog/internal/dhcache"
)

type testAIConfigSource struct {
	endpoint      string
	prompt        string
	summaryPrompt string
}

func (s testAIConfigSource) LoadAITaggingConfig(context.Context) (string, string, string, string, error) {
	return s.endpoint, "test-key", "test-model", s.prompt, nil
}

func (s testAIConfigSource) LoadAISummaryConfig(context.Context) (string, string, string, string, error) {
	return s.endpoint, "test-key", "test-model", s.summaryPrompt, nil
}

func TestGenerateTagsRendersExistingTagsAsJSON(t *testing.T) {
	var renderedPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(request.Messages) != 1 {
			t.Errorf("message count = %d, want 1", len(request.Messages))
		} else {
			renderedPrompt = request.Messages[0].Content
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"[\"Go\",\"知识管理\"]"}}]}`))
	}))
	t.Cleanup(server.Close)

	cache := dhcache.NewCache()
	t.Cleanup(cache.Shutdown)
	service := NewAIService(testAIConfigSource{
		endpoint: server.URL,
		prompt:   "article={{.Article}}\ntags={{.Tags}}",
	}, cache)

	tags, err := service.GenerateTags("正文", []string{"Go", "知识管理"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0] != "Go" || tags[1] != "知识管理" {
		t.Fatalf("tags = %#v", tags)
	}
	if !strings.Contains(renderedPrompt, `tags=["Go","知识管理"]`) {
		t.Fatalf("rendered prompt does not contain JSON tags: %q", renderedPrompt)
	}
}

func TestGenerateSummaryExtractsBracketedContent(t *testing.T) {
	var renderedPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(request.Messages) == 1 {
			renderedPrompt = request.Messages[0].Content
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"[本文讲述了摘要要点。]"}}]}`))
	}))
	t.Cleanup(server.Close)

	cache := dhcache.NewCache()
	t.Cleanup(cache.Shutdown)
	service := NewAIService(testAIConfigSource{
		endpoint:      server.URL,
		summaryPrompt: "article={{.ArticleContent}}",
	}, cache)

	summary, err := service.GenerateSummary("正文内容")
	if err != nil {
		t.Fatal(err)
	}
	if summary != "本文讲述了摘要要点。" {
		t.Fatalf("summary = %q", summary)
	}
	if !strings.Contains(renderedPrompt, "article=正文内容") {
		t.Fatalf("rendered prompt = %q, want it to carry the article content", renderedPrompt)
	}
}

func TestGenerateSummaryRejectsEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	t.Cleanup(server.Close)

	cache := dhcache.NewCache()
	t.Cleanup(cache.Shutdown)
	service := NewAIService(testAIConfigSource{
		endpoint:      server.URL,
		summaryPrompt: "article={{.ArticleContent}}",
	}, cache)

	if _, err := service.GenerateSummary("正文内容"); err == nil {
		t.Fatal("expected an error when the API returns no choices")
	}
}

func TestExtractBracketedFallsBackToRawText(t *testing.T) {
	cases := map[string]string{
		"[被包裹的摘要]":   "被包裹的摘要",
		" 没有方括号的摘要 ": "没有方括号的摘要",
		"前缀 [正文] 后缀": "正文",
	}
	for input, want := range cases {
		if got := extractBracketed(input); got != want {
			t.Fatalf("extractBracketed(%q) = %q, want %q", input, got, want)
		}
	}
}
