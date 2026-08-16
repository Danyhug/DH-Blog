package files

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

// 上传会话参数上限。分片大小设上限防止单次请求构造超大切片，
// 会话目录过期后由 temp janitor 清理（见 service.go）。
const (
	maxChunkSizeBytes    = 64 * 1024 * 1024 // 单个分片最大 64MB
	maxUploadChunkGroups = 1000000          // totalChunks 上限，防止 info.txt 被构造出天文数字
)

// chunkSessionInfo 是 temp 目录中 info.txt 的解析结果。
// hasUserID 区分新旧格式：旧会话没有 userId 行，无法归属，一律视为失效。
type chunkSessionInfo struct {
	fileName    string
	fileSize    int64
	totalChunks int
	chunkSize   int64
	parentID    string
	userID      uint64
	hasUserID   bool
}

func (info *chunkSessionInfo) belongsTo(userID uint64) bool {
	return info.hasUserID && info.userID == userID
}

// validateUploadID 拒绝任何可能改变临时目录位置的会话 ID。
func validateUploadID(uploadID string) error {
	if uploadID == "" || uploadID == "." || uploadID == ".." {
		return fmt.Errorf("uploadId不能为空或为非法路径")
	}
	if strings.ContainsAny(uploadID, `/\`) || filepath.Base(uploadID) != uploadID {
		return fmt.Errorf("uploadId包含非法字符")
	}
	return nil
}

// validateFileName 拒绝路径分隔符、相对路径元素与控制字符。
// 控制字符（换行等）在 Linux/macOS 上合法但会破坏 info.txt 的行式格式，
// 文件名只允许作为单个路径段使用。
func validateFileName(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("文件名不能为空或为非法路径")
	}
	if strings.ContainsAny(name, `/\`) || filepath.Base(name) != name {
		return fmt.Errorf("文件名包含非法字符")
	}
	if strings.ContainsAny(name, "\r\n\t\x00") {
		return fmt.Errorf("文件名包含控制字符")
	}
	return nil
}

// readChunkSessionInfo 解析 temp 目录中的 info.txt。文件不存在时返回错误。
func readChunkSessionInfo(tempDir string) (*chunkSessionInfo, error) {
	data, err := os.ReadFile(filepath.Join(tempDir, "info.txt"))
	if err != nil {
		return nil, err
	}

	info := &chunkSessionInfo{}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.TrimSpace(parts[1])
		switch parts[0] {
		case "fileName":
			info.fileName = value
		case "fileSize":
			info.fileSize, _ = strconv.ParseInt(value, 10, 64)
		case "totalChunks":
			info.totalChunks, _ = strconv.Atoi(value)
		case "chunkSize":
			info.chunkSize, _ = strconv.ParseInt(value, 10, 64)
		case "parentId":
			info.parentID = value
		case "userId":
			info.userID, _ = strconv.ParseUint(value, 10, 64)
			info.hasUserID = true
		}
	}
	return info, nil
}

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

	userID := h.getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, response.Error("未授权"))
		return
	}

	fileName := strings.TrimSpace(req.FileName)
	if err := validateFileName(fileName); err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}
	if req.FileSize <= 0 {
		c.JSON(http.StatusOK, response.Error("文件名和文件大小不能为空"))
		return
	}

	chunkSize := req.ChunkSize
	if chunkSize == 0 {
		chunkSize = int64(h.fileService.ChunkSizeKB() * 1024)
	}
	if chunkSize <= 0 || chunkSize > maxChunkSizeBytes {
		c.JSON(http.StatusOK, response.Error("分片大小非法"))
		return
	}

	uploadId := strings.TrimSpace(req.UploadId)
	if uploadId == "" {
		uploadId = fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), fileName)
	} else if err := validateUploadID(uploadId); err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}

	totalChunks := int((req.FileSize + chunkSize - 1) / chunkSize)
	if totalChunks <= 0 || totalChunks > maxUploadChunkGroups {
		c.JSON(http.StatusOK, response.Error("文件大小与分片大小不匹配"))
		return
	}

	baseDir := h.fileService.GetStoragePath()
	tempDir := filepath.Join(baseDir, tempDirName, uploadId)

	// 会话已存在时按三种情况处理：他人会话拒绝、参数一致直接复用（幂等）、
	// 参数不一致则丢弃旧分片重建，否则新旧分片边界错位会拼出损坏文件。
	if existing, err := readChunkSessionInfo(tempDir); err == nil {
		if existing.hasUserID && existing.userID != userID {
			c.JSON(http.StatusOK, response.Error("上传会话已被其他用户占用"))
			return
		}
		sameParams := existing.fileName == fileName &&
			existing.fileSize == req.FileSize &&
			existing.chunkSize == chunkSize
		if sameParams && existing.hasUserID {
			c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
				"uploadId":    uploadId,
				"chunkSize":   chunkSize,
				"totalChunks": totalChunks,
				"fileName":    fileName,
				"fileSize":    req.FileSize,
				"parentId":    strings.TrimSpace(req.ParentId),
			}))
			return
		}
		if err := os.RemoveAll(tempDir); err != nil {
			c.JSON(http.StatusOK, response.Error("清理旧上传会话失败"))
			return
		}
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusOK, response.Error("创建临时目录失败"))
		return
	}

	infoContent := fmt.Sprintf("fileName=%s\nfileSize=%d\ntotalChunks=%d\nchunkSize=%d\nparentId=%s\nuserId=%d",
		fileName, req.FileSize, totalChunks, chunkSize, strings.TrimSpace(req.ParentId), userID)
	if err := os.WriteFile(filepath.Join(tempDir, "info.txt"), []byte(infoContent), 0644); err != nil {
		c.JSON(http.StatusOK, response.Error("保存上传信息失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"uploadId":    uploadId,
		"chunkSize":   chunkSize,
		"totalChunks": totalChunks,
		"fileName":    fileName,
		"fileSize":    req.FileSize,
		"parentId":    strings.TrimSpace(req.ParentId),
	}))
}

// GetUploadedChunks 获取已上传分片列表
// @Summary 获取已上传分片列表
// @Description 获取指定上传会话已上传的分片索引列表
// @Tags 文件上传
// @Produce json
// @Param uploadId path string true "上传会话ID"
// @Success http.StatusOK {object} map[string]interface{} "{"chunks": [0,1,2], "totalChunks": 10, "chunkSize": 5242880}"
// @Failure http.StatusOK {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/{uploadId}/chunks [get]
func (h *chunkUploadHandler) GetUploadedChunks(c *gin.Context) {
	uploadId := c.Param("uploadId")
	if uploadId == "" {
		c.JSON(http.StatusOK, response.Error("uploadId不能为空"))
		return
	}
	if err := validateUploadID(uploadId); err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}

	baseDir := h.fileService.GetStoragePath()
	tempDir := filepath.Join(baseDir, tempDirName, uploadId)

	// 会话不存在时返回空列表（首次上传时的预期行为）；
	// 无法归属的旧格式会话同样返回空列表，前端会重新 init 并触发重建。
	info, err := readChunkSessionInfo(tempDir)
	if err != nil {
		c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
			"chunks":      []int{},
			"totalChunks": 0,
			"uploadId":    uploadId,
		}))
		return
	}
	if info.hasUserID && !info.belongsTo(h.getUserID(c)) {
		c.JSON(http.StatusForbidden, response.Error("无权访问此上传会话"))
		return
	}
	if !info.hasUserID {
		c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
			"chunks":      []int{},
			"totalChunks": 0,
			"uploadId":    uploadId,
		}))
		return
	}

	files, err := filepath.Glob(filepath.Join(tempDir, "chunk_*"))
	if err != nil {
		c.JSON(http.StatusOK, response.Error("读取分片列表失败"))
		return
	}

	// 只把大小与会话参数一致的分片算作已上传：历史上传中断留下的
	// 截断分片会被剔除，客户端会重传而不是永远跳过，避免续传死循环。
	lastChunkExpected := info.fileSize - info.chunkSize*(int64(info.totalChunks)-1)
	var chunks []int
	for _, file := range files {
		filename := filepath.Base(file)
		var index int
		if _, err := fmt.Sscanf(filename, "chunk_%d", &index); err != nil {
			continue
		}
		if index < 0 || index >= info.totalChunks {
			continue
		}
		expected := info.chunkSize
		if index == info.totalChunks-1 {
			expected = lastChunkExpected
		}
		stat, err := os.Stat(file)
		if err != nil || stat.Size() != expected {
			continue
		}
		chunks = append(chunks, index)
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"chunks":      chunks,
		"totalChunks": info.totalChunks,
		"chunkSize":   info.chunkSize,
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
	if err := validateUploadID(uploadId); err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}

	tempDir := filepath.Join(h.fileService.GetStoragePath(), tempDirName, uploadId)
	// 取消是清理操作，会话已经不在时按成功处理，方便前端无条件调用
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
			"success":  true,
			"uploadId": uploadId,
		}))
		return
	}

	// 有归属的会话只允许创建者取消；旧格式会话无主可清，任何登录用户都能删除。
	if info, err := readChunkSessionInfo(tempDir); err == nil && info.hasUserID && !info.belongsTo(h.getUserID(c)) {
		c.JSON(http.StatusForbidden, response.Error("无权操作此上传会话"))
		return
	}

	if err := os.RemoveAll(tempDir); err != nil {
		c.JSON(http.StatusOK, response.Error("清理临时文件失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"success":  true,
		"uploadId": uploadId,
	}))
}
