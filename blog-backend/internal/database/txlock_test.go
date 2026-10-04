package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// A transaction that reads before it writes must survive another connection
// committing in between; with deferred BEGIN this fails with SQLITE_BUSY_SNAPSHOT (517).
func TestTransactionReadThenWriteSurvivesConcurrentWriter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(dsn(filepath.Join(t.TempDir(), "lock.db"))), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO items (name) VALUES ('a')").Error; err != nil {
		t.Fatal(err)
	}

	other := make(chan error, 1)
	err = db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Raw("SELECT COUNT(*) FROM items").Scan(&count).Error; err != nil {
			return err
		}
		// Another pooled connection writes while this transaction is open.
		go func() { other <- db.Exec("INSERT INTO items (name) VALUES ('b')").Error }()
		time.Sleep(200 * time.Millisecond)
		return tx.Exec("DELETE FROM items WHERE name = 'a'").Error
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
	if err := <-other; err != nil {
		t.Fatalf("concurrent writer failed: %v", err)
	}
}
