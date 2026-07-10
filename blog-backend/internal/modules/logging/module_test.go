package logging

import (
	"testing"
	"time"

	"dh-blog/internal/middleware"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestModule(t *testing.T) (*Module, *gorm.DB, *memoryCache) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cache := newMemoryCache()
	module := New(db, cache)
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate logging models: %v", err)
	}
	return module, db, cache
}

func TestModuleRegistersLoggingRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	module, _, _ := newTestModule(t)
	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	module.RegisterRoutes(routes)

	want := map[string]bool{
		"GET /api/admin/log/overview/visitLog": false,
		"GET /api/admin/stats/daily":           false,
		"GET /api/admin/log/stats/visits":      false,
		"GET /api/admin/log/stats/monthly":     false,
		"GET /api/admin/log/stats/daily-chart": false,
		"POST /api/admin/ip/ban/:ip/:status":   false,
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

func TestMigrationModels(t *testing.T) {
	models := MigrationModels()
	if len(models) != 2 {
		t.Fatalf("MigrationModels() len = %d, want 2", len(models))
	}
	if _, ok := models[0].(*AccessLog); !ok {
		t.Fatalf("MigrationModels()[0] type = %T, want *AccessLog", models[0])
	}
	if _, ok := models[1].(*IPBlacklist); !ok {
		t.Fatalf("MigrationModels()[1] type = %T, want *IPBlacklist", models[1])
	}
	if got := (AccessLog{}).TableName(); got != "access_logs" {
		t.Fatalf("AccessLog.TableName() = %q", got)
	}
	if got := (IPBlacklist{}).TableName(); got != "ip_blacklist" {
		t.Fatalf("IPBlacklist.TableName() = %q", got)
	}
}

func TestIPServiceConvertsMiddlewareRecord(t *testing.T) {
	module, db, _ := newTestModule(t)
	module.repository.batchSize = 1
	wantTime := time.Date(2026, 7, 10, 12, 30, 0, 0, time.Local)
	record := middleware.AccessRecord{
		IPAddress:    "203.0.113.7",
		AccessDate:   wantTime,
		UserAgent:    "Linux; Firefox",
		RequestURL:   "/api/article/1?preview=true",
		City:         "测试省/测试市",
		ResourceType: "article",
	}
	if err := module.IPService().RecordRequest(record); err != nil {
		t.Fatalf("record request: %v", err)
	}

	var got AccessLog
	if err := db.First(&got).Error; err != nil {
		t.Fatalf("read access log: %v", err)
	}
	if got.IPAddress != record.IPAddress || got.UserAgent != record.UserAgent ||
		got.RequestURL != record.RequestURL || got.City != record.City ||
		got.ResourceType != record.ResourceType || !got.AccessDate.Equal(wantTime) {
		t.Fatalf("persisted access log = %#v, want record %#v", got, record)
	}
}
