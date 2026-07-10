package files

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
)

func TestModuleRegistersFileUploadAndPublicRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	storagePath := t.TempDir()
	module := New(Dependencies{
		DB:                 openTestDB(t),
		InitialStoragePath: storagePath,
		StaticFilesPath:    t.TempDir(),
	})

	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	module.RegisterRoutes(routes)

	want := map[string]bool{
		"GET /api/files/list":                          false,
		"POST /api/files/folder":                       false,
		"GET /api/files/download/:id":                  false,
		"PUT /api/files/rename/:id":                    false,
		"DELETE /api/files/:id":                        false,
		"GET /api/files/directory-tree":                false,
		"POST /api/files/sync":                         false,
		"POST /api/files/upload/chunk/init":            false,
		"POST /api/files/upload/chunk/chunk":           false,
		"POST /api/files/upload/chunk/complete":        false,
		"GET /api/files/upload/chunk/:uploadId/chunks": false,
		"DELETE /api/files/upload/chunk/:uploadId":     false,
		"GET /api/博客/*filepath":                        false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route not registered: %s", route)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/files/list", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list status=%d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestMigrationModelsOwnFilesTable(t *testing.T) {
	models := MigrationModels()
	if len(models) != 1 {
		t.Fatalf("got %d migration models, want 1", len(models))
	}
	file, ok := models[0].(*File)
	if !ok {
		t.Fatalf("migration model type=%T, want *files.File", models[0])
	}
	if file.TableName() != "files" {
		t.Fatalf("table name=%q, want files", file.TableName())
	}
}
