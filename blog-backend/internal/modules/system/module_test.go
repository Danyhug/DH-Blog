package system

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type storageRuntimeStub struct {
	path       string
	chunkSize  int
	err        error
	initCalls  int
	applyCalls int
}

func (s *storageRuntimeStub) InitializeStorageConfig(_ context.Context, path string, chunkSizeKB int) error {
	s.initCalls++
	if s.err != nil {
		return s.err
	}
	s.path, s.chunkSize = path, chunkSizeKB
	return nil
}

func (s *storageRuntimeStub) ApplyStorageConfig(_ context.Context, path string, chunkSizeKB int) error {
	s.applyCalls++
	if s.err != nil {
		return s.err
	}
	s.path, s.chunkSize = path, chunkSizeKB
	return nil
}

func TestNewAppliesPersistedStorageConfig(t *testing.T) {
	db := openSystemTestDB(t)
	path := t.TempDir()
	if err := db.Create([]Setting{{SettingKey: SettingKeyFileStoragePath, SettingValue: path, ConfigType: ConfigTypeStorage}, {SettingKey: SettingKeyWebDAVChunkSize, SettingValue: "2048", ConfigType: ConfigTypeStorage}}).Error; err != nil {
		t.Fatal(err)
	}
	runtime := &storageRuntimeStub{path: "initial", chunkSize: 5120}
	newSystemTestModule(t, db, runtime)
	if runtime.initCalls != 1 || runtime.applyCalls != 0 || runtime.path != path || runtime.chunkSize != 2048 {
		t.Fatalf("runtime not bootstrapped: %+v", runtime)
	}
}
func (s *storageRuntimeStub) GetStoragePath() string            { return s.path }
func (s *storageRuntimeStub) ProtectedDirectoryNames() []string { return []string{"博客"} }

func openSystemTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatal(err)
	}
	return db
}

