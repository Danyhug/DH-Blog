package share

import (
	"context"
	"testing"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestModule(t *testing.T) *Module {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	module := New(Dependencies{DB: db})
	t.Cleanup(module.Shutdown)
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate share models: %v", err)
	}
	return module
}

func TestModuleRegistersShareRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	module := newTestModule(t)
	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	module.RegisterRoutes(routes)

	want := map[string]bool{
		"GET /api/share/:shareId":          false,
		"POST /api/share/:shareId/verify":  false,
		"GET /api/share/:shareId/download": false,
		"POST /api/files/share":            false,
		"GET /api/files/share":             false,
		"GET /api/files/share/:id":         false,
		"DELETE /api/files/share/:id":      false,
		"GET /api/files/share/:id/logs":    false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route %s was not registered", route)
		}
	}
}

func TestMigrationModelsPreserveTableNames(t *testing.T) {
	models := MigrationModels()
	if len(models) != 2 {
		t.Fatalf("MigrationModels() len = %d, want 2", len(models))
	}
	if _, ok := models[0].(*Share); !ok {
		t.Fatalf("MigrationModels()[0] type = %T, want *Share", models[0])
	}
	if _, ok := models[1].(*ShareAccessLog); !ok {
		t.Fatalf("MigrationModels()[1] type = %T, want *ShareAccessLog", models[1])
	}
	if got := (Share{}).TableName(); got != "shares" {
		t.Fatalf("Share.TableName() = %q, want shares", got)
	}
	if got := (ShareAccessLog{}).TableName(); got != "share_access_logs" {
		t.Fatalf("ShareAccessLog.TableName() = %q, want share_access_logs", got)
	}
}

func TestRepositoryPersistsAndCountsShares(t *testing.T) {
	module := newTestModule(t)
	ctx := context.Background()
	first := &Share{ShareID: "first", FileKey: "file-1", CreatedAt: model.JSONTime{Time: time.Now()}}
	second := &Share{ShareID: "second", FileKey: "file-1", CreatedAt: model.JSONTime{Time: time.Now()}}
	if err := module.repository.Create(ctx, first); err != nil {
		t.Fatalf("create first share: %v", err)
	}
	if err := module.repository.Create(ctx, second); err != nil {
		t.Fatalf("create second share: %v", err)
	}
	if err := module.repository.IncrementViewCount(ctx, "first"); err != nil {
		t.Fatalf("increment view count: %v", err)
	}
	if err := module.repository.IncrementDownloadCount(ctx, "first"); err != nil {
		t.Fatalf("increment download count: %v", err)
	}

	got, err := module.repository.FindByShareID(ctx, "first")
	if err != nil {
		t.Fatalf("find by share ID: %v", err)
	}
	if got.ViewCount != 1 || got.DownloadCount != 1 {
		t.Fatalf("counts = (%d, %d), want (1, 1)", got.ViewCount, got.DownloadCount)
	}
	byFile, err := module.repository.FindByFileKey(ctx, "file-1")
	if err != nil {
		t.Fatalf("find by file key: %v", err)
	}
	if len(byFile) != 2 {
		t.Fatalf("shares for file = %d, want 2", len(byFile))
	}
	page, total, err := module.repository.ListByPage(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	if total != 2 || len(page) != 1 || page[0].ShareID != "second" {
		t.Fatalf("page = %#v, total = %d; want newest share and total 2", page, total)
	}
}

func TestAccessLogRepositoryPaginatesByShare(t *testing.T) {
	module := newTestModule(t)
	ctx := context.Background()
	for _, log := range []*ShareAccessLog{
		{ShareID: "target", ActionType: ShareActionView},
		{ShareID: "other", ActionType: ShareActionView},
		{ShareID: "target", ActionType: ShareActionDownload},
	} {
		if err := module.accessLogRepository.Create(ctx, log); err != nil {
			t.Fatalf("create access log: %v", err)
		}
	}
	logs, total, err := module.accessLogRepository.ListByShareID(ctx, "target", 1, 10)
	if err != nil {
		t.Fatalf("list access logs: %v", err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("logs = %d, total = %d; want 2, 2", len(logs), total)
	}
	if logs[0].ActionType != ShareActionDownload {
		t.Fatalf("newest action = %q, want download", logs[0].ActionType)
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	module := New(Dependencies{DB: db})
	module.Shutdown()
	module.Shutdown()
	select {
	case <-module.service.tokens.done:
	default:
		t.Fatal("token cleanup worker did not stop")
	}
}
