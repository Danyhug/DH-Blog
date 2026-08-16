package share

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"dh-blog/internal/model"
	filesmodule "dh-blog/internal/modules/files"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestModule(t *testing.T) *Module {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	module := New(Dependencies{DB: db})
	t.Cleanup(module.Shutdown)
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate share models: %v", err)
	}
	return module
}

func TestModuleRegistersShareRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	module := newTestModule(t)
	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	module.RegisterRoutes(routes)

	want := map[string]bool{
		"GET /api/share/:shareId":          false,
		"POST /api/share/:shareId/verify":  false,
		"GET /api/share/:shareId/download": false,
		"POST /api/files/share":            false,
		"GET /api/files/share":             false,
		"DELETE /api/files/share/:id":      false,
		"GET /api/files/share/:id/logs":    false,
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

func TestMigrationModelsPreserveTableNames(t *testing.T) {
	models := MigrationModels()
	if len(models) != 2 {
		t.Fatalf("MigrationModels() len = %d, want 2", len(models))
	}
	if _, ok := models[0].(*Share); !ok {
		t.Fatalf("MigrationModels()[0] type = %T, want *Share", models[0])
	}
	if _, ok := models[1].(*ShareAccessLog); !ok {
		t.Fatalf("MigrationModels()[1] type = %T, want *ShareAccessLog", models[1])
	}
	if got := (Share{}).TableName(); got != "shares" {
		t.Fatalf("Share.TableName() = %q, want shares", got)
	}
	if got := (ShareAccessLog{}).TableName(); got != "share_access_logs" {
		t.Fatalf("ShareAccessLog.TableName() = %q, want share_access_logs", got)
	}
}

func TestRepositoryPersistsAndCountsShares(t *testing.T) {
	module := newTestModule(t)
	ctx := context.Background()
	first := &Share{ShareID: "first", FileKey: "file-1", CreatedAt: model.JSONTime{Time: time.Now()}}
	second := &Share{ShareID: "second", FileKey: "file-1", CreatedAt: model.JSONTime{Time: time.Now()}}
	if err := module.repository.Create(ctx, first); err != nil {
		t.Fatalf("create first share: %v", err)
	}
	if err := module.repository.Create(ctx, second); err != nil {
		t.Fatalf("create second share: %v", err)
	}
	if err := module.repository.IncrementViewCount(ctx, "first"); err != nil {
		t.Fatalf("increment view count: %v", err)
	}
	if err := module.repository.IncrementDownloadCount(ctx, "first"); err != nil {
		t.Fatalf("increment download count: %v", err)
	}

	got, err := module.repository.FindByShareID(ctx, "first")
	if err != nil {
		t.Fatalf("find by share ID: %v", err)
	}
	if got.ViewCount != 1 || got.DownloadCount != 1 {
		t.Fatalf("counts = (%d, %d), want (1, 1)", got.ViewCount, got.DownloadCount)
	}
	byFile, err := module.repository.FindByFileKey(ctx, "file-1")
	if err != nil {
		t.Fatalf("find by file key: %v", err)
	}
	if len(byFile) != 2 {
		t.Fatalf("shares for file = %d, want 2", len(byFile))
	}
	page, total, err := module.repository.ListByPage(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	if total != 2 || len(page) != 1 || page[0].ShareID != "second" {
		t.Fatalf("page = %#v, total = %d; want newest share and total 2", page, total)
	}
}

func TestAccessLogRepositoryPaginatesByShare(t *testing.T) {
	module := newTestModule(t)
	ctx := context.Background()
	for _, log := range []*ShareAccessLog{
		{ShareID: "target", ActionType: ShareActionView},
		{ShareID: "other", ActionType: ShareActionView},
		{ShareID: "target", ActionType: ShareActionDownload},
	} {
		if err := module.accessLogRepository.Create(ctx, log); err != nil {
			t.Fatalf("create access log: %v", err)
		}
	}
	logs, total, err := module.accessLogRepository.ListByShareID(ctx, "target", 1, 10)
	if err != nil {
		t.Fatalf("list access logs: %v", err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("logs = %d, total = %d; want 2, 2", len(logs), total)
	}
	if logs[0].ActionType != ShareActionDownload {
		t.Fatalf("newest action = %q, want download", logs[0].ActionType)
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	module := New(Dependencies{DB: db})
	module.Shutdown()
	module.Shutdown()
	select {
	case <-module.service.tokens.done:
	default:
		t.Fatal("token cleanup worker did not stop")
	}
}

// stubFileService 只实现分享模块真正会用到的取文件信息，其余方法为空实现。
type stubFileService struct {
	files map[string]*filesmodule.File
}

func (s stubFileService) UploadFile(context.Context, uint64, string, string, int64, io.Reader) (*filesmodule.File, error) {
	return nil, errors.New("not implemented")
}
func (s stubFileService) GetDownloadInfo(_ context.Context, _ uint64, fileID string) (*filesmodule.File, error) {
	if file, ok := s.files[fileID]; ok {
		return file, nil
	}
	return nil, errors.New("文件不存在")
}
func (s stubFileService) GetDownloadInfoForShare(_ context.Context, fileID string) (*filesmodule.File, error) {
	if file, ok := s.files[fileID]; ok {
		return file, nil
	}
	return nil, errors.New("文件不存在")
}
func (s stubFileService) GetStoragePath() string                           { return "" }
func (s stubFileService) EnsureProtectedDirectories(context.Context) error { return nil }
func (s stubFileService) GetProtectedDirectoryID(context.Context, string) (string, error) {
	return "", nil
}
func (s stubFileService) SyncFilesFromDiskDebounced() {}

func newTestModuleWithFiles(t *testing.T, files map[string]*filesmodule.File) *Module {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	module := New(Dependencies{DB: db, FileService: stubFileService{files: files}})
	t.Cleanup(module.Shutdown)
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate share models: %v", err)
	}
	return module
}

func TestListSharesResolvesFileAndHidesPassword(t *testing.T) {
	module := newTestModuleWithFiles(t, map[string]*filesmodule.File{
		"file-1": {Name: "报告.pdf", Size: 2048},
	})
	ctx := context.Background()
	expired := time.Now().Add(-time.Hour)
	share := &Share{
		ShareID:   "abc123",
		FileKey:   "file-1",
		Password:  "$2a$10$hashedsecret",
		ExpireAt:  &expired,
		CreatedAt: model.JSONTime{Time: time.Now()},
	}
	if err := module.repository.Create(ctx, share); err != nil {
		t.Fatalf("create share: %v", err)
	}

	summaries, total, err := module.Service().ListShares(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	if total != 1 || len(summaries) != 1 {
		t.Fatalf("list returned %d shares (total %d), want 1", len(summaries), total)
	}
	got := summaries[0]
	if got.FileName != "报告.pdf" || got.FileSize != 2048 {
		t.Fatalf("file info = %q/%d, want 报告.pdf/2048", got.FileName, got.FileSize)
	}
	if !got.HasPassword {
		t.Fatal("has_password should be true for a password-protected share")
	}
	if !got.IsExpired {
		t.Fatal("is_expired should be true for a share past its expiry")
	}
	if got.FileMissing {
		t.Fatal("file_missing should be false when the file still exists")
	}

	// 管理端不应该拿到密码：ShareSummary 里根本没有这个字段
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}
	if strings.Contains(string(encoded), "hashedsecret") || strings.Contains(string(encoded), `"password"`) {
		t.Fatalf("share summary leaked the password: %s", encoded)
	}
}

func TestListSharesFlagsDeletedFiles(t *testing.T) {
	module := newTestModuleWithFiles(t, map[string]*filesmodule.File{})
	ctx := context.Background()
	share := &Share{ShareID: "gone", FileKey: "missing-file", CreatedAt: model.JSONTime{Time: time.Now()}}
	if err := module.repository.Create(ctx, share); err != nil {
		t.Fatalf("create share: %v", err)
	}

	summaries, _, err := module.Service().ListShares(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list shares should not fail when the file is gone: %v", err)
	}
	if len(summaries) != 1 || !summaries[0].FileMissing {
		t.Fatalf("summaries = %#v, want one entry flagged as file_missing", summaries)
	}
}
