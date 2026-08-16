package article

import (
	"encoding/json"
	"testing"
)

func TestArticleJSONExposesAuthorIdentity(t *testing.T) {
	article := Article{
		Title:       "标题",
		AuthorType:  "agent",
		AuthorName:  "写作助手",
		AuthorKeyID: 42,
	}
	encoded, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("marshal article: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if got, _ := fields["authorType"].(string); got != "agent" {
		t.Fatalf("authorType = %#v, want %q", fields["authorType"], "agent")
	}
	if got, _ := fields["authorName"].(string); got != "写作助手" {
		t.Fatalf("authorName = %#v, want %q", fields["authorName"], "写作助手")
	}
	// 内部 key id 不该出现在公开接口里
	if _, ok := fields["authorKeyId"]; ok {
		t.Fatalf("marshal output leaked authorKeyId: %s", encoded)
	}
}

func TestZeroValueArticleJSONKeepsAuthorTypeEmpty(t *testing.T) {
	// 存量文章三个字段都是零值，前台不显示徽标
	encoded, err := json.Marshal(Article{})
	if err != nil {
		t.Fatalf("marshal zero article: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if got, _ := fields["authorType"].(string); got != "" {
		t.Fatalf("authorType = %#v, want empty string", fields["authorType"])
	}
}
