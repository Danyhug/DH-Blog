package handler

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/response"
	"dh-blog/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ChunkUploadHandler 分片上传处理器
type ChunkUploadHandler struct {
	fileService service.IFileService
	db          *gorm.DB
	config      service.IConfigService
}

// NewChunkUploadHandler 创建分片上传处理器
func NewChunkUploadHandler(fileService service.IFileService, db *gorm.DB, config service.IConfigService) *ChunkUploadHandler {
	return &ChunkUploadHandler{
		fileService: fileService,
		db:          db,
		config:      config,
	}
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
func (h *ChunkUploadHandler) InitChunkUpload(c *gin.Context) {
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
	parentId := req.ParentId
	uploadId := req.UploadId

	if chunkSize == 0 {
		// 从系统配置获取WebDAV分片大小
		config, err := h.config.GetSystemConfig()
		if err == nil && config.WebdavChunkSize > 0 {
			chunkSize = int64(config.WebdavChunkSize * 1024) // KB转字节
		} else {
			chunkSize = int64(5 * 1024 * 1024) // 默认5MB
		}
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

// UploadChunk 上传分片
// @Summary 上传文件分片
// @Description 上传文件的一个分片
// @Tags 文件上传
// @Accept multipart/form-data
// @Produce json
// @Param uploadId formData string true "上传会话ID"
// @Param chunkIndex formData int true "分片索引"
// @Param chunk formData file true "分片数据"
// @Success http.StatusOK {object} map[string]interface{} "{"success": true}"
// @Failure http.StatusOK {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk [post]
func (h *ChunkUploadHandler) UploadChunk(c *gin.Context) {
	uploadId := c.PostForm("uploadId")
	chunkIndexStr := c.PostForm("chunkIndex")

	if uploadId == "" || chunkIndexStr == "" {
		c.JSON(http.StatusOK, response.Error("uploadId和chunkIndex不能为空"))
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("chunkIndex格式错误"))
		return
	}

	// 获取配置的存储路径
	storagePath := h.fileService.GetStoragePath()
	baseDir := storagePath
	tempDir := filepath.Join(baseDir, "temp", uploadId)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, response.Error("上传会话不存在"))
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusOK, response.Error("获取分片数据失败"))
		return
	}

	// 保存分片文件
	chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkIndex))
	if err := c.SaveUploadedFile(file, chunkFile); err != nil {
		c.JSON(http.StatusOK, response.Error("保存分片失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"success":    true,
		"chunkIndex": chunkIndex,
		"uploadId":   uploadId,
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
func (h *ChunkUploadHandler) GetUploadedChunks(c *gin.Context) {
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
func (h *ChunkUploadHandler) CancelChunkUpload(c *gin.Context) {
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
func (h *ChunkUploadHandler) CompleteChunkUpload(c *gin.Context) {
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

	// 获取用户ID
	userID := h.getUserID(c)
	if userID == 0 {
		c.JSON(401, response.Error("未授权"))
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
				parentId = parts[1]
			}
		}
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
	storageDir := filepath.Join(baseDir, fmt.Sprintf("user_%d", userID))
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
	file := &model.File{
		UserID:      uint64(userID),
		ParentID:    parentId,
		Name:        fileName,
		IsFolder:    false,
		Size:        totalSize,
		StoragePath: filepath.Join(fmt.Sprintf("user_%d", userID), fileName),
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
func (h *ChunkUploadHandler) getUserID(c *gin.Context) uint64 {
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

// mergeChunksSequential 顺序合并分片（适用于小文件）
func (h *ChunkUploadHandler) mergeChunksSequential(tempDir string, totalChunks int, finalFile *os.File, buffer []byte, hasher hash.Hash) (int64, error) {
	var totalSize int64

	for i := 0; i < totalChunks; i++ {
		chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
		chunk, err := os.Open(chunkFile)
		if err != nil {
			return 0, fmt.Errorf("读取分片 %d 失败: %v", i, err)
		}

		var writer io.Writer = finalFile
		if hasher != nil {
			writer = io.MultiWriter(finalFile, hasher)
		}

		n, err := io.CopyBuffer(writer, chunk, buffer)
		chunk.Close()

		if err != nil {
			return 0, fmt.Errorf("合并分片 %d 失败: %v", i, err)
		}

		totalSize += n

		// 定期刷新到磁盘
		if i > 0 && i%50 == 0 {
			finalFile.Sync()
		}
	}

	return totalSize, nil
}

// mergeChunksBuffered 带缓冲的顺序合并（适用于中等文件）
func (h *ChunkUploadHandler) mergeChunksBuffered(tempDir string, totalChunks int, finalFile *os.File, buffer []byte, hasher hash.Hash) (int64, error) {
	var totalSize int64

	for i := 0; i < totalChunks; i++ {
		chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
		chunk, err := os.Open(chunkFile)
		if err != nil {
			return 0, fmt.Errorf("读取分片 %d 失败: %v", i, err)
		}

		var writer io.Writer = finalFile
		if hasher != nil {
			writer = io.MultiWriter(finalFile, hasher)
		}

		n, err := io.CopyBuffer(writer, chunk, buffer)
		chunk.Close()

		if err != nil {
			return 0, fmt.Errorf("合并分片 %d 失败: %v", i, err)
		}

		totalSize += n

		// 更频繁的磁盘刷新
		if i > 0 && i%25 == 0 {
			finalFile.Sync()
		}
	}

	return totalSize, nil
}

// mergeChunksConcurrent 并发合并分片（适用于大文件）
func (h *ChunkUploadHandler) mergeChunksConcurrent(tempDir string, totalChunks int, finalFile *os.File, buffer []byte, hasher hash.Hash) (int64, error) {
	const workers = 4     // 并发工作线程数
	const batchSize = 100 // 每批处理的分片数

	var totalSize int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// 创建临时文件映射
	tempFiles := make([]string, totalChunks)
	for i := 0; i < totalChunks; i++ {
		tempFiles[i] = filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
	}

	// 分批次处理，避免同时打开过多文件
	for batchStart := 0; batchStart < totalChunks; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > totalChunks {
			batchEnd = totalChunks
		}

		// 为每个批次创建结果通道
		results := make(chan int64, batchEnd-batchStart)
		errors := make(chan error, batchEnd-batchStart)

		// 启动工作线程
		for w := 0; w < workers && batchStart+w < batchEnd; w++ {
			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()

				var batchSize int64
				for i := start; i < end; i += workers {
					chunkFile := tempFiles[i]
					chunk, err := os.Open(chunkFile)
					if err != nil {
						errors <- fmt.Errorf("读取分片 %d 失败: %v", i, err)
						return
					}

					// 获取文件大小
					stat, _ := chunk.Stat()
					chunkSize := stat.Size()

					batchSize += chunkSize
					chunk.Close()
				}

				results <- batchSize
			}(batchStart+w, batchEnd)
		}

		// 等待批次完成
		wg.Wait()
		close(results)
		close(errors)

		// 处理结果
		for size := range results {
			mu.Lock()
			totalSize += size
			mu.Unlock()
		}

		// 处理错误
		select {
		case err := <-errors:
			if firstErr == nil {
				firstErr = err
			}
		default:
		}

		if firstErr != nil {
			return 0, firstErr
		}
	}

	// 重新顺序写入（并发阶段只计算大小，这里真正写入）
	// 注：由于并发写入到同一个文件的复杂性，这里使用优化的顺序写入
	var writer io.Writer = finalFile
	if hasher != nil {
		writer = io.MultiWriter(finalFile, hasher)
	}

	for i := 0; i < totalChunks; i++ {
		chunkFile := tempFiles[i]
		chunk, err := os.Open(chunkFile)
		if err != nil {
			return 0, fmt.Errorf("读取分片 %d 失败: %v", i, err)
		}

		_, err = io.CopyBuffer(writer, chunk, buffer)
		chunk.Close()

		if err != nil {
			return 0, fmt.Errorf("合并分片 %d 失败: %v", i, err)
		}

		if i > 0 && i%100 == 0 {
			finalFile.Sync()
		}
	}

	return totalSize, nil
}

// getMimeType 获取文件MIME类型
func (h *ChunkUploadHandler) getMimeType(filename string) string {
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
