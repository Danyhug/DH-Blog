package files

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate files: %v", err)
	}
	return db
}
