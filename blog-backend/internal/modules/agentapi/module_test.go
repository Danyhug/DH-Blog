package agentapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
)

func TestModuleRejectsNilDB(t *testing.T) {
	if _, err := New(Dependencies{}); err == nil {
		t.Fatal("New with nil DB should fail")
	}
}

func TestModuleRejectsNilArticlesDependency(t *testing.T) {
	db := openTestDB(t)
	if _, err := New(Dependencies{DB: db}); err == nil {
		t.Fatal("New with nil Articles should fail")
	}
}

func TestModuleMigrationModelsContainEditGrant(t *testing.T) {
	models := MigrationModels()
	found := false
	for _, m := range models {
		if _, ok := m.(*EditGrant); ok {
			found = true
		}
	}
	if !found {
		t.Fatalf("MigrationModels() = %#v, want EditGrant included", models)
	}
}

func TestModuleRegistersGrantRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	module, _ := newTestModule(t, fixedNow())
	engine := gin.New()
	routes := &router.Routes{Engine: engine, PublicAPI: engine.Group("/api"), AdminAPI: engine.Group("/api/admin")}
	module.RegisterRoutes(routes)

	want := map[string]bool{
		"POST /api/admin/agent/grants":           false,
		"GET /api/admin/agent/grants":            false,
		"GET /api/admin/agent/grants/:id/reveal": false,
		"DELETE /api/admin/agent/grants/:id":     false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := want[key]; exists {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route %s was not registered", route)
		}
	}
}

func TestEditGrantTableNameAndJSONShape(t *testing.T) {
	if (EditGrant{}).TableName() != "agent_edit_grants" {
		t.Fatalf("table name = %q, want agent_edit_grants", (EditGrant{}).TableName())
	}

	var grant EditGrant
	body := `{"id":1,"tokenPrefix":"ag_grant_abc","expireAt":"2026-08-16 11:00:00",` +
		`"articleId":42,"revoked":false,"usedCount":2,"note":"备注","createTime":"2026-08-16 10:00:00"}`
	if err := json.Unmarshal([]byte(body), &grant); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if grant.TokenPrefix != "ag_grant_abc" || grant.ArticleID != 42 || grant.UsedCount != 2 || grant.Note != "备注" {
		t.Fatalf("grant = %#v", grant)
	}
	// JSONTime parses zone-less input as UTC then converts to Local; replicate
	// that exactly so the instant comparison is location-independent.
	parsed, err := time.Parse("2006-01-02 15:04:05", "2026-08-16 11:00:00")
	if err != nil {
		t.Fatalf("parse fixed time: %v", err)
	}
	if !grant.ExpireAt.Equal(jsonTime(parsed.Local()).Time) {
		t.Fatalf("expireAt = %v, want %v", grant.ExpireAt.Time, parsed.Local())
	}

	// Marshal via a pointer: encoding/json only honours JSONTime's pointer
	// receiver MarshalJSON on addressable values, and the handler deliberately
	// formats times by hand for that reason (see grantView). Pin a known Local
	// instant so the format assertion holds in any timezone.
	grant.ExpireAt = jsonTime(time.Date(2026, 8, 16, 11, 0, 0, 0, time.Local))
	encoded, err := json.Marshal(&grant)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	asString := string(encoded)
	if asString == "" {
		t.Fatal("marshal returned empty output")
	}
	// 密文与明文都不能出现在任何 JSON 输出里；expireAt 走仓库的日期格式
	for _, secret := range []string{"tokenHash", "tokenPlain"} {
		if strings.Contains(asString, secret) {
			t.Fatalf("marshal leaked %s in %s", secret, asString)
		}
	}
	if !strings.Contains(asString, `"expireAt":"2026-08-16 11:00:00"`) {
		t.Fatalf("marshal = %s, want the repo date format for expireAt", asString)
	}
}
