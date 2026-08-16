package agentapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newTestModule(t *testing.T, now time.Time) (*Module, *gorm.DB) {
	t.Helper()
	service, repo := newTestService(t, now)
	return &Module{service: service, handler: newGrantHandler(service)}, repo.db
}

func grantEngine(t *testing.T, module *Module) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	routes := &router.Routes{Engine: engine, PublicAPI: engine.Group("/api"), AdminAPI: engine.Group("/api/admin")}
	module.RegisterRoutes(routes)
	return engine
}

func doRequest(t *testing.T, engine *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response %s: %v", recorder.Body.String(), err)
	}
	return recorder, payload
}

func TestHandlerCreateGrantReturnsPlaintextOnce(t *testing.T) {
	module, db := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	recorder, payload := doRequest(t, engine, http.MethodPost, "/api/admin/agent/grants", `{"note":"让 Claude 改错别字"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if code, _ := payload["code"].(float64); code != 1 {
		t.Fatalf("code = %v, want 1", payload["code"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %#v", payload["data"])
	}
	token, _ := data["token"].(string)
	if !strings.HasPrefix(token, "ag_grant_") || len(token) != len("ag_grant_")+32 {
		t.Fatalf("token = %q, want ag_grant_ + 32 chars", token)
	}
	if id, _ := data["id"].(float64); id <= 0 {
		t.Fatalf("id = %v, want > 0", data["id"])
	}
	expireAt, _ := data["expireAt"].(string)
	if want := fixedNow().Add(grantTTL).Format("2006-01-02 15:04:05"); expireAt != want {
		t.Fatalf("expireAt = %q, want %q", expireAt, want)
	}
	if articleID, _ := data["articleId"].(float64); articleID != 0 {
		t.Fatalf("articleId = %v, want 0", data["articleId"])
	}
	if note, _ := data["note"].(string); note != "让 Claude 改错别字" {
		t.Fatalf("note = %q", note)
	}

	var stored EditGrant
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("load stored grant: %v", err)
	}
	if stored.TokenPlain != token {
		t.Fatalf("stored plaintext = %q, want issued %q", stored.TokenPlain, token)
	}
}

func TestHandlerCreateGrantAcceptsArticleBinding(t *testing.T) {
	module, _ := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	_, payload := doRequest(t, engine, http.MethodPost, "/api/admin/agent/grants", `{"note":"只改一篇","articleId":42}`)
	data, _ := payload["data"].(map[string]any)
	if articleID, _ := data["articleId"].(float64); articleID != 42 {
		t.Fatalf("articleId = %v, want 42", data["articleId"])
	}
}

func TestHandlerCreateGrantRejectsMalformedBody(t *testing.T) {
	module, _ := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	recorder, _ := doRequest(t, engine, http.MethodPost, "/api/admin/agent/grants", `{"note":`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestHandlerCreateGrantRejectsNegativeArticleID(t *testing.T) {
	module, _ := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	recorder, payload := doRequest(t, engine, http.MethodPost, "/api/admin/agent/grants", `{"articleId":-1}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", recorder.Code, recorder.Body.String())
	}
	if msg, _ := payload["msg"].(string); !strings.Contains(msg, "articleId 不能为负数") {
		t.Fatalf("msg = %q, want articleId 不能为负数", msg)
	}
}

func TestHandlerGrantsRejectNonNumericID(t *testing.T) {
	module, _ := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/admin/agent/grants/abc/reveal"},
		{http.MethodDelete, "/api/admin/agent/grants/abc"},
	}
	for _, tc := range cases {
		recorder, payload := doRequest(t, engine, tc.method, tc.path, "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s %s status = %d, want 400, body = %s", tc.method, tc.path, recorder.Code, recorder.Body.String())
		}
		if msg, _ := payload["msg"].(string); msg != "无效的 ID" {
			t.Fatalf("%s %s msg = %q, want 无效的 ID", tc.method, tc.path, msg)
		}
	}
}

