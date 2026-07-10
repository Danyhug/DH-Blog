package files

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CompleteChunkUpload 完成分片上传
// @Summary 完成分片上传
// @Description 合并所有分片并完成文件上传
// @Tags 文件上传
// @Accept json
// @Produce json
// @Param uploadId body string true "上传会话ID"
// @Success http.StatusOK {object} map[string]interface{} "{"id": 123, "name": "文件名", "size": 1024}"
// @Failure http.StatusOK {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/complete [post]
func (h *chunkUploadHandler) CompleteChunkUpload(c *gin.Context) {
	var req struct {
		UploadId string `json:"uploadId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, response.Error("参数错误"))
		return
	}

	if req.UploadId == "" {
		c.JSON(http.StatusOK, response.Error("uploadId不能为空"))
		return
	}

	userID := h.getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, response.Error("未授权"))
		return
	}

	// 获取配置的存储路径
	baseDir := h.fileService.GetStoragePath()
	tempDir := filepath.Join(baseDir, "temp", req.UploadId)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, response.Error("上传会话不存在"))
		return
	}

	// 读取上传信息文件
	infoFile := filepath.Join(tempDir, "info.txt")
	infoData, err := os.ReadFile(infoFile)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("读取上传信息失败"))
		return
	}

	// 解析上传信息
	var fileName, parentId string
	var fileSize, totalChunks int
	lines := strings.Split(string(infoData), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "fileName":
				fileName = parts[1]
			case "fileSize":
				fmt.Sscanf(parts[1], "%d", &fileSize)
			case "totalChunks":
				fmt.Sscanf(parts[1], "%d", &totalChunks)
			case "parentId":
				parentId = strings.TrimSpace(parts[1])
			}
		}
	}

	parentId = strings.TrimSpace(parentId)

	var parentStoragePath string
	if parentId != "" {
		parentNumeric, err := strconv.Atoi(parentId)
		if err != nil {
			c.JSON(http.StatusOK, response.Error("父目录ID无效"))
			return
		}

		var parent File
		if err := h.db.First(&parent, parentNumeric).Error; err != nil {
			c.JSON(http.StatusOK, response.Error("父目录不存在"))
			return
		}

		if !parent.IsFolder {
			c.JSON(http.StatusOK, response.Error("父目录不是文件夹"))
			return
		}

		parentStoragePath = parent.StoragePath
	}

	// 获取已上传的分片文件
	files, err := filepath.Glob(filepath.Join(tempDir, "chunk_*"))
	if err != nil {
		c.JSON(http.StatusOK, response.Error("读取分片失败"))
		return
	}

	// 检查分片完整性
	if len(files) != totalChunks {
		c.JSON(http.StatusOK, response.Error("分片不完整"))
		return
	}

	// 创建最终存储路径 - 使用配置的存储路径
	relativeDir := sanitizeRelativePath(parentStoragePath)
	if parentId != "" && relativeDir == "" {
		relativeDir = sanitizeRelativePath(parentId)
	}
	storageDir := filepath.Join(baseDir, relativeDir)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		c.JSON(http.StatusOK, response.Error("创建存储目录失败"))
		return
	}

	finalPath := filepath.Join(storageDir, fileName)

	// 检查是否已存在同名文件
	if _, err := os.Stat(finalPath); err == nil {
		ext := filepath.Ext(fileName)
		nameWithoutExt := strings.TrimSuffix(fileName, ext)
		fileName = fmt.Sprintf("%s_%d%s", nameWithoutExt, time.Now().Unix(), ext)
		finalPath = filepath.Join(storageDir, fileName)
	}

	// 合并所有分片 - 优化大文件合并性能
	finalFile, err := os.Create(finalPath)
	if err != nil {
		logrus.Error("创建最终文件失败: ", err)
		c.JSON(http.StatusOK, response.Error("创建最终文件失败"))
		return
	}
	defer finalFile.Close()

	// 根据文件大小选择最优的合并策略
	var hasher hash.Hash

	// 为大文件添加SHA256校验
	if fileSize > 100*1024*1024 { // 100MB以上文件计算哈希
		hasher = sha256.New()
	}

	// 使用缓冲区减少内存占用
	buffer := make([]byte, 64*1024*1024) // 64MB缓冲区，根据现代系统优化

	// 优化的合并策略：根据分片数量选择不同算法
	var totalSize int64
	if totalChunks <= 100 {
		// 小文件：顺序合并，减少复杂度
		totalSize, err = h.mergeChunksSequential(tempDir, totalChunks, finalFile, buffer, hasher)
	} else if totalChunks <= 1000 {
		// 中等文件：带缓冲的顺序合并
		totalSize, err = h.mergeChunksBuffered(tempDir, totalChunks, finalFile, buffer, hasher)
	} else {
		// 大文件：并发合并（8GB文件可能有8000+分片）
		totalSize, err = h.mergeChunksConcurrent(tempDir, totalChunks, finalFile, buffer, hasher)
	}

	if err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}

	// 确保所有数据写入磁盘
	finalFile.Sync()

	// 验证文件完整性
	if totalSize != int64(fileSize) {
		c.JSON(http.StatusOK, response.Error(fmt.Sprintf("文件大小不匹配：期望 %d，实际 %d", fileSize, totalSize)))
		return
	}

	// 验证SHA256（如果计算了）
	if hasher != nil {
		expectedHash := hasher.Sum(nil)
		_ = expectedHash // 可以存储到数据库用于后续验证
	}

	// 异步清理临时目录（避免阻塞响应）
	go func() {
		time.Sleep(5 * time.Second) // 延迟清理，确保客户端已收到响应
		os.RemoveAll(tempDir)
	}()

	// 创建文件数据库记录
	file := &File{
		UserID:      uint64(userID),
		ParentID:    parentId,
		Name:        fileName,
		IsFolder:    false,
		Size:        totalSize,
		StoragePath: filepath.Join(relativeDir, fileName),
		MimeType:    h.getMimeType(fileName),
	}

	// 保存到数据库
	if err := h.db.Create(file).Error; err != nil {
		// 删除已创建的文件
		os.Remove(finalPath)
		c.JSON(http.StatusOK, response.Error("保存文件记录失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"id":   file.ID,
		"name": file.Name,
		"size": file.Size,
	}))
}

// getUserID 从上下文中获取用户ID
func (h *chunkUploadHandler) getUserID(c *gin.Context) uint64 {
	if userID, exists := c.Get("userID"); exists {
		if id, ok := userID.(float64); ok {
			return uint64(id)
		}
		if id, ok := userID.(uint64); ok {
			return id
		}
	}
	return 0
}

func sanitizeRelativePath(input string) string {
	input = strings.TrimSpace(input)
	cleaned := filepath.Clean(input)
	if input == "" || cleaned == "." {
		return ""
	}
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		logrus.Warnf("检测到非法父目录路径: %s，已重置为根目录", input)
		return ""
	}
	return cleaned
}

// getMimeType 获取文件MIME类型
func (h *chunkUploadHandler) getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	default:
		return "application/octet-stream"
	}
}
