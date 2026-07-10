package database

import (
	"errors"
	"testing"
)

func TestInitRequiresMigrationModels(t *testing.T) {
	db, err := Init(nil)
	if db != nil {
		t.Fatal("Init() returned a database without migration models")
	}
	if !errors.Is(err, ErrMigrationModelsRequired) {
		t.Fatalf("Init() error = %v, want %v", err, ErrMigrationModelsRequired)
	}
}
