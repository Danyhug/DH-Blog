package router

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dh-blog/internal/middleware"
	"dh-blog/internal/model"
	"dh-blog/internal/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupChunkUploadRoutes 注册分片上传相关路由
func SetupChunkUploadRoutes(engine *gin.Engine, db *gorm.DB) {
	uploadGroup := engine.Group("/api/files/upload/chunk")
	uploadGroup.Use(middleware.JWTMiddleware()) // 添加JWT中间件
	{
		// 初始化分片上传
		uploadGroup.POST("/init", initChunkUpload)
		// 上传分片
		uploadGroup.POST("/chunk", uploadChunk)
		// 完成分片上传
		uploadGroup.POST("/complete", completeChunkUpload(db))
		// 获取已上传分片列表
		uploadGroup.GET("/:uploadId/chunks", getUploadedChunks)
		// 取消分片上传
		uploadGroup.DELETE("/:uploadId", cancelChunkUpload)
	}
}

// initChunkUpload 初始化分片上传
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
// @Success 200 {object} map[string]interface{} "{"uploadId": "上传会话ID"}"
// @Failure 400 {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/init [post]
func initChunkUpload(c *gin.Context) {
	var req struct {
		FileName  string `json:"fileName"`
		FileSize  int64  `json:"fileSize"`
		ChunkSize int64  `json:"chunkSize"`
		ParentId  string `json:"parentId"`
		UploadId  string `json:"uploadId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error("参数错误"))
		return
	}

	if req.FileName == "" || req.FileSize == 0 {
		c.JSON(400, response.Error("文件名和文件大小不能为空"))
		return
	}

	fileName := req.FileName
	fileSize := req.FileSize
	chunkSize := req.ChunkSize
	parentId := req.ParentId
	uploadId := req.UploadId

	if chunkSize == 0 {
		chunkSize = int64(5 * 1024 * 1024) // 默认5MB
	}

	// 如果没有指定uploadId，则生成新的
	if uploadId == "" {
		uploadId = fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), fileName)
	}
	totalChunks := (fileSize + chunkSize - 1) / chunkSize

	// 创建临时目录 - 使用绝对路径
	executable, err := os.Executable()
	if err != nil {
		c.JSON(500, response.Error("获取可执行文件路径失败"))
		return
	}
	baseDir := filepath.Join(filepath.Dir(executable), "data", "webdav")
	tempDir := filepath.Join(baseDir, "temp", uploadId)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(500, response.Error("创建临时目录失败"))
		return
	}

	// 保存上传信息
	infoFile := filepath.Join(tempDir, "info.txt")
	infoContent := fmt.Sprintf("fileName=%s\nfileSize=%d\ntotalChunks=%d\nchunkSize=%d\nparentId=%s", fileName, fileSize, totalChunks, chunkSize, parentId)
	if err := os.WriteFile(infoFile, []byte(infoContent), 0644); err != nil {
		c.JSON(500, response.Error("保存上传信息失败"))
		return
	}

	c.JSON(200, response.SuccessWithData(gin.H{
		"uploadId":    uploadId,
		"chunkSize":   chunkSize,
		"totalChunks": totalChunks,
		"fileName":    fileName,
		"fileSize":    fileSize,
		"parentId":    parentId,
	}))
}

// uploadChunk 上传分片
// @Summary 上传文件分片
// @Description 上传文件的一个分片
// @Tags 文件上传
// @Accept multipart/form-data
// @Produce json
// @Param uploadId formData string true "上传会话ID"
// @Param chunkIndex formData int true "分片索引"
// @Param chunk formData file true "分片数据"
// @Success 200 {object} map[string]interface{} "{"success": true}"
// @Failure 400 {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk [post]
func uploadChunk(c *gin.Context) {
	time.Sleep(1 * time.Second)
	uploadId := c.PostForm("uploadId")
	chunkIndexStr := c.PostForm("chunkIndex")

	if uploadId == "" || chunkIndexStr == "" {
		c.JSON(400, response.Error("uploadId和chunkIndex不能为空"))
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(400, response.Error("chunkIndex格式错误"))
		return
	}

	// 检查临时目录是否存在 - 使用绝对路径
	executable, err := os.Executable()
	if err != nil {
		c.JSON(500, response.Error("获取可执行文件路径失败"))
		return
	}
	baseDir := filepath.Join(filepath.Dir(executable), "data", "webdav")
	tempDir := filepath.Join(baseDir, "temp", uploadId)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(400, response.Error("上传会话不存在"))
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(400, response.Error("获取分片数据失败"))
		return
	}

	// 保存分片文件
	chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkIndex))
	if err := c.SaveUploadedFile(file, chunkFile); err != nil {
		c.JSON(500, response.Error("保存分片失败"))
		return
	}

	c.JSON(200, response.SuccessWithData(gin.H{
		"success":    true,
		"chunkIndex": chunkIndex,
		"uploadId":   uploadId,
	}))
}

// getUserID 从上下文中获取用户ID
func getUserID(c *gin.Context) uint64 {
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

// getMimeType 获取文件MIME类型
func getMimeType(filename string) string {
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

// completeChunkUpload 完成分片上传
// @Summary 完成分片上传
// @Description 合并所有分片并完成文件上传
// @Tags 文件上传
// @Accept json
// @Produce json
// @Param uploadId body string true "上传会话ID"
// @Success 200 {object} map[string]interface{} "{"id": 123, "name": "文件名", "size": 1024}"
// @Failure 400 {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/complete [post]
func completeChunkUpload(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UploadId string `json:"uploadId"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, response.Error("参数错误"))
			return
		}

		if req.UploadId == "" {
			c.JSON(400, response.Error("uploadId不能为空"))
			return
		}

		// 获取用户ID
		userID := getUserID(c)
		if userID == 0 {
			c.JSON(401, response.Error("未授权"))
			return
		}

		// 检查临时目录 - 使用绝对路径
		executable, err := os.Executable()
		if err != nil {
			c.JSON(500, response.Error("获取可执行文件路径失败"))
			return
		}
		baseDir := filepath.Join(filepath.Dir(executable), "data", "webdav")
		tempDir := filepath.Join(baseDir, "temp", req.UploadId)
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			c.JSON(400, response.Error("上传会话不存在"))
			return
		}

		// 读取上传信息文件
		infoFile := filepath.Join(tempDir, "info.txt")
		infoData, err := os.ReadFile(infoFile)
		if err != nil {
			c.JSON(500, response.Error("读取上传信息失败"))
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
			c.JSON(500, response.Error("读取分片失败"))
			return
		}

		// 检查分片完整性
		if len(files) != totalChunks {
			c.JSON(400, response.Error("分片不完整"))
			return
		}

		// 创建最终存储路径 - 使用绝对路径
		storageDir := filepath.Join(baseDir, fmt.Sprintf("user_%d", userID))
		if err := os.MkdirAll(storageDir, 0755); err != nil {
			c.JSON(500, response.Error("创建存储目录失败"))
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

		// 合并所有分片
		finalFile, err := os.Create(finalPath)
		if err != nil {
			c.JSON(500, response.Error("创建最终文件失败"))
			return
		}
		defer finalFile.Close()

		var totalSize int64
		for i := 0; i < totalChunks; i++ {
			chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
			chunkData, err := os.ReadFile(chunkFile)
			if err != nil {
				c.JSON(500, response.Error(fmt.Sprintf("读取分片 %d 失败", i)))
				return
			}

			if _, err := finalFile.Write(chunkData); err != nil {
				c.JSON(500, response.Error(fmt.Sprintf("写入分片 %d 失败", i)))
				return
			}

			totalSize += int64(len(chunkData))
		}

		// 清理临时目录
		os.RemoveAll(tempDir)

		// 创建文件数据库记录
		file := &model.File{
			UserID:      uint64(userID),
			ParentID:    parentId,
			Name:        fileName,
			IsFolder:    false,
			Size:        totalSize,
			StoragePath: filepath.Join(fmt.Sprintf("user_%d", userID), fileName),
			MimeType:    getMimeType(fileName),
		}

		// 保存到数据库
		if err := db.Create(file).Error; err != nil {
			// 删除已创建的文件
			os.Remove(finalPath)
			c.JSON(500, response.Error("保存文件记录失败"))
			return
		}

		c.JSON(200, response.SuccessWithData(gin.H{
			"id":   file.ID,
			"name": file.Name,
			"size": file.Size,
		}))
	}
}

