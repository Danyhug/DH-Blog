package system

import (
	"context"
	"fmt"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

type StorageRuntime interface {
	InitializeStorageConfig(ctx context.Context, path string, chunkSizeKB int) error
	ApplyStorageConfig(ctx context.Context, path string, chunkSizeKB int) error
	GetStoragePath() string
	ProtectedDirectoryNames() []string
}

type AIConfigSource interface {
	LoadAITaggingConfig(ctx context.Context) (endpoint, apiKey, model, prompt string, err error)
}

type Dependencies struct {
	DB           *gorm.DB
	Cache        dhcache.Cache
	DataDir      string
	DatabasePath string
	Storage      StorageRuntime
}

type Module struct {
	service *service
	handler *handler
	ai      aiConfigSource
}

func New(deps Dependencies) (*Module, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("system: DB is required")
	}
	if deps.Cache == nil {
		return nil, fmt.Errorf("system: cache is required")
	}
	if deps.Storage == nil {
		return nil, fmt.Errorf("system: storage runtime is required")
	}
	repository := newSettingRepository(deps.DB, deps.Cache)
	if err := repository.ensureDefaults(context.Background()); err != nil {
		return nil, err
	}
	service := newService(repository)
	stored, err := service.configByType(context.Background(), ConfigTypeStorage)
	if err != nil {
		return nil, fmt.Errorf("system: load storage config: %w", err)
	}
	storagePath := stored.FileStoragePath
	if storagePath == "" {
		storagePath = deps.Storage.GetStoragePath()
		if err := repository.update(context.Background(), SettingKeyFileStoragePath, storagePath, ConfigTypeStorage); err != nil {
			return nil, fmt.Errorf("system: persist default storage path: %w", err)
		}
	}
	if err := deps.Storage.InitializeStorageConfig(context.Background(), storagePath, stored.WebDAVChunkSize); err != nil {
		return nil, fmt.Errorf("system: apply storage config: %w", err)
	}
	handler := newHandler(service, deps.Storage, deps.DataDir, deps.DatabasePath)
	return &Module{service: service, handler: handler, ai: aiConfigSource{service: service}}, nil
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	config := routes.AdminAPI.Group("/config")
	config.GET("", m.handler.getConfigs)
	config.PUT("", m.handler.updateConfigs)
	config.GET("/blog", m.handler.getBlogConfig)
	config.PUT("/blog", m.handler.updateBlogConfig)
	config.GET("/email", m.handler.getEmailConfig)
	config.PUT("/email", m.handler.updateEmailConfig)
	config.GET("/ai", m.handler.getAIConfig)
	config.PUT("/ai", m.handler.updateAIConfig)
	config.GET("/ai/prompts", m.handler.getAIPromptTags)
	config.GET("/storage", m.handler.getStorageConfig)
	config.PUT("/storage", m.handler.updateStorageConfig)
	config.GET("/backup/dirs", m.handler.getBackupDirs)
	config.GET("/backup", m.handler.backupData)
	config.GET("/storage-path", m.handler.getStoragePath)
	config.PUT("/storage-path", m.handler.updateStoragePath)

	settings := routes.AdminAPI.Group("/system-setting")
	settings.GET("/list", m.handler.listSettings)
	settings.POST("", m.handler.addSetting)
	settings.PUT("", m.handler.updateSetting)
	settings.DELETE("/:id", m.handler.deleteSetting)

	// Compatibility endpoint kept under the files API, but persistence remains system-owned.
	routes.AuthenticatedAPI("/api/files").PUT("/storage-path", m.handler.updateStoragePath)
}

func (m *Module) AIConfigSource() AIConfigSource { return m.ai }