func newSystemTestModule(t *testing.T, db *gorm.DB, storage StorageRuntime) *Module {
	t.Helper()
	if stub, ok := storage.(*storageRuntimeStub); ok && stub.path == "" {
		stub.path = t.TempDir()
		stub.chunkSize = 5120
	}
	cache := dhcache.NewCache()
	t.Cleanup(cache.Shutdown)
	module, err := New(Dependencies{DB: db, Cache: cache, DataDir: t.TempDir(), Storage: storage})
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestNewBackfillsDefaultsAndRepairsConfigTypes(t *testing.T) {
	db := openSystemTestDB(t)
	if err := db.Create(&Setting{SettingKey: SettingKeyAIModel, SettingValue: "old", ConfigType: ConfigTypeBlog}).Error; err != nil {
		t.Fatal(err)
	}
	runtime := &storageRuntimeStub{}
	newSystemTestModule(t, db, runtime)
	var count int64
	if err := db.Model(&Setting{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(len(DefaultSettings())) {
		t.Fatalf("settings=%d want=%d", count, len(DefaultSettings()))
	}
	var setting Setting
	if err := db.Where("setting_key = ?", SettingKeyAIModel).First(&setting).Error; err != nil {
		t.Fatal(err)
	}
	if setting.ConfigType != ConfigTypeAI {
		t.Fatalf("config type=%q", setting.ConfigType)
	}
	var chunk Setting
	if err := db.Where("setting_key = ?", SettingKeyWebDAVChunkSize).First(&chunk).Error; err != nil {
		t.Fatal(err)
	}
	if chunk.SettingValue != "5120" || chunk.ConfigType != ConfigTypeStorage {
		t.Fatalf("chunk=%+v", chunk)
	}
	var storagePath Setting
	if err := db.Where("setting_key = ?", SettingKeyFileStoragePath).First(&storagePath).Error; err != nil {
		t.Fatal(err)
	}
	if storagePath.SettingValue != runtime.path {
		t.Fatalf("persisted storage path=%q want=%q", storagePath.SettingValue, runtime.path)
	}
}

func TestNewUpgradesOnlyLegacyDefaultPrompts(t *testing.T) {
	tests := []struct {
		name          string
		legacyKey     string
		legacyValue   string
		upgradedValue string
		customKey     string
	}{
		{"tags", SettingKeyAIPromptGetTags, legacyDefaultTagsPrompt, DefaultTagsPrompt, SettingKeyAIPromptGetAbstract},
		{"abstract", SettingKeyAIPromptGetAbstract, legacyDefaultAbstractPrompt, DefaultAbstractPrompt, SettingKeyAIPromptGetTags},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openSystemTestDB(t)
			customPrompt := "custom prompt"
			if err := db.Create([]Setting{
				{SettingKey: test.legacyKey, SettingValue: test.legacyValue, ConfigType: ConfigTypeAI},
				{SettingKey: test.customKey, SettingValue: customPrompt, ConfigType: ConfigTypeAI},
			}).Error; err != nil {
				t.Fatal(err)
			}
			newSystemTestModule(t, db, &storageRuntimeStub{})

			var upgraded Setting
			if err := db.Where("setting_key = ?", test.legacyKey).First(&upgraded).Error; err != nil {
				t.Fatal(err)
			}
			if upgraded.SettingValue != test.upgradedValue {
				t.Fatalf("legacy %s prompt was not upgraded", test.name)
			}
			var custom Setting
			if err := db.Where("setting_key = ?", test.customKey).First(&custom).Error; err != nil {
				t.Fatal(err)
			}
			if custom.SettingValue != customPrompt {
				t.Fatalf("custom prompt was overwritten while upgrading %s", test.name)
			}
		})
	}
}

func TestDefaultAIPromptsKeepRequiredTemplateContracts(t *testing.T) {
	for _, placeholder := range []string{"{{.Article}}", "{{.Tags}}"} {
		if !strings.Contains(DefaultTagsPrompt, placeholder) {
			t.Fatalf("tag prompt is missing placeholder %s", placeholder)
		}
	}
	if !strings.Contains(DefaultTagsPrompt, "JSON") || !strings.Contains(DefaultTagsPrompt, "不得超过 8 个") {
		t.Fatal("tag prompt is missing structured output constraints")
	}
	if !strings.Contains(DefaultAbstractPrompt, "{{.ArticleContent}}") || !strings.Contains(DefaultAbstractPrompt, "本文讲述了") {
		t.Fatal("abstract prompt is missing its compatibility contract")
	}
}

func TestAIConfigSourceReadsUpdatesImmediately(t *testing.T) {
	db := openSystemTestDB(t)
	module := newSystemTestModule(t, db, &storageRuntimeStub{})
	source := module.AIConfigSource()
	_, _, model, _, err := source.LoadAITaggingConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if model != "gpt-4.1-mini" {
		t.Fatalf("model=%q", model)
	}
	if err := module.service.settings.update(context.Background(), SettingKeyAIModel, "new-model", ConfigTypeAI); err != nil {
		t.Fatal(err)
	}
	_, _, model, _, err = source.LoadAITaggingConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if model != "new-model" {
		t.Fatalf("updated model=%q", model)
	}
	if err := db.Where("setting_key = ?", SettingKeyAIPromptGetTags).Delete(&Setting{}).Error; err != nil {
		t.Fatal(err)
	}
	_, _, _, prompt, err := source.LoadAITaggingConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if prompt != DefaultTagsPrompt {
		t.Fatal("missing AI prompt did not fall back to the compatible default")
	}
}

func TestUpdateConfigsMergesPartialStoragePayload(t *testing.T) {
	runtime := &storageRuntimeStub{path: t.TempDir(), chunkSize: 5120}
	module := newSystemTestModule(t, openSystemTestDB(t), runtime)
	runtime.applyCalls = 0

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/admin/config", bytes.NewBufferString(`{"webdav_chunk_size":2048}`))
	context.Request.Header.Set("Content-Type", "application/json")
	module.handler.updateConfigs(context)
	if response.Code != http.StatusOK {
		t.Fatalf("partial update status=%d body=%s", response.Code, response.Body.String())
	}

	config, err := module.service.config(context.Request.Context())
	if err != nil {
		t.Fatal(err)
	}
	if config.FileStoragePath != runtime.path || config.WebDAVChunkSize != 2048 {
		t.Fatalf("partial update overwrote stored values: %+v runtime=%+v", config, runtime)
	}
	if runtime.applyCalls != 1 || runtime.chunkSize != 2048 {
		t.Fatalf("runtime did not receive merged storage config: %+v", runtime)
	}
}

func TestUpdateConfigsRejectsNonexistentStoragePathBeforePersisting(t *testing.T) {
	runtime := &storageRuntimeStub{path: t.TempDir(), chunkSize: 5120}
	module := newSystemTestModule(t, openSystemTestDB(t), runtime)
	runtime.applyCalls = 0
	missingPath := filepath.Join(t.TempDir(), "missing")

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/admin/config", bytes.NewBufferString(`{"file_storage_path":"`+missingPath+`"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	module.handler.updateConfigs(context)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid storage path status=%d body=%s", response.Code, response.Body.String())
	}
	config, err := module.service.config(context.Request.Context())
	if err != nil {
		t.Fatal(err)
	}
	if config.FileStoragePath != runtime.path || runtime.applyCalls != 0 {
		t.Fatalf("invalid path changed state: config=%+v runtime=%+v", config, runtime)
	}
}

