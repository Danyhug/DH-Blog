package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/response"
	"dh-blog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FileHandler 文件处理器
type FileHandler struct {
	fileService service.IFileService
}

// NewFileHandler 创建文件处理器
func NewFileHandler(fileService service.IFileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

// RegisterRoutes 注册路由
func (h *FileHandler) RegisterRoutes(router *gin.RouterGroup) {
	fileGroup := router.Group("/files")
	{
		fileGroup.GET("/list", h.ListFiles)
		fileGroup.POST("/upload", h.UploadFile)
		fileGroup.POST("/folder", h.CreateFolder)
		fileGroup.GET("/download/:id", h.DownloadFile)
		fileGroup.PUT("/rename/:id", h.RenameFile)
		fileGroup.DELETE("/:id", h.DeleteFile)
		fileGroup.PUT("/storage-path", h.UpdateStoragePath) // 添加更新存储路径的路由
	}
}

// ListFiles 列出文件
// @Summary 列出指定目录下的文件和文件夹
// @Description 获取指定目录下的文件和文件夹列表
// @Tags 文件
// @Accept json
// @Produce json
// @Param parentId query string false "父目录ID，空表示根目录"
// @Success 200 {object} []model.File "文件列表"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/list [get]
func (h *FileHandler) ListFiles(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	parentID := c.Query("parentId")

	files, err := h.fileService.ListFiles(c.Request.Context(), userID, parentID)
	if err != nil {
		logrus.Errorf("列出文件失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "获取文件列表失败")
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(files))
}

// UploadFile 上传文件
// @Summary 上传文件
// @Description 将文件上传到指定目录
// @Tags 文件
// @Accept multipart/form-data
// @Produce json
// @Param parentId formData string false "父目录ID，空表示根目录"
// @Param file formData file true "要上传的文件"
// @Success 200 {object} model.File "上传的文件信息"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	parentID := c.PostForm("parentId")

	// 获取文件
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "获取文件失败")
		return
	}
	defer file.Close()

	// 上传文件
	uploadedFile, err := h.fileService.UploadFile(
		c.Request.Context(),
		userID,
		parentID,
		fileHeader.Filename,
		fileHeader.Size,
		file,
	)

	if err != nil {
		logrus.Errorf("上传文件失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, fmt.Sprintf("上传失败: %v", err))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(uploadedFile))
}

// CreateFolder 创建文件夹
// @Summary 创建文件夹
// @Description 在指定目录下创建新文件夹
// @Tags 文件
// @Accept json
// @Produce json
// @Param request body map[string]string true "包含 parentId 和 folderName 的请求体"
// @Success 200 {object} model.File "创建的文件夹信息"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/folder [post]
func (h *FileHandler) CreateFolder(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	var request struct {
		ParentID   string `json:"parentId"`
		FolderName string `json:"folderName"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误")
		return
	}

	if request.FolderName == "" {
		response.FailWithCode(c, http.StatusBadRequest, "文件夹名称不能为空")
		return
	}

	folder, err := h.fileService.CreateFolder(
		c.Request.Context(),
		userID,
		request.ParentID,
		request.FolderName,
	)

	if err != nil {
		logrus.Errorf("创建文件夹失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, fmt.Sprintf("创建失败: %v", err))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(folder))
}

// DownloadFile 下载文件
// @Summary 下载文件
// @Description 根据文件ID下载文件
// @Tags 文件
// @Produce octet-stream
// @Param id path string true "文件ID"
// @Success 200 {file} file "文件内容"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 404 {object} response.Response "文件不存在"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/download/{id} [get]
func (h *FileHandler) DownloadFile(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	fileID := c.Param("id")
	if fileID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	// 获取文件信息
	fileInfo, err := h.fileService.GetDownloadInfo(c.Request.Context(), userID, fileID)
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, fmt.Sprintf("获取文件失败: %v", err))
		return
	}

	// 设置文件名和内容类型
	fileName := fileInfo.Name
	contentType := fileInfo.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 设置响应头
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", contentType)
	c.File(fileInfo.StoragePath)
}

// RenameFile 重命名文件
// @Summary 重命名文件或文件夹
// @Description 根据ID重命名文件或文件夹
// @Tags 文件
// @Accept json
// @Produce json
// @Param id path string true "文件ID"
// @Param request body map[string]string true "包含 newName 的请求体"
// @Success 200 {object} response.Response "重命名成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/rename/{id} [put]
func (h *FileHandler) RenameFile(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	fileID := c.Param("id")
	if fileID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	var request struct {
		NewName string `json:"newName"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误")
		return
	}

	if request.NewName == "" {
		response.FailWithCode(c, http.StatusBadRequest, "新名称不能为空")
		return
	}

	err := h.fileService.RenameFile(c.Request.Context(), userID, fileID, request.NewName)
	if err != nil {
		logrus.Errorf("重命名文件失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, fmt.Sprintf("重命名失败: %v", err))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// DeleteFile 删除文件
// @Summary 删除文件或文件夹
// @Description 根据ID删除文件或文件夹
// @Tags 文件
// @Accept json
// @Produce json
// @Param id path string true "文件ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/{id} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	fileID := c.Param("id")
	if fileID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	err := h.fileService.DeleteFile(c.Request.Context(), userID, fileID)
	if err != nil {
		logrus.Errorf("删除文件失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, fmt.Sprintf("删除失败: %v", err))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// UpdateStoragePath 更新文件存储路径
func (h *FileHandler) UpdateStoragePath(c *gin.Context) {
	// 检查权限（仅允许管理员操作）
	userID := h.getCurrentUserID(c)
	if userID != 1 { // 假设ID为1的用户是管理员
		response.FailWithCode(c, http.StatusUnauthorized, "只有管理员可以更新存储路径")
		return
	}

	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "请求参数无效: "+err.Error())
		return
	}

	if req.Path == "" {
		response.FailWithCode(c, http.StatusBadRequest, "存储路径不能为空")
		return
	}

	// 调用文件服务更新存储路径并清空文件表
	err := h.fileService.UpdateStoragePath(req.Path)
	if err != nil {
		logrus.Errorf("更新存储路径失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "更新存储路径失败: "+err.Error())
		return
	}

	// 使用自定义消息的响应
	c.JSON(http.StatusOK, response.AjaxResult{
		Code: 1,
		Msg:  "存储路径已更新，文件表已清空，所有文件需要重新上传",
	})
}

// requireAuth 验证用户是否已登录
func (h *FileHandler) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证JWT或会话
		// 简化版：从Query或Header中获取userID
		userIDStr := c.Query("userId")
		if userIDStr == "" {
			userIDStr = c.GetHeader("X-User-ID")
		}

		if userIDStr == "" {
			response.FailWithCode(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil || userID == 0 {
			response.FailWithCode(c, http.StatusUnauthorized, "无效的用户ID")
			c.Abort()
			return
		}

		// 设置用户ID到上下文
		c.Set("userID", userID)
		c.Next()
	}
}

// getCurrentUserID 获取当前用户ID
func (h *FileHandler) getCurrentUserID(c *gin.Context) uint64 {
	userID, exists := c.Get("userID")
	if !exists {
		return 0
	}
	return userID.(uint64)
}
