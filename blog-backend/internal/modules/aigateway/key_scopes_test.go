package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// createKeyResponse parses the {code,msg,data} shape the create endpoint
// returns, so a test can keep the key id the API handed out.
type createKeyResponse struct {
	Data struct {
		ID     int    `json:"id"`
		APIKey string `json:"apiKey"`
	} `json:"data"`
}

func createAgentKey(t *testing.T, engine *gin.Engine, body string) createKeyResponse {
	t.Helper()
	recorder := doAdmin(engine, http.MethodPost, "/api/admin/gateway/keys", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("创建 Key 状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var created createKeyResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("解析创建响应失败: %v", err)
	}
	if created.Data.ID == 0 {
		t.Fatal("创建响应没有带出 key id")
	}
	return created
}

func TestAdminCreateKeyPersistsScopesAndAuthorName(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	created := createAgentKey(t, engine,
		`{"name":"writer","scopes":"content:read, content:write","authorName":"  Claude  "}`)

	keys, err := module.service.repo.listAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("listAPIKeys 返回错误: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("Key 数量 = %d, 期望 1", len(keys))
	}
	stored := keys[0]
	if stored.ID != created.Data.ID {
		t.Fatalf("落库 ID = %d, 期望 %d", stored.ID, created.Data.ID)
	}
	if stored.Scopes != "content:read,content:write" {
		t.Errorf("Scopes = %q, 期望规范化、去重后的列表", stored.Scopes)
	}
	if stored.Byline != "Claude" {
		t.Errorf("Byline = %q, 期望去除首尾空白", stored.Byline)
	}

	// The same fields must come back through the admin list endpoint.
	list := doAdmin(engine, http.MethodGet, "/api/admin/gateway/keys", "")
	var envelope struct {
		Data []apiKeyView `json:"data"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析列表响应失败: %v (body=%s)", err, list.Body.String())
	}
	if len(envelope.Data) != 1 {
		t.Fatalf("列表条数 = %d, 期望 1", len(envelope.Data))
	}
	view := envelope.Data[0]
	if view.Scopes != "content:read,content:write" {
		t.Errorf("apiKeyView.Scopes = %q", view.Scopes)
	}
	if view.AuthorName != "Claude" {
		t.Errorf("apiKeyView.AuthorName = %q", view.AuthorName)
	}
	if strings.Contains(list.Body.String(), created.Data.APIKey) {
		t.Fatal("列表接口不应返回明文 Key")
	}
	// The wire names have to be scopes/authorName for the admin form to bind.
	var rawList struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &rawList); err != nil {
		t.Fatalf("解析列表响应失败: %v (body=%s)", err, list.Body.String())
	}
	for _, raw := range rawList.Data {
		if _, ok := raw["scopes"]; !ok {
			t.Error("apiKeyView JSON 应包含 scopes 字段")
		}
		if _, ok := raw["authorName"]; !ok {
			t.Error("apiKeyView JSON 应包含 authorName 字段")
		}
	}
}

func TestAdminCreateKeyRejectsUnknownScope(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	recorder := doAdmin(engine, http.MethodPost, "/api/admin/gateway/keys",
		`{"name":"writer","scopes":"admin"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, 未知 scope 应返回 400, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "不支持") {
		t.Errorf("错误信息应是中文: %s", recorder.Body.String())
	}

	keys, err := module.service.repo.listAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("listAPIKeys 返回错误: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("被拒绝的创建不应落库, Key 数量 = %d", len(keys))
	}
}

func TestAdminCreateKeyWithoutScopesStaysSearchOnly(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	createAgentKey(t, engine, `{"name":"reader"}`)

	keys, err := module.service.repo.listAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("listAPIKeys 返回错误: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("Key 数量 = %d", len(keys))
	}
	if keys[0].Scopes != "" {
		t.Errorf("Scopes = %q, 不带 scopes 应保持空串（仅 search）", keys[0].Scopes)
	}
	if keys[0].Byline != "" {
		t.Errorf("Byline = %q, 不带 authorName 应为空", keys[0].Byline)
	}
	if !keys[0].HasScope(ScopeSearch) {
		t.Error("新 Key 应保有 search 语义")
	}
}

func TestAdminUpdateKeyScopesAndByline(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)
	created := createAgentKey(t, engine, `{"name":"agent"}`)
	path := "/api/admin/gateway/keys/" + strconv.Itoa(created.Data.ID)

	// 更新 scopes 生效，且不碰 authorName
	recorder := doAdmin(engine, http.MethodPut, path, `{"scopes":" content:write "}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("更新 scopes 状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	key := apiKeyFromRepo(t, module, created.Data.ID)
	if key.Scopes != "content:write" {
		t.Errorf("更新后 Scopes = %q, 期望 content:write", key.Scopes)
	}
	if key.Byline != "" {
		t.Errorf("更新 scopes 不应改动 Byline: %q", key.Byline)
	}

	// 去掉 scope：空串应让 key 回到仅 search
	recorder = doAdmin(engine, http.MethodPut, path, `{"scopes":""}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("清空 scopes 状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	key = apiKeyFromRepo(t, module, created.Data.ID)
	if key.Scopes != "" {
		t.Errorf("清空后 Scopes = %q, 应为空串", key.Scopes)
	}

	// 设置署名
	recorder = doAdmin(engine, http.MethodPut, path, `{"authorName":"Alice"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("设置署名状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	key = apiKeyFromRepo(t, module, created.Data.ID)
	if key.Byline != "Alice" {
		t.Errorf("设置后 Byline = %q, 期望 Alice", key.Byline)
	}
	if key.Scopes != "" {
		t.Errorf("设置署名不应改动 Scopes: %q", key.Scopes)
	}

	// 空串署名 = 置空，指针区分"不改"与"清空"
	recorder = doAdmin(engine, http.MethodPut, path, `{"authorName":""}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("清空署名状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	key = apiKeyFromRepo(t, module, created.Data.ID)
	if key.Byline != "" {
		t.Errorf("清空后 Byline = %q, 应为空串", key.Byline)
	}
}

func TestAdminUpdateKeyOmittingScopesAndBylineLeavesThemAlone(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)
	created := createAgentKey(t, engine,
		`{"name":"agent","scopes":"content:write","authorName":"Alice"}`)

	recorder := doAdmin(engine, http.MethodPut,
		"/api/admin/gateway/keys/"+strconv.Itoa(created.Data.ID), `{"name":"renamed"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	key := apiKeyFromRepo(t, module, created.Data.ID)
	if key.Name != "renamed" {
		t.Errorf("Name = %q, name 更新应生效", key.Name)
	}
	if key.Scopes != "content:write" || key.Byline != "Alice" {
		t.Errorf("未带 scopes/authorName 的更新不应动到它们: Scopes=%q Byline=%q", key.Scopes, key.Byline)
	}
}

func apiKeyFromRepo(t *testing.T, module *Module, id int) APIKey {
	t.Helper()
	key, err := module.service.repo.apiKeyByID(context.Background(), id)
	if err != nil {
		t.Fatalf("apiKeyByID(%d) 返回错误: %v", id, err)
	}
	return key
}

func TestAdminUpdateKeyRejectsUnknownScope(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)
	created := createAgentKey(t, engine, `{"name":"agent"}`)

	recorder := doAdmin(engine, http.MethodPut,
		"/api/admin/gateway/keys/"+strconv.Itoa(created.Data.ID), `{"scopes":"admin"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, 未知 scope 的更新应返回 400, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "不支持") {
		t.Errorf("错误信息应是中文: %s", recorder.Body.String())
	}
	key := apiKeyFromRepo(t, module, created.Data.ID)
	if key.Scopes != "" {
		t.Errorf("被拒绝的更新不应落库, Scopes = %q", key.Scopes)
	}
}

func TestAPIKeyViewsCarriesScopesAndAuthorName(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)
	created := createAgentKey(t, engine,
		`{"name":"writer","scopes":"search,content:write","authorName":"博客助手"}`)

	views, err := module.service.apiKeyViews(context.Background())
	if err != nil {
		t.Fatalf("apiKeyViews 返回错误: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("条数 = %d", len(views))
	}
	view := views[0]
	if view.ID != created.Data.ID {
		t.Fatalf("视图 ID = %d, 期望 %d", view.ID, created.Data.ID)
	}
	if view.Scopes != "search,content:write" {
		t.Errorf("Scopes = %q", view.Scopes)
	}
	if view.AuthorName != "博客助手" {
		t.Errorf("AuthorName = %q", view.AuthorName)
	}

	// KeyPlain 永远不出现在视图里（既有保证，随新字段一起回归）
	body, err := json.Marshal(views)
	if err != nil {
		t.Fatalf("marshal views: %v", err)
	}
	if strings.Contains(string(body), created.Data.APIKey) {
		t.Fatal("apiKeyViews 不应泄露明文 Key")
	}
}
