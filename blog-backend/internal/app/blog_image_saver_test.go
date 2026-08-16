package app

import (
	"context"
	"strings"
	"testing"

	filesmodule "dh-blog/internal/modules/files"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestBlogImageSaverOwnsUploadsToAdminUser reproduces the review finding: blog
// images were uploaded as user 0, and the backend Files UI's ownership check
// (service_metadata.go GetDownloadInfo) then refuses to preview or manage them.
// SaveBlogImage must land as the admin user (id 1), the same owner the fixed
// 博客 protected directory is created under.
func TestBlogImageSaverOwnsUploadsToAdminUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(filesmodule.MigrationModels()...); err != nil {
		t.Fatalf("migrate files models: %v", err)
	}

	module := filesmodule.New(filesmodule.Dependencies{
		DB:                 db,
		StaticFilesPath:    t.TempDir(),
		InitialStoragePath: t.TempDir(),
		InitialChunkSizeKB: 5120,
	})
	// registry.go 里跟真实装配一致：固定目录由 system() 构建时的
	// EnsureProtectedDirectories 建好，SaveBlogImage 依赖它存在。
	if err := module.Service().EnsureProtectedDirectories(context.Background()); err != nil {
		t.Fatalf("EnsureProtectedDirectories: %v", err)
	}
	saver := blogImageSaver{files: module.Service()}

	url, err := saver.SaveBlogImage(context.Background(), "cover.png", []byte("png-bytes"))
	if err != nil {
		t.Fatalf("SaveBlogImage: %v", err)
	}
	if !strings.HasPrefix(url, "/api/博客/") {
		t.Fatalf("url = %q, want /api/博客/ prefix (admin-editor shape)", url)
	}

	storagePath := strings.TrimPrefix(url, "/api/")
	var stored filesmodule.File
	if err := db.Where("storage_path = ?", storagePath).First(&stored).Error; err != nil {
		t.Fatalf("load uploaded file record: %v", err)
	}
	if stored.UserID != 1 {
		t.Fatalf("image UserID = %d, want 1 (admin, so the Files UI can manage it)", stored.UserID)
	}
}
