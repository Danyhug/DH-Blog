package user_test

import (
	"testing"

	"dh-blog/internal/modules/user"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestModuleOwnsUserModelAndRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(user.MigrationModels()...); err != nil {
		t.Fatalf("migrate user models: %v", err)
	}

	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	user.New(db, nil).RegisterRoutes(routes)

	want := map[string]bool{
		"POST /api/user/login": false,
		"POST /api/user/check": false,
		"GET /api/user/heart":  false,
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
