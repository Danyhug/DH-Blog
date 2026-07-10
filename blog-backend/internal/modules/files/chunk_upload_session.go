package files

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

// InitChunkUpload 初始化分片上传
// @Summary 初始化分片上传
// @Description 创建一个新的分片上传会话
// @Tags 文件上传
// @Accept json
// @Produce json
// @Param parentId formData string false "父目录ID"
// @Param fileName formData string true "文件名"
// @Param fileSize formData int true "文件大小"
// @Param chunkSize formData int false "分片大小，默认5MB"
// @Param uploadId formData string false "指定上传会话ID（用于断点续传）"
// @Success http.StatusOK {object} map[string]interface{} "{"uploadId": "上传会话ID"}"
// @Failure http.StatusOK {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/init [post]
func (h *chunkUploadHandler) InitChunkUpload(c *gin.Context) {
	var req struct {
		FileName  string `json:"fileName"`
		FileSize  int64  `json:"fileSize"`
		ChunkSize int64  `json:"chunkSize"`
		ParentId  string `json:"parentId"`
		UploadId  string `json:"uploadId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, response.Error("参数错误"))
		return
	}

	if req.FileName == "" || req.FileSize == 0 {
		c.JSON(http.StatusOK, response.Error("文件名和文件大小不能为空"))
		return
	}

	fileName := req.FileName
	fileSize := req.FileSize
	chunkSize := req.ChunkSize
	parentId := strings.TrimSpace(req.ParentId)
	uploadId := req.UploadId

	if chunkSize == 0 {
		chunkSize = int64(h.fileService.ChunkSizeKB() * 1024)
	}

	// 如果没有指定uploadId，则生成新的
	if uploadId == "" {
		uploadId = fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), fileName)
	}
	totalChunks := (fileSize + chunkSize - 1) / chunkSize

	// 获取配置的存储路径
	storagePath := h.fileService.GetStoragePath()
	baseDir := storagePath
	tempDir := filepath.Join(baseDir, "temp", uploadId)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusOK, response.Error("创建临时目录失败"))
		return
	}

	// 保存上传信息
	infoFile := filepath.Join(tempDir, "info.txt")
	infoContent := fmt.Sprintf("fileName=%s\nfileSize=%d\ntotalChunks=%d\nchunkSize=%d\nparentId=%s", fileName, fileSize, totalChunks, chunkSize, parentId)
	if err := os.WriteFile(infoFile, []byte(infoContent), 0644); err != nil {
		c.JSON(http.StatusOK, response.Error("保存上传信息失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"uploadId":    uploadId,
		"chunkSize":   chunkSize,
		"totalChunks": totalChunks,
		"fileName":    fileName,
		"fileSize":    fileSize,
		"parentId":    parentId,
	}))
}

// GetUploadedChunks 获取已上传分片列表
// @Summary 获取已上传分片列表
// @Description 获取指定上传会话已上传的分片索引列表
// @Tags 文件上传
// @Produce json
// @Param uploadId path string true "上传会话ID"
// @Success http.StatusOK {object} map[string]interface{} "{"chunks": [0,1,2], "totalChunks": 10}"
// @Failure http.StatusOK {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/{uploadId}/chunks [get]
func (h *chunkUploadHandler) GetUploadedChunks(c *gin.Context) {
	uploadId := c.Param("uploadId")
	if uploadId == "" {
		c.JSON(http.StatusOK, response.Error("uploadId不能为空"))
		return
	}

	// 获取配置的存储路径
	storagePath := h.fileService.GetStoragePath()
	baseDir := storagePath
	tempDir := filepath.Join(baseDir, "temp", uploadId)

	// 如果上传会话不存在，返回空的分片列表（首次上传时的预期行为）
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
			"chunks":      []int{},
			"totalChunks": 0,
			"uploadId":    uploadId,
		}))
		return
	}

	// 获取已上传的分片文件
	pattern := filepath.Join(tempDir, "chunk_*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("读取分片列表失败"))
		return
	}

	// 提取分片索引
	var chunks []int
	for _, file := range files {
		filename := filepath.Base(file)
		var index int
		_, err := fmt.Sscanf(filename, "chunk_%d", &index)
		if err == nil {
			chunks = append(chunks, index)
		}
	}

	// 读取上传信息文件获取总分片数
	infoFile := filepath.Join(tempDir, "info.txt")
	var totalChunks int
	if data, err := os.ReadFile(infoFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && parts[0] == "totalChunks" {
				fmt.Sscanf(parts[1], "%d", &totalChunks)
				break
			}
		}
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"chunks":      chunks,
		"totalChunks": totalChunks,
		"uploadId":    uploadId,
	}))
}

// CancelChunkUpload 取消分片上传
// @Summary 取消分片上传
// @Description 取消并清理分片上传会话
// @Tags 文件上传
// @Produce json
// @Param uploadId path string true "上传会话ID"
// @Success http.StatusOK {object} map[string]interface{} "{"success": true}"
// @Failure http.StatusOK {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/{uploadId} [delete]
func (h *chunkUploadHandler) CancelChunkUpload(c *gin.Context) {
	uploadId := c.Param("uploadId")
	if uploadId == "" {
		c.JSON(http.StatusOK, response.Error("uploadId不能为空"))
		return
	}

	// 获取配置的存储路径
	storagePath := h.fileService.GetStoragePath()
	tempDir := filepath.Join(storagePath, "temp", uploadId)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, response.Error("上传会话不存在"))
		return
	}

	// 清理临时目录
	if err := os.RemoveAll(tempDir); err != nil {
		c.JSON(http.StatusOK, response.Error("清理临时文件失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"success":  true,
		"uploadId": uploadId,
	}))
}
