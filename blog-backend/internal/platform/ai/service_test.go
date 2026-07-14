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
	endpoint string
	prompt   string
}

func (s testAIConfigSource) LoadAITaggingConfig(context.Context) (string, string, string, string, error) {
	return s.endpoint, "test-key", "test-model", s.prompt, nil
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
