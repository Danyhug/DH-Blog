package files

import (
	"crypto/sha256"
	"encoding/hex"
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

// sha256FileThreshold 超过该大小的文件在合并时计算 SHA256 并存入 FileHash。
const sha256FileThreshold = 100 * 1024 * 1024

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
	if err := validateUploadID(req.UploadId); err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}

	userID := h.getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, response.Error("未授权"))
		return
	}

	baseDir := h.fileService.GetStoragePath()
	tempDir := filepath.Join(baseDir, tempDirName, req.UploadId)
	info, err := readChunkSessionInfo(tempDir)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("上传会话不存在"))
		return
	}
	if !info.hasUserID {
		c.JSON(http.StatusOK, response.Error("上传会话已失效，请重新初始化"))
		return
	}
	if !info.belongsTo(userID) {
		c.JSON(http.StatusForbidden, response.Error("无权操作此上传会话"))
		return
	}

	fileName := strings.TrimSpace(info.fileName)
	if err := validateFileName(fileName); err != nil {
		c.JSON(http.StatusOK, response.Error("会话中的文件名非法"))
		return
	}
	if info.fileSize <= 0 || info.totalChunks <= 0 || info.chunkSize <= 0 {
		c.JSON(http.StatusOK, response.Error("上传会话参数非法"))
		return
	}

	parentId := strings.TrimSpace(info.parentID)
	var parentStoragePath string
	if parentId != "" {
		parentNumeric, err := strconv.Atoi(parentId)
		if err != nil {
			c.JSON(http.StatusOK, response.Error("父目录ID无效"))
			return
		}

		parent, err := h.fileService.findFolderByID(c.Request.Context(), parentNumeric)
		if err != nil {
			c.JSON(http.StatusOK, response.Error("父目录不存在"))
			return
		}
		// 目录只能归属上传者自己，防止把文件塞进别人的文件夹。
		if parent.UserID != userID {
			c.JSON(http.StatusForbidden, response.Error("无权操作此父目录"))
			return
		}
		parentStoragePath = parent.StoragePath
	}

	// 逐索引校验分片完整性与大小：只数数量会放过"缺一块多一块"的组合。
	lastChunkExpected := info.fileSize - info.chunkSize*(int64(info.totalChunks)-1)
	for i := 0; i < info.totalChunks; i++ {
		expected := info.chunkSize
		if i == info.totalChunks-1 {
			expected = lastChunkExpected
		}
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
		stat, err := os.Stat(chunkPath)
		if err != nil {
			c.JSON(http.StatusOK, response.Error(fmt.Sprintf("分片 %d 缺失", i)))
			return
		}
		if stat.Size() > expected {
			c.JSON(http.StatusOK, response.Error(fmt.Sprintf("分片 %d 大小非法", i)))
			return
		}
		if i != info.totalChunks-1 && stat.Size() != expected {
			c.JSON(http.StatusOK, response.Error(fmt.Sprintf("分片 %d 大小非法", i)))
			return
		}
	}

	// 与 CreateFolder/UploadFile 保持一致：同名直接报错，不静默改名。
	conflict, err := h.fileService.hasNameConflict(c.Request.Context(), userID, parentId, fileName)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("检查同名文件失败"))
		return
	}
	if conflict {
		c.JSON(http.StatusOK, response.Error("同名文件已存在"))
		return
	}

	relativeDir := sanitizeRelativePath(parentStoragePath)
	storageDir := filepath.Join(baseDir, relativeDir)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		c.JSON(http.StatusOK, response.Error("创建存储目录失败"))
		return
	}

	finalPath := filepath.Join(storageDir, fileName)

	// O_EXCL 原子创建：hasNameConflict 只查了索引，WebDAV 刚写入、防抖同步
	// 尚未落库的文件不在索引里，直接 os.Create 会把它们静默截断。
	finalFile, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			c.JSON(http.StatusOK, response.Error("同名文件已存在"))
			return
		}
		logrus.Error("创建最终文件失败: ", err)
		c.JSON(http.StatusOK, response.Error("创建最终文件失败"))
		return
	}
	defer func() { _ = finalFile.Close() }()

	var hasher hash.Hash
	if info.fileSize > sha256FileThreshold {
		hasher = sha256.New()
	}

	buffer := make([]byte, 1024*1024)
	totalSize, err := mergeChunks(tempDir, info.totalChunks, finalFile, buffer, hasher)
	if err != nil {
		_ = os.Remove(finalPath)
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}

	if err := finalFile.Sync(); err != nil {
		logrus.Warnf("同步文件到磁盘失败: %v", err)
	}

	if totalSize != info.fileSize {
		_ = os.Remove(finalPath)
		c.JSON(http.StatusOK, response.Error(fmt.Sprintf("文件大小不匹配：期望 %d，实际 %d", info.fileSize, totalSize)))
		return
	}

	var fileHash string
	if hasher != nil {
		fileHash = hex.EncodeToString(hasher.Sum(nil))
	}

	// 异步清理临时目录（避免阻塞响应）
	go func() {
		time.Sleep(5 * time.Second) // 延迟清理，确保客户端已收到响应
		if err := os.RemoveAll(tempDir); err != nil {
			logrus.Warnf("清理上传临时目录失败: %v", err)
		}
	}()

	file := &File{
		UserID:      userID,
		ParentID:    parentId,
		Name:        fileName,
		IsFolder:    false,
		Size:        totalSize,
		StoragePath: filepath.Join(relativeDir, fileName),
		MimeType:    getMimeType(fileName),
		FileHash:    fileHash,
	}

	if err := h.fileService.createFileRecord(c.Request.Context(), file); err != nil {
		_ = os.Remove(finalPath)
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