// getUploadedChunks 获取已上传分片列表
// @Summary 获取已上传分片列表
// @Description 获取指定上传会话已上传的分片索引列表
// @Tags 文件上传
// @Produce json
// @Param uploadId path string true "上传会话ID"
// @Success 200 {object} map[string]interface{} "{"chunks": [0,1,2], "totalChunks": 10}"
// @Failure 400 {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/{uploadId}/chunks [get]
func getUploadedChunks(c *gin.Context) {
	uploadId := c.Param("uploadId")
	if uploadId == "" {
		c.JSON(400, response.Error("uploadId不能为空"))
		return
	}

	// 检查临时目录 - 使用绝对路径
	executable, err := os.Executable()
	if err != nil {
		c.JSON(500, response.Error("获取可执行文件路径失败"))
		return
	}
	baseDir := filepath.Join(filepath.Dir(executable), "data", "webdav")
	tempDir := filepath.Join(baseDir, "temp", uploadId)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(400, response.Error("上传会话不存在"))
		return
	}

	// 获取已上传的分片文件
	pattern := filepath.Join(tempDir, "chunk_*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		c.JSON(500, response.Error("读取分片列表失败"))
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

	c.JSON(200, response.SuccessWithData(gin.H{
		"chunks":      chunks,
		"totalChunks": totalChunks,
		"uploadId":    uploadId,
	}))
}

// cancelChunkUpload 取消分片上传
// @Summary 取消分片上传
// @Description 取消并清理分片上传会话
// @Tags 文件上传
// @Produce json
// @Param uploadId path string true "上传会话ID"
// @Success 200 {object} map[string]interface{} "{"success": true}"
// @Failure 400 {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk/{uploadId} [delete]
func cancelChunkUpload(c *gin.Context) {
	uploadId := c.Param("uploadId")
	if uploadId == "" {
		c.JSON(400, response.Error("uploadId不能为空"))
		return
	}

	// 检查临时目录
	tempDir := filepath.Join("uploads", "temp", uploadId)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(400, response.Error("上传会话不存在"))
		return
	}

	// 清理临时目录
	if err := os.RemoveAll(tempDir); err != nil {
		c.JSON(500, response.Error("清理临时文件失败"))
		return
	}

	c.JSON(200, response.SuccessWithData(gin.H{
		"success":  true,
		"uploadId": uploadId,
	}))
}
