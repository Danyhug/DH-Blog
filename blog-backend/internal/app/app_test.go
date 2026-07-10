package app_test

import (
	"testing"

	"dh-blog/internal/app"
	"dh-blog/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestApplicationCompositionAndLifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(app.SchemaModels()...); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	application, err := app.New(config.DefaultConfig(), db)
	if err != nil {
		t.Fatalf("compose application: %v", err)
	}
	application.Start()
	application.Start()
	application.Shutdown()
	application.Shutdown()

	if application.Router == nil {
		t.Fatal("application router was not initialized")
	}
}
