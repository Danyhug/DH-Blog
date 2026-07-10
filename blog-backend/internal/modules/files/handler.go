package files

import (
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FileController 文件控制器
type handler struct {
	fileService *fileService
}

// newHandler 创建文件 HTTP handler。
func newHandler(fileService *fileService) *handler {
	return &handler{
		fileService: fileService,
	}
}

// ListFiles 列出文件
// @Summary 列出指定目录下的文件和文件夹
// @Description 获取指定目录下的文件和文件夹列表
// @Tags 文件
// @Accept json
// @Produce json
// @Param parentId query string false "父目录ID，空表示根目录"
// @Success 200 {object} []files.File "文件列表"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/list [get]
func (h *handler) ListFiles(c *gin.Context) {
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

// CreateFolder 创建文件夹
// @Summary 创建文件夹
// @Description 在指定目录下创建新文件夹
// @Tags 文件
// @Accept json
// @Produce json
// @Param request body map[string]string true "包含 parentId 和 folderName 的请求体"
// @Success 200 {object} files.File "创建的文件夹信息"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/folder [post]
func (h *handler) CreateFolder(c *gin.Context) {
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
func (h *handler) DownloadFile(c *gin.Context) {
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

	// 判断是否为预览模式（用于音视频流式传输）
	preview := c.Query("preview") == "true"
	disposition := "attachment"
	if preview {
		disposition = "inline"
	}

	// 设置响应头
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, fileName))
	c.Header("Content-Type", contentType)
	// c.File 内部使用 http.ServeFile，自动支持 Range 请求
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
func (h *handler) RenameFile(c *gin.Context) {
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
func (h *handler) DeleteFile(c *gin.Context) {
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

// GetDirectoryTree 获取系统目录树
// @Summary 获取系统目录树
// @Description 获取系统目录结构，用于前端选择存储路径
// @Tags 文件
// @Accept json
// @Produce json
// @Param rootPath query string false "根目录路径，为空则使用系统根目录"
// @Param maxDepth query int false "最大深度，默认为2"
// @Success 200 {object} files.DirectoryNode "目录树结构"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/files/directory-tree [get]
func (h *handler) GetDirectoryTree(c *gin.Context) {
	// 检查权限（仅允许管理员操作）
	userID := h.getCurrentUserID(c)
	if userID != 1 { // 假设ID为1的用户是管理员
		response.FailWithCode(c, http.StatusUnauthorized, "只有管理员可以查看系统目录")
		return
	}

	// 获取请求参数
	rootPath := c.Query("rootPath")
	maxDepthStr := c.DefaultQuery("maxDepth", "2")

	maxDepth, err := strconv.Atoi(maxDepthStr)
	if err != nil || maxDepth < 0 {
		maxDepth = 2 // 默认深度为2
	}

	// 调用服务获取目录树
	directoryTree, err := h.fileService.GetSystemDirectoryTree(c.Request.Context(), rootPath, maxDepth)
	if err != nil {
		logrus.Errorf("获取目录树失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "获取目录树失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(directoryTree))
}

// requireAuth 验证用户是否已登录
func (h *handler) requireAuth() gin.HandlerFunc {
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
func (h *handler) getCurrentUserID(c *gin.Context) uint64 {
	userID, exists := c.Get("userID")
	if !exists {
		return 0
	}
	return userID.(uint64)
}

// SyncFiles 手动同步磁盘文件到数据库
func (h *handler) SyncFiles(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID != 1 {
		response.FailWithCode(c, http.StatusUnauthorized, "只有管理员可以同步文件")
		return
	}

	if err := h.fileService.SyncFilesFromDisk(); err != nil {
		logrus.Errorf("同步文件失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "同步文件失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, response.AjaxResult{
		Code: 1,
		Msg:  "文件同步完成",
	})
}
