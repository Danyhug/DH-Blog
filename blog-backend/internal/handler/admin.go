package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"dh-blog/internal/response"
	"dh-blog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AdminHandler 负责处理后台管理相关的请求
type AdminHandler struct {
	fileService service.IFileService
	AIService   service.AIService // 添加 AIService 依赖
}

func NewAdminHandler(fileService service.IFileService, aiService service.AIService) *AdminHandler {
	return &AdminHandler{
		fileService: fileService,
		AIService:   aiService,
	}
}

// UploadFile 处理文件上传
func (h *AdminHandler) UploadFile(c *gin.Context) {
	// 从 URL 参数中获取上传类型
	uploadTypeStr := c.Param("type")
	if uploadTypeStr != "blog" {
		if uploadTypeStr == "webdav" {
			response.FailWithCode(c, http.StatusInternalServerError, "WebDAV 上传功能尚未实现")
			return
		}
		response.FailWithCode(c, http.StatusInternalServerError, "不支持的上传类型")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "获取文件失败")
		return
	}

	userID := h.getUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	parentID, err := h.fileService.GetProtectedDirectoryID(c.Request.Context(), "博客")
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, fmt.Sprintf("获取存储目录失败: %v", err))
		return
	}

	src, err := file.Open()
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, "读取文件失败")
		return
	}
	defer func() {
		if err := src.Close(); err != nil {
			logrus.Errorf("关闭上传文件失败: %v", err)
		}
	}()

	originName := filepath.Base(file.Filename)
	if originName == "" {
		response.FailWithCode(c, http.StatusBadRequest, "文件名无效")
		return
	}
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), originName)

	savedFile, err := h.fileService.UploadFile(c.Request.Context(), userID, parentID, fileName, file.Size, src)
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	urlPath := filepath.ToSlash(savedFile.StoragePath)
	c.JSON(http.StatusOK, response.SuccessWithData(urlPath))
}

func (h *AdminHandler) getUserID(c *gin.Context) uint64 {
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
