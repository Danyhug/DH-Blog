package aigateway

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"dh-blog/internal/platform/mcp"
)

func fetchMCPCatalog(t *testing.T, config gatewayTestConfig) mcpCatalogView {
	t.Helper()
	engine := newTestEngine(newGatewayTestModule(t, config))
	recorder := doAdmin(engine, http.MethodGet, "/api/admin/gateway/mcp/tools", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("目录接口状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Data mcpCatalogView `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析目录响应失败: %v (body=%s)", err, recorder.Body.String())
	}
	return body.Data
}

func toolIn(catalog mcpCatalogView, name string) (mcpToolView, bool) {
	for _, tool := range catalog.Tools {
		if tool.Name == name {
			return tool, true
		}
	}
	return mcpToolView{}, false
}

// TestMCPCatalogListsEveryMountedToolWithItsScope is the point of the endpoint:
// the admin page must not need editing when a module contributes a new tool.
// A real agentapi module stands in for that contribution.
func TestMCPCatalogListsEveryMountedToolWithItsScope(t *testing.T) {
	catalog := fetchMCPCatalog(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: newRealAgentModule(t),
	})

	wantScopes := map[string]string{
		mcpToolWebSearch: ScopeSearch,
		"list_articles":  ScopeContentRead,
		"get_article":    ScopeContentRead,
		"create_article": ScopeContentWrite,
		"update_article": ScopeContentWrite,
		"upload_image":   ScopeContentWrite,
	}
	if len(catalog.Tools) != len(wantScopes) {
		t.Fatalf("工具数量 = %d, 期望 %d: %+v", len(catalog.Tools), len(wantScopes), catalog.Tools)
	}
	for name, scope := range wantScopes {
		tool, ok := toolIn(catalog, name)
		if !ok {
			t.Errorf("目录里缺少工具 %q", name)
			continue
		}
		if tool.Scope != scope {
			t.Errorf("%s 的 scope = %q, 期望 %q", name, tool.Scope, scope)
		}
		if tool.Title == "" || tool.Description == "" {
			t.Errorf("%s 缺少展示所需的标题或说明: %+v", name, tool)
		}
	}

	// web_search is mounted for every key, so the catalog must not present it
	// as something a key could be missing.
	search, _ := toolIn(catalog, mcpToolWebSearch)
	if search.Scope != ScopeSearch {
		t.Errorf("web_search 应归入基线能力, 实际 scope = %q", search.Scope)
	}
}

// TestMCPCatalogScopesCoverEveryToolScope guards the pairing the page relies on:
// every scope a tool declares has a descriptor to render it under, and the
// baseline flag marks exactly the scope that is not a checkbox.
func TestMCPCatalogScopesCoverEveryToolScope(t *testing.T) {
	catalog := fetchMCPCatalog(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: newRealAgentModule(t),
	})

	described := map[string]ScopeDescriptor{}
	for _, scope := range catalog.Scopes {
		described[scope.Value] = scope
	}
	for _, tool := range catalog.Tools {
		descriptor, ok := described[tool.Scope]
		if !ok {
			t.Errorf("工具 %s 的 scope %q 没有对应描述，页面会渲染成无名分组", tool.Name, tool.Scope)
			continue
		}
		if descriptor.Label == "" {
			t.Errorf("scope %q 缺少中文标签", tool.Scope)
		}
	}

	baseline := make([]string, 0, 1)
	for _, scope := range catalog.Scopes {
		if scope.Baseline {
			baseline = append(baseline, scope.Value)
		}
	}
	if len(baseline) != 1 || baseline[0] != ScopeSearch {
		t.Errorf("基线 scope = %v, 期望只有 %q", baseline, ScopeSearch)
	}
}

// TestMCPCatalogScopesAreAcceptedByNormalize keeps the key form honest: every
// selectable scope it renders must survive the validation the create endpoint
// runs, or the admin gets "不支持的 scope" from an option the page offered.
func TestMCPCatalogScopesAreAcceptedByNormalize(t *testing.T) {
	for _, scope := range ScopeCatalog() {
		got, err := NormalizeScopes(scope.Value)
		if err != nil {
			t.Errorf("NormalizeScopes(%q) 返回错误: %v", scope.Value, err)
			continue
		}
		if got != scope.Value {
			t.Errorf("NormalizeScopes(%q) = %q, 期望原样保留", scope.Value, got)
		}
	}
}