func TestStorageSettingsRollbackWhenRuntimeApplyFails(t *testing.T) {
	runtime := &storageRuntimeStub{path: t.TempDir()}
	module := newSystemTestModule(t, openSystemTestDB(t), runtime)
	oldPath := t.TempDir()
	if err := module.service.settings.updateBatch(context.Background(), map[string]string{SettingKeyFileStoragePath: oldPath, SettingKeyWebDAVChunkSize: "1024"}, ConfigTypeStorage); err != nil {
		t.Fatal(err)
	}
	runtime.err = errors.New("runtime failed")
	err := module.handler.applyStorage(context.Background(), StorageConfig{FileStoragePath: t.TempDir(), WebDAVChunkSize: 2048})
	if err == nil {
		t.Fatal("expected apply failure")
	}
	config, err := module.service.configByType(context.Background(), ConfigTypeStorage)
	if err != nil {
		t.Fatal(err)
	}
	if config.FileStoragePath != oldPath || config.WebDAVChunkSize != 1024 {
		t.Fatalf("config not rolled back: %+v", config)
	}
}

func TestBackupDirectoriesRejectPathTraversal(t *testing.T) {
	runtime := &storageRuntimeStub{path: t.TempDir(), chunkSize: 5120}
	module := newSystemTestModule(t, openSystemTestDB(t), runtime)
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	err := module.handler.addBackupDirectories(writer, "selected", "../../outside")
	_ = writer.Close()
	if err == nil {
		t.Fatal("expected path traversal directory to be rejected")
	}
}

func TestModuleRegistersExistingSystemRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	module := newSystemTestModule(t, openSystemTestDB(t), &storageRuntimeStub{})
	engine := gin.New()
	routes := &router.Routes{Engine: engine, PublicAPI: engine.Group("/api"), AdminAPI: engine.Group("/api/admin")}
	module.RegisterRoutes(routes)
	want := map[string]bool{"GET /api/admin/config": false, "PUT /api/admin/config/storage": false, "GET /api/admin/config/backup": false, "GET /api/admin/config/storage-path": false, "GET /api/admin/system-setting/list": false, "DELETE /api/admin/system-setting/:id": false, "PUT /api/files/storage-path": false}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route not registered: %s", route)
		}
	}
}
