package agentapi

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate agentapi models: %v", err)
	}
	return db
}

// newTestService builds a grant service backed by an in-memory database with
// an injected clock, so every expiry-sensitive assertion uses a known instant.
func newTestService(t *testing.T, now time.Time) (*grantService, *grantRepository) {
	t.Helper()
	repo := &grantRepository{db: openTestDB(t)}
	service := newGrantService(repo)
	service.now = func() time.Time { return now }
	return service, repo
}