// TestMCPCatalogRendersToolParameters checks the flattening, including that
// `required` survives both schema shapes and sorts ahead of optional fields.
func TestMCPCatalogRendersToolParameters(t *testing.T) {
	catalog := fetchMCPCatalog(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: newRealAgentModule(t),
	})

	get, ok := toolIn(catalog, "get_article")
	if !ok {
		t.Fatal("目录里缺少 get_article")
	}
	if len(get.Params) != 1 {
		t.Fatalf("get_article 参数 = %+v, 期望只有 id", get.Params)
	}
	if get.Params[0].Name != "id" || get.Params[0].Type != "integer" || !get.Params[0].Required {
		t.Errorf("get_article 的 id 参数渲染有误: %+v", get.Params[0])
	}

	create, ok := toolIn(catalog, "create_article")
	if !ok {
		t.Fatal("目录里缺少 create_article")
	}
	// 必填在前、同组按名字排序，保证同一份 schema 每次渲染顺序一致
	optionalSeen := false
	for _, param := range create.Params {
		if !param.Required {
			optionalSeen = true
			continue
		}
		if optionalSeen {
			t.Fatalf("必填参数排在了可选参数之后: %+v", create.Params)
		}
	}

	search, _ := toolIn(catalog, mcpToolWebSearch)
	if len(search.Params) == 0 {
		t.Error("web_search 没有渲染出任何参数")
	}
}

func TestSchemaParamsIgnoresUnexpectedShapes(t *testing.T) {
	cases := []struct {
		name   string
		schema any
	}{
		{name: "非对象", schema: "string"},
		{name: "缺 properties", schema: map[string]any{"type": "object"}},
		{name: "properties 类型异常", schema: map[string]any{"properties": []any{"a"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := schemaParams(tc.schema); got != nil {
				t.Errorf("schemaParams(%#v) = %+v, 期望 nil", tc.schema, got)
			}
		})
	}

	// JSON 往返后 required 会变成 []any，两种形态都得认
	roundTripped := schemaParams(map[string]any{
		"properties": map[string]any{"id": map[string]any{"type": "integer"}},
		"required":   []any{"id"},
	})
	if len(roundTripped) != 1 || !roundTripped[0].Required {
		t.Errorf("[]any 形式的 required 未生效: %+v", roundTripped)
	}
}

// TestMCPInstructionsFollowKeyScopes pins the fix for a server that described
// itself the same way to every caller: a content-only key was still told the
// server "提供联网搜索"… and handed a list of writing tools, while a pure search
// key was told about writing tools it could not see. The text now names only
// tools the key's own tools/list will return.
func TestMCPInstructionsFollowKeyScopes(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: newRealAgentModule(t),
	})
	engine := newTestEngine(module)

	instructionsFor := func(t *testing.T, token string) string {
		t.Helper()
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18",` +
			`"capabilities":{},"clientInfo":{"name":"claude-code","version":"1"}}}`
		return rpcResultAs[mcp.InitializeResult](t, doMCP(engine, token, body)).Instructions
	}

	searchOnly := instructionsFor(t, issueTestKey(t, module, nil))
	if !strings.Contains(searchOnly, "web_search") {
		t.Errorf("纯搜索 key 的说明没提 web_search: %q", searchOnly)
	}
	for _, name := range []string{"create_article", "list_articles", "写作后台"} {
		if strings.Contains(searchOnly, name) {
			t.Errorf("纯搜索 key 的说明提到了它看不到的 %q: %q", name, searchOnly)
		}
	}

	writer := instructionsFor(t, issueTestKey(t, module, func(key *APIKey) { key.Scopes = ScopeContentWrite }))
	for _, name := range []string{"web_search", "create_article", "update_article", "upload_image"} {
		if !strings.Contains(writer, name) {
			t.Errorf("写入 key 的说明漏了 %q: %q", name, writer)
		}
	}
	// content:write 不含 content:read，说明里也不该出现只读工具
	if strings.Contains(writer, "list_articles") {
		t.Errorf("只有 content:write 的 key 看不到 list_articles，说明不该提它: %q", writer)
	}
}

// TestRateLimitWrapperKeepsToolScope guards the wrapper that carries content
// tools at request time: embedding mcp.Tool promotes only that interface's
// methods, so a forwarding Scope() is the only thing keeping a wrapped
// create_article from being classified as a baseline tool.
func TestRateLimitWrapperKeepsToolScope(t *testing.T) {
	wrapped := rateLimitedTool{
		Tool:  &scopedFakeTool{name: "fake_write", scope: ScopeContentWrite},
		allow: func() bool { return true },
	}
	if got := toolScope(wrapped); got != ScopeContentWrite {
		t.Errorf("包装后的 scope = %q, 期望 %q", got, ScopeContentWrite)
	}
}
