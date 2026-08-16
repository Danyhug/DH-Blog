package aigateway

import (
	"strings"
	"testing"

	"dh-blog/internal/platform/mcp"
)

// TestMCPContentToolsEnforcePerMinuteRateLimit proves the per-key allowance
// that the search path already applies also guards the content tools. A stolen
// write key must not be able to spam creates and uploads at full speed once its
// search calls have been metered: the first extra-tool call goes through, the
// next one in the same minute is refused with a readable tool error.
func TestMCPContentToolsEnforcePerMinuteRateLimit(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: fakeToolSource{tools: []mcp.Tool{&plainFakeTool{name: "fake_content"}}},
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) {
		key.RateLimitPerMin = 1
		key.Scopes = ScopeContentWrite
	})

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fake_content","arguments":{}}}`
	first := rpcResultAs[mcp.Result](t, doMCP(engine, token, body))
	if first.IsError {
		t.Fatalf("首次内容工具调用不应失败: %+v", first.Content)
	}
	second := rpcResultAs[mcp.Result](t, doMCP(engine, token, body))
	if !second.IsError {
		t.Fatal("同分钟内第二次内容工具调用应被限流")
	}
	if !strings.Contains(second.Content[0].Text, "请求过于频繁") {
		t.Errorf("限流文案不易读: %q", second.Content[0].Text)
	}
}

// TestMCPUnlimitedKeyBypassesContentRateLimit pins RateLimitPerMin=0 as
// "no per-minute limit": the content-tool wrapper shares the search semantics.
func TestMCPUnlimitedKeyBypassesContentRateLimit(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:      braveOK("b1"),
		ExtraTools: fakeToolSource{tools: []mcp.Tool{&plainFakeTool{name: "fake_content"}}},
	})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, func(key *APIKey) { key.Scopes = ScopeContentWrite })

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fake_content","arguments":{}}}`
	for i := 0; i < 2; i++ {
		result := rpcResultAs[mcp.Result](t, doMCP(engine, token, body))
		if result.IsError {
			t.Fatalf("不限速的 key 第 %d 次调用失败: %+v", i+1, result.Content)
		}
	}
}