package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncFilesFromDiskReportsChanges(t *testing.T) {
	storagePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(storagePath, "new.mp3"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	repository := newRepository(openTestDB(t))
	service := newService(repository, storagePath, 5120)
	ctx := context.Background()

	gone := &File{UserID: 1, Name: "轨迹.mp4", Size: 10, StoragePath: "轨迹.mp4", MimeType: "video/mp4"}
	if err := repository.Create(ctx, gone); err != nil {
		t.Fatal(err)
	}

	stats, err := service.SyncFilesFromDisk()
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	// The fixed 博客 directory is created after reconciliation and is not
	// counted; only the disk-vs-index differences are.
	if stats.Added != 1 || stats.Removed != 1 || stats.Updated != 0 {
		t.Fatalf("stats = %+v, want 1 added, 1 removed", stats)
	}
	if _, err := repository.FindByID(ctx, gone.ID); err == nil {
		t.Fatal("record of a file missing from disk should be removed")
	}
}

func TestSyncFilesFromDiskRefusesUnreachableRoot(t *testing.T) {
	storagePath := filepath.Join(t.TempDir(), "mounted")
	repository := newRepository(openTestDB(t))
	service := newService(repository, storagePath, 5120)
	ctx := context.Background()
	// newService creates the directory; simulate the disk going away afterwards.
	if err := os.RemoveAll(storagePath); err != nil {
		t.Fatal(err)
	}

	kept := &File{UserID: 1, Name: "a.mp4", StoragePath: "a.mp4"}
	if err := repository.Create(ctx, kept); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SyncFilesFromDisk(); err == nil {
		t.Fatal("sync against a missing root must fail")
	}
	if _, err := repository.FindByID(ctx, kept.ID); err != nil {
		t.Fatalf("index must survive an unreachable root: %v", err)
	}
}