func TestHandlerListGrantsShowsPrefixNotPlaintext(t *testing.T) {
	module, db := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	grant := EditGrant{
		TokenPrefix: "ag_grant_abc12345",
		TokenHash:   "deadbeef",
		TokenPlain:  "ag_grant_secret-plaintext-000",
		ExpireAt:    jsonTime(fixedNow().Add(grantTTL)),
		Note:        "让 Claude 改错别字",
	}
	if err := db.Create(&grant).Error; err != nil {
		t.Fatalf("seed grant: %v", err)
	}

	recorder, payload := doRequest(t, engine, http.MethodGet, "/api/admin/agent/grants", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	rawList, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("data = %#v, want a list", payload["data"])
	}
	if len(rawList) != 1 {
		t.Fatalf("list length = %d, want 1", len(rawList))
	}
	item, _ := rawList[0].(map[string]any)
	if item["tokenPrefix"] != "ag_grant_abc12345" {
		t.Fatalf("tokenPrefix = %v", item["tokenPrefix"])
	}
	if item["note"] != "让 Claude 改错别字" {
		t.Fatalf("note = %v", item["note"])
	}
	if _, leaked := item["token"]; leaked {
		t.Fatal("list leaked a plaintext token field")
	}
	if _, leaked := item["tokenPlain"]; leaked {
		t.Fatal("list leaked a tokenPlain field")
	}
	raw, _ := json.Marshal(item)
	if strings.Contains(string(raw), "secret-plaintext") {
		t.Fatalf("list leaked the plaintext inside %s", raw)
	}
}

func TestHandlerRevealGrant(t *testing.T) {
	module, _ := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	_, created := doRequest(t, engine, http.MethodPost, "/api/admin/agent/grants", `{"note":"x"}`)
	data, _ := created["data"].(map[string]any)
	id := int(data["id"].(float64))
	token, _ := data["token"].(string)

	recorder, payload := doRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/admin/agent/grants/%d/reveal", id), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	revealed, _ := payload["data"].(map[string]any)
	if revealed["token"] != token {
		t.Fatalf("reveal = %v, want %q", revealed["token"], token)
	}

	recorder, _ = doRequest(t, engine, http.MethodGet, "/api/admin/agent/grants/4242/reveal", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("reveal unknown status = %d, want 404", recorder.Code)
	}
}

// TestHandlerRevealExpiredUnsweptGrant pins today's behaviour: Reveal checks
// only revocation, not expiry, so a grant that expired but was never swept
// (no new grant issued since) still yields its plaintext. The sweep happens
// as a side effect of Grant, not of Reveal.
func TestHandlerRevealExpiredUnsweptGrantStillReturnsPlaintext(t *testing.T) {
	module, db := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	_, created := doRequest(t, engine, http.MethodPost, "/api/admin/agent/grants", `{"note":"x"}`)
	data, _ := created["data"].(map[string]any)
	id := int(data["id"].(float64))
	token, _ := data["token"].(string)

	past := fixedNow().Add(-time.Minute)
	if err := db.Model(&EditGrant{}).Where("id = ?", id).Update("expire_at", past).Error; err != nil {
		t.Fatalf("backdate grant: %v", err)
	}

	recorder, payload := doRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/admin/agent/grants/%d/reveal", id), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", recorder.Code, recorder.Body.String())
	}
	revealed, _ := payload["data"].(map[string]any)
	if revealed["token"] != token {
		t.Fatalf("reveal = %v, want %q", revealed["token"], token)
	}
}

func TestHandlerRevokeGrantBlocksRevealAndValidate(t *testing.T) {
	module, _ := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)
	service := module.service

	_, created := doRequest(t, engine, http.MethodPost, "/api/admin/agent/grants", `{"note":"x"}`)
	data, _ := created["data"].(map[string]any)
	id := int(data["id"].(float64))
	token, _ := data["token"].(string)

	if _, err := service.Validate(0, token); err != nil {
		t.Fatalf("validate before revoke: %v", err)
	}

	recorder, payload := doRequest(t, engine, http.MethodDelete, fmt.Sprintf("/api/admin/agent/grants/%d", id), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if code, _ := payload["code"].(float64); code != 1 {
		t.Fatalf("revoke code = %v, want 1", payload["code"])
	}

	recorder, _ = doRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/admin/agent/grants/%d/reveal", id), "")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("reveal after revoke status = %d, want 403", recorder.Code)
	}
	if _, err := service.Validate(0, token); err != ErrGrantRevoked {
		t.Fatalf("validate after revoke err = %v, want ErrGrantRevoked", err)
	}
}

func TestHandlerRevokeUnknownGrantIs404(t *testing.T) {
	module, _ := newTestModule(t, fixedNow())
	engine := grantEngine(t, module)

	recorder, _ := doRequest(t, engine, http.MethodDelete, "/api/admin/agent/grants/4242", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func jsonTime(t time.Time) model.JSONTime { return model.JSONTime{Time: t} }
