package aigateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dh-blog/internal/modules/agentapi"
	"dh-blog/internal/modules/article"
	"dh-blog/internal/platform/mcp"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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

// --- real agentapi module integration ---

// stubAgentArticles is the smallest article-persistence stand-in the real agent
// module accepts at construction; the scope test never dispatches a tool call,
// so every method is a no-op.
type stubAgentArticles struct{}

func (stubAgentArticles) List(context.Context, string, int, int) ([]article.ArticleBrief, int64, error) {
	return nil, 0, nil
}
func (stubAgentArticles) Get(context.Context, int) (*article.ArticleDetail, error) {
	return &article.ArticleDetail{}, nil
}
func (stubAgentArticles) Create(context.Context, article.CreateInput) (int, error) {
	return 1, nil
}
func (stubAgentArticles) Update(context.Context, article.UpdateInput) error { return nil }

// stubAgentImages and stubAgentTasks are the other two collaborators the agent
// module requires; Events stays nil and falls back to the module's no-op.
type stubAgentImages struct{}

func (stubAgentImages) SaveBlogImage(context.Context, string, []byte) (string, error) {
	return "", nil
}

type stubAgentTasks struct{}

func (stubAgentTasks) SubmitTagGeneration(int, string)     {}
func (stubAgentTasks) SubmitSummaryGeneration(int, string) {}

// newRealAgentModule builds the actual agentapi module with real MCPTools(),
// standing in for what the registry wires in production. The gateway must accept
// its tool table exactly like the fakes above, and the tool names / scopes are
// the literal wire contract this test pins down.
func newRealAgentModule(t *testing.T) *agentapi.Module {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 agentapi 测试数据库失败: %v", err)
	}
	module, err := agentapi.New(agentapi.Dependencies{
		DB:       db,
		Articles: stubAgentArticles{},
		Images:   stubAgentImages{},
		Tasks:    stubAgentTasks{},
		Events:   nil,
	})
	if err != nil {
		t.Fatalf("构建真实 agentapi 模块失败: %v", err)
	}
	return module
}

// TestMCPHiddenWriteToolIsNotCallable drives the HTTP layer end to end with a
// content:read-only key and a real agent request for a write tool. Hiding is
// only worth anything if it also means un-invokable: a direct tools/call for
// create_article must be refused at the protocol layer as "未知的工具"
// (-32602 InvalidParams), never executed as a side effect.
func TestMCPHiddenWriteToolIsNotCallable(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: newRealAgentModule(t),
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.Scopes = ScopeContentRead })

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call",` +
		`"params":{"name":"create_article","arguments":{"title":"越权测试","content":"正文"}}}`
	response := decodeRPC(t, doMCP(engine, token, body))
	if response.Error == nil {
		t.Fatalf("对隐藏工具的调用被放行了: %+v", response.Result)
	}
	if response.Error.Code != mcp.InvalidParams {
		t.Fatalf("错误码 = %d, 期望 %d (-32602)", response.Error.Code, mcp.InvalidParams)
	}
	if !strings.Contains(response.Error.Message, "未知的工具") {
		t.Fatalf("错误文案不易读: %q", response.Error.Message)
	}
	// 列表侧的对照：这把 key 的 tools/list 本就看不到写作工具
	list := rpcResultAs[mcp.ToolListResult](t, doMCP(engine, token,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	for _, tool := range list.Tools {
		if tool.Name == "create_article" {
			t.Fatal("content:read key 的 tools/list 里不该有 create_article")
		}
	}
}

// TestMCPRealAgentToolsRespectScopeContract drives tools/list through the real
// agentapi tool table and real keys, asserting visibility against the wire names
// both modules spell out. agentapi deliberately re-declares its scope strings so
// it never imports aigateway, so this test is the tripwire: if those literals
// drift from ScopeContentRead / ScopeContentWrite, a content:read key stops
// being shielded from the write tools and this case goes red.
func TestMCPRealAgentToolsRespectScopeContract(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: newRealAgentModule(t),
	})
	engine := newTestEngine(module)

	readOnly := []string{"list_articles", "get_article"}
	writeOnly := []string{"create_article", "update_article", "upload_image"}

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
	seen := func(names []string, tool string) bool {
		for _, name := range names {
			if name == tool {
				return true
			}
		}
		return false
	}

	t.Run("只有 content:read 只见读工具", func(t *testing.T) {
		token := issueTestKey(t, module, func(key *APIKey) { key.Scopes = ScopeContentRead })
		names := list(t, token)
		for _, tool := range readOnly {
			if !seen(names, tool) {
				t.Errorf("content:read key 看不到 %s, 列表 = %v", tool, names)
			}
		}
		for _, tool := range writeOnly {
			if seen(names, tool) {
				t.Errorf("content:read key 不该看到 %s, 列表 = %v", tool, names)
			}
		}
	})

	t.Run("只有 content:write 只见写作工具", func(t *testing.T) {
		// 设计文档明确：write 不代表 read，一把只有写权限的 key 看不到读工具。
		token := issueTestKey(t, module, func(key *APIKey) { key.Scopes = ScopeContentWrite })
		names := list(t, token)
		for _, tool := range writeOnly {
			if !seen(names, tool) {
				t.Errorf("content:write key 看不到 %s, 列表 = %v", tool, names)
			}
		}
		for _, tool := range readOnly {
			if seen(names, tool) {
				t.Errorf("content:write key 不该看到 %s, 列表 = %v", tool, names)
			}
		}
	})

	t.Run("read + write 五个写作工具全部可见", func(t *testing.T) {
		token := issueTestKey(t, module, func(key *APIKey) {
			key.Scopes = ScopeContentRead + "," + ScopeContentWrite
		})
		names := list(t, token)
		for _, tool := range append(append([]string{}, readOnly...), writeOnly...) {
			if !seen(names, tool) {
				t.Errorf("read+write key 看不到 %s, 列表 = %v", tool, names)
			}
		}
	})
}
