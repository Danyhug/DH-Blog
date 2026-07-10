package files

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Dependencies are the application-owned collaborators needed by the files module.
type Dependencies struct {
	DB                 *gorm.DB
	StaticFilesPath    string
	InitialStoragePath string
	InitialChunkSizeKB int
}

// Module owns file persistence, business logic, HTTP handlers, and routes.
type Module struct {
	repository         fileRepository
	service            *fileService
	handler            *handler
	chunkUploadHandler *chunkUploadHandler
	staticFilesPath    string
}

// New assembles the complete files vertical module.
func New(deps Dependencies) *Module {
	repository := newRepository(deps.DB)
	service := newService(repository, deps.InitialStoragePath, deps.InitialChunkSizeKB)
	return &Module{
		repository:         repository,
		service:            service,
		handler:            newHandler(service),
		chunkUploadHandler: newChunkUploadHandler(service, deps.DB),
		staticFilesPath:    deps.StaticFilesPath,
	}
}

// StorageRuntime exposes the narrow runtime storage port consumed by the system module.
type StorageRuntime interface {
	InitializeStorageConfig(ctx context.Context, path string, chunkSizeKB int) error
	ApplyStorageConfig(ctx context.Context, path string, chunkSizeKB int) error
	GetStoragePath() string
	ProtectedDirectoryNames() []string
}

// StorageRuntime returns a settings-agnostic adapter for runtime storage changes.
func (m *Module) StorageRuntime() StorageRuntime { return m.service }

// Service exposes file operations to other feature modules.
func (m *Module) Service() Service {
	return m.service
}

// MigrationModels declares the database table owned by this module.
func MigrationModels() []any {
	return []any{&File{}}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	fileAPI := routes.AuthenticatedAPI("/api/files")
	fileAPI.GET("/list", m.handler.ListFiles)
	fileAPI.POST("/folder", m.handler.CreateFolder)
	fileAPI.GET("/download/:id", m.handler.DownloadFile)
	fileAPI.PUT("/rename/:id", m.handler.RenameFile)
	fileAPI.DELETE("/:id", m.handler.DeleteFile)
	fileAPI.GET("/directory-tree", m.handler.GetDirectoryTree)
	fileAPI.POST("/sync", m.handler.SyncFiles)

	chunkAPI := fileAPI.Group("/upload/chunk")
	chunkAPI.POST("/init", m.chunkUploadHandler.InitChunkUpload)
	chunkAPI.POST("/chunk", m.chunkUploadHandler.UploadChunk)
	chunkAPI.POST("/complete", m.chunkUploadHandler.CompleteChunkUpload)
	chunkAPI.GET("/:uploadId/chunks", m.chunkUploadHandler.GetUploadedChunks)
	chunkAPI.DELETE("/:uploadId", m.chunkUploadHandler.CancelChunkUpload)

	routes.PublicAPI.Static("/uploads", m.staticFilesPath)
	logrus.Infof("静态文件服务路径: /uploads -> %s", m.staticFilesPath)

	routes.PublicAPI.GET("/博客/*filepath", func(c *gin.Context) {
		storagePath := m.service.GetStoragePath()
		if storagePath == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		filePath := strings.TrimPrefix(c.Param("filepath"), "/")
		fullPath := filepath.Join(storagePath, "博客", filePath)
		c.File(fullPath)
	})
}
