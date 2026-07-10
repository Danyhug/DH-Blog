package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type failOnceCreateRepository struct {
	fileRepository
	fail bool
}

func (r *failOnceCreateRepository) Create(ctx context.Context, file *File) error {
	if r.fail {
		r.fail = false
		return errors.New("injected create failure")
	}
	return r.fileRepository.Create(ctx, file)
}

func TestApplyStorageConfigRestoresRuntimeWhenScanFails(t *testing.T) {
	oldPath := t.TempDir()
	newPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(oldPath, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newPath, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	baseRepository := newRepository(openTestDB(t))
	existing := &File{UserID: 1, Name: "old.txt", StoragePath: "old.txt", FileHash: "preserve-me"}
	if err := baseRepository.Create(context.Background(), existing); err != nil {
		t.Fatal(err)
	}
	repository := &failOnceCreateRepository{fileRepository: baseRepository, fail: true}
	service := newService(repository, oldPath, 1024)
	if err := service.ApplyStorageConfig(context.Background(), newPath, 2048); err == nil {
		t.Fatal("expected scan failure")
	}
	if service.GetStoragePath() != oldPath || service.ChunkSizeKB() != 1024 {
		t.Fatalf("runtime changed after failure: path=%q chunk=%d", service.GetStoragePath(), service.ChunkSizeKB())
	}
	restored, err := baseRepository.FindByID(context.Background(), existing.ID)
	if err != nil {
		t.Fatalf("original file ID was not restored: %v", err)
	}
	if restored.FileHash != existing.FileHash {
		t.Fatalf("restored metadata hash=%q want=%q", restored.FileHash, existing.FileHash)
	}
	if !restored.CreatedAt.Equal(existing.CreatedAt.Time) || !restored.UpdatedAt.Equal(existing.UpdatedAt.Time) {
		t.Fatalf("restored timestamps changed: created=%v/%v updated=%v/%v", restored.CreatedAt, existing.CreatedAt, restored.UpdatedAt, existing.UpdatedAt)
	}
}

func TestInitializeStorageConfigPreservesExistingFileIDs(t *testing.T) {
	db := openTestDB(t)
	repository := newRepository(db)
	oldPath := t.TempDir()
	service := newService(repository, oldPath, 1024)
	record := &File{UserID: 1, Name: "existing.txt", StoragePath: "existing.txt"}
	if err := repository.Create(context.Background(), record); err != nil {
		t.Fatal(err)
	}

	if err := service.InitializeStorageConfig(context.Background(), t.TempDir(), 2048); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.FindByID(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("existing file ID was lost during startup initialization: %v", err)
	}
	if stored.ID != record.ID {
		t.Fatalf("stored ID=%d want=%d", stored.ID, record.ID)
	}
	if err := service.ApplyStorageConfig(context.Background(), service.GetStoragePath(), 4096); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindByID(context.Background(), record.ID); err != nil {
		t.Fatalf("chunk-size-only update rebuilt the file index: %v", err)
	}
}

func TestServiceCreatesFoldersUploadsAndDownloadsFiles(t *testing.T) {
	storagePath := t.TempDir()
	repository := newRepository(openTestDB(t))
	service := newService(repository, storagePath, 5120)
	ctx := context.Background()

	folder, err := service.CreateFolder(ctx, 7, "", "docs")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if folder.ParentID != "" || folder.StoragePath != "docs" || !folder.IsFolder {
		t.Fatalf("unexpected folder: %+v", folder)
	}

	parentID := strconv.Itoa(folder.ID)
	uploaded, err := service.UploadFile(ctx, 7, parentID, "readme.txt", 999, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("upload file: %v", err)
	}
	if uploaded.Size != 5 || uploaded.ParentID != parentID || uploaded.StoragePath != filepath.Join("docs", "readme.txt") {
		t.Fatalf("unexpected uploaded file: %+v", uploaded)
	}
	content, err := os.ReadFile(filepath.Join(storagePath, "docs", "readme.txt"))
	if err != nil || string(content) != "hello" {
		t.Fatalf("read uploaded content: content=%q err=%v", content, err)
	}

	children, err := service.ListFiles(ctx, 7, parentID)
	if err != nil || len(children) != 1 || children[0].ID != uploaded.ID {
		t.Fatalf("list children: children=%v err=%v", children, err)
	}

	download, err := service.GetDownloadInfo(ctx, 7, strconv.Itoa(uploaded.ID))
	if err != nil {
		t.Fatalf("get download: %v", err)
	}
	if download.StoragePath != filepath.Join(storagePath, "docs", "readme.txt") {
		t.Fatalf("download path=%q", download.StoragePath)
	}
	if _, err := service.GetDownloadInfo(ctx, 8, strconv.Itoa(uploaded.ID)); err == nil {
		t.Fatal("expected another user to be denied")
	}
	if _, err := service.UploadFile(ctx, 7, parentID, "readme.txt", 5, strings.NewReader("again")); err == nil {
		t.Fatal("expected duplicate upload to fail")
	}
}
