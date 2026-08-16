package aigateway

import (
	"encoding/json"
	"testing"
)

func TestEmptyScopesMeanSearchOnly(t *testing.T) {
	// 与 AllowedProviders 相反：空清单不给任何写权限，升级存量 key 不会凭空扩大能力
	key := APIKey{}
	if !key.HasScope(ScopeSearch) {
		t.Fatal("empty scopes should grant search")
	}
	if key.HasScope(ScopeContentRead) {
		t.Fatal("empty scopes must not grant content:read")
	}
	if key.HasScope(ScopeContentWrite) {
		t.Fatal("empty scopes must not grant content:write")
	}
}

func TestListedScopesAreGrantedAndSearchIsNotImplied(t *testing.T) {
	key := APIKey{Scopes: "content:read, content:write"}
	if !key.HasScope(ScopeContentRead) || !key.HasScope(ScopeContentWrite) {
		t.Fatal("listed scopes should be granted")
	}
	if key.HasScope(ScopeSearch) {
		t.Fatal("an explicit list must not imply search")
	}
}

func TestExplicitSearchScope(t *testing.T) {
	key := APIKey{Scopes: "search"}
	if !key.HasScope(ScopeSearch) {
		t.Fatal("explicit search scope should be granted")
	}
}

func TestScopesListSplitsAndTrims(t *testing.T) {
	key := APIKey{Scopes: " content:read , search , "}
	got := key.ScopesList()
	if len(got) != 2 || got[0] != "content:read" || got[1] != "search" {
		t.Fatalf("ScopesList() = %#v, want [content:read search]", got)
	}
}

func TestNormalizeScopes(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "空输入等价于仅 search", raw: "", want: ""},
		{name: "去空白段", raw: " search , content:write, search ", want: "search,content:write"},
		{name: "全量合法组合", raw: "content:read,content:write,search", want: "content:read,content:write,search"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeScopes(tc.raw)
			if err != nil {
				t.Fatalf("NormalizeScopes(%q) error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeScopes(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
	for _, raw := range []string{"write", "content:delete", "content:read, evil"} {
		t.Run("拒绝未知 scope "+raw, func(t *testing.T) {
			if _, err := NormalizeScopes(raw); err == nil {
				t.Fatalf("NormalizeScopes(%q) should reject unknown scope", raw)
			}
		})
	}
}

func TestAPIKeyJSONExposesScopesAndAuthorName(t *testing.T) {
	key := APIKey{
		Name:       "writer",
		KeyPrefix:  "gw_live_abc12345",
		Scopes:     "search,content:write",
		Byline:     "写作助手",
	}
	encoded, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("marshal api key: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if got, _ := fields["scopes"].(string); got != "search,content:write" {
		t.Fatalf("scopes = %#v, want %q", fields["scopes"], "search,content:write")
	}
	if got, _ := fields["authorName"].(string); got != "写作助手" {
		t.Fatalf("authorName = %#v, want %q", fields["authorName"], "写作助手")
	}
}
