package files

// ChunkUploadController 分片上传控制器
type chunkUploadHandler struct {
	fileService *fileService
}

// newChunkUploadHandler 创建分片上传 HTTP handler。
// 持久化操作通过 fileService 完成，handler 不直接接触数据库。
func newChunkUploadHandler(fileService *fileService) *chunkUploadHandler {
	return &chunkUploadHandler{
		fileService: fileService,
	}
}
