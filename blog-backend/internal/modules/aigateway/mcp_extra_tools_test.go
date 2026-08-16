package aigateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dh-blog/internal/modules/agentapi"
	"dh-blog/internal/platform/mcp"
)

// fakeToolSource hands the module a fixed tool table, standing in for the
// agent module the registry wires in production.
type fakeToolSource struct{ tools []mcp.Tool }

func (s fakeToolSource) MCPTools() []mcp.Tool { return s.tools }

// scopedFakeTool is an extra tool that declares a scope and, on call, verifies
// the identity the transport stashed in the context.
type scopedFakeTool struct {
	name         string
	scope        string
	wantKeyID    int
	seenIdentity agentapi.Identity
}

func (t *scopedFakeTool) Name() string  { return t.name }
func (t *scopedFakeTool) Scope() string { return t.scope }
func (t *scopedFakeTool) Definition(context.Context) mcp.Definition {
	return mcp.Definition{Name: t.name, Title: t.name, Description: "fake tool", InputSchema: map[string]any{"type": "object"}}
}

func (t *scopedFakeTool) Call(ctx context.Context, _ json.RawMessage) mcp.Result {
	identity, ok := agentapi.IdentityFrom(ctx)
	if !ok {
		return mcp.ToolError("context 里没有身份")
	}
	t.seenIdentity = identity
	if identity.KeyID() != t.wantKeyID {
		return mcp.ToolError("身份 KeyID 不匹配")
	}
	return mcp.Text("written by " + identity.AuthorName())
}

// plainFakeTool is an extra tool without Scope() — the search-side case that
// must stay visible to every key.
type plainFakeTool struct{ name string }

func (t *plainFakeTool) Name() string { return t.name }
func (t *plainFakeTool) Definition(context.Context) mcp.Definition {
	return mcp.Definition{Name: t.name, Title: t.name, Description: "plain fake", InputSchema: map[string]any{"type": "object"}}
}

func (t *plainFakeTool) Call(context.Context, json.RawMessage) mcp.Result { return mcp.Text("plain") }

// TestMCPToolListFiltersByScope verifies tools/list only advertises extra
// tools whose scope the calling key holds; a key without the scope never sees
// the writing tools, so the model does not burn tokens trying them.
func TestMCPToolListFiltersByScope(t *testing.T) {
	readTool := &scopedFakeTool{name: "fake_read", scope: ScopeContentRead}
	openTool := &plainFakeTool{name: "fake_open"}
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: fakeToolSource{tools: []mcp.Tool{readTool, openTool}},
	})
	engine := newTestEngine(module)

	list := func(t *testing.T, token string) []string {
		t.Helper()
		result := rpcResultAs[mcp.ToolListResult](t, doMCP(engine, token,
			`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		return names
	}
	assertTools := func(t *testing.T, got []string, want ...string) {
		t.Helper()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("工具列表 = %v, 期望 %v", got, want)
		}
	}

	t.Run("空 scope 只见搜索与无 scope 工具", func(t *testing.T) {
		token := issueTestKey(t, module, nil)
		assertTools(t, list(t, token), mcpToolWebSearch, openTool.name)
	})
	t.Run("只有 content:write 也看不到 content:read 工具", func(t *testing.T) {
		token := issueTestKey(t, module, func(key *APIKey) { key.Scopes = ScopeContentWrite })
		assertTools(t, list(t, token), mcpToolWebSearch, openTool.name)
	})
	t.Run("有 content:read 全部可见", func(t *testing.T) {
		token := issueTestKey(t, module, func(key *APIKey) { key.Scopes = ScopeContentRead })
		assertTools(t, list(t, token), mcpToolWebSearch, readTool.name, openTool.name)
	})
}

// TestMCPExtraToolCallCarriesIdentity verifies the transport puts the calling
// key into the context as an agentapi.Identity, so a writing tool can act on
// the caller's behalf without ever knowing the concrete key type.
func TestMCPExtraToolCallCarriesIdentity(t *testing.T) {
	tool := &scopedFakeTool{name: "fake_write", scope: ScopeContentWrite}
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: fakeToolSource{tools: []mcp.Tool{tool}},
	})
	engine := newTestEngine(module)

	plain, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey 返回错误: %v", err)
	}
	key := APIKey{Name: "写作代理", KeyPrefix: APIKeyPrefixOf(plain), KeyHash: HashAPIKey(plain), Enabled: true, Scopes: ScopeContentWrite}
	if err := module.service.repo.createAPIKey(context.Background(), &key); err != nil {
		t.Fatalf("createAPIKey 返回错误: %v", err)
	}
	tool.wantKeyID = key.ID

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fake_write","arguments":{}}}`
	result := rpcResultAs[mcp.Result](t, doMCP(engine, plain, body))
	if result.IsError {
		t.Fatalf("工具调用失败: %+v", result.Content)
	}
	if tool.seenIdentity == nil {
		t.Fatal("工具没有拿到身份")
	}
	if tool.seenIdentity.KeyID() != key.ID {
		t.Errorf("身份 KeyID = %d, 期望 %d", tool.seenIdentity.KeyID(), key.ID)
	}
	if got := tool.seenIdentity.AuthorName(); got != "写作代理" {
		t.Errorf("AuthorName = %q, 期望 %q", got, "写作代理")
	}
	if len(result.Content) != 1 || !strings.Contains(result.Content[0].Text, "written by 写作代理") {
		t.Errorf("返回文本异常: %+v", result.Content)
	}
}

func TestAPIKeyAuthorNameFallsBackToName(t *testing.T) {
	key := APIKey{Name: "默认名", Byline: "   "}
	if got := key.AuthorName(); got != "默认名" {
		t.Errorf("空白 Byline 未回退到 Name: %q", got)
	}
	key.Byline = "署名"
	if got := key.AuthorName(); got != "署名" {
		t.Errorf("AuthorName = %q, 期望 %q", got, "署名")
	}
}
