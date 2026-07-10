package files

import "gorm.io/gorm"

// ChunkUploadController 分片上传控制器
type chunkUploadHandler struct {
	fileService *fileService
	db          *gorm.DB
}

// newChunkUploadHandler 创建分片上传 HTTP handler。
func newChunkUploadHandler(fileService *fileService, db *gorm.DB) *chunkUploadHandler {
	return &chunkUploadHandler{
		fileService: fileService,
		db:          db,
	}
}
