package modules

import (
	"net/http"
	"path/filepath"
	"strings"

	"dh-blog/internal/controller"
	"dh-blog/internal/router"
	"dh-blog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FilesModule 注册文件管理、分片上传和公开文件访问路由。
type FilesModule struct {
	fileController        *controller.FileController
	chunkUploadController *controller.ChunkUploadController
	staticFilesPath       string
	fileService           service.IFileService
}

func NewFilesModule(
	fileController *controller.FileController,
	chunkUploadController *controller.ChunkUploadController,
	staticFilesPath string,
	fileService service.IFileService,
) *FilesModule {
	return &FilesModule{
		fileController:        fileController,
		chunkUploadController: chunkUploadController,
		staticFilesPath:       staticFilesPath,
		fileService:           fileService,
	}
}

func (m *FilesModule) RegisterRoutes(routes *router.Routes) {
	fileAPI := routes.FileAPI
	fileAPI.GET("/list", m.fileController.ListFiles)
	fileAPI.POST("/folder", m.fileController.CreateFolder)
	fileAPI.GET("/download/:id", m.fileController.DownloadFile)
	fileAPI.PUT("/rename/:id", m.fileController.RenameFile)
	fileAPI.DELETE("/:id", m.fileController.DeleteFile)
	fileAPI.PUT("/storage-path", m.fileController.UpdateStoragePath)
	fileAPI.GET("/directory-tree", m.fileController.GetDirectoryTree)
	fileAPI.POST("/sync", m.fileController.SyncFiles)

	chunkAPI := fileAPI.Group("/upload/chunk")
	chunkAPI.POST("/init", m.chunkUploadController.InitChunkUpload)
	chunkAPI.POST("/chunk", m.chunkUploadController.UploadChunk)
	chunkAPI.POST("/complete", m.chunkUploadController.CompleteChunkUpload)
	chunkAPI.GET("/:uploadId/chunks", m.chunkUploadController.GetUploadedChunks)
	chunkAPI.DELETE("/:uploadId", m.chunkUploadController.CancelChunkUpload)

	routes.PublicAPI.Static("/uploads", m.staticFilesPath)
	logrus.Infof("静态文件服务路径: /uploads -> %s", m.staticFilesPath)

	routes.PublicAPI.GET("/博客/*filepath", func(c *gin.Context) {
		storagePath := m.fileService.GetStoragePath()
		if storagePath == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		filePath := strings.TrimPrefix(c.Param("filepath"), "/")
		fullPath := filepath.Join(storagePath, "博客", filePath)
		c.File(fullPath)
	})
}
