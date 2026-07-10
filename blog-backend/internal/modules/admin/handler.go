package admin

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	filesmodule "dh-blog/internal/modules/files"
	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type handler struct {
	fileService filesmodule.Service
}

func newHandler(fileService filesmodule.Service) *handler {
	return &handler{fileService: fileService}
}

func (h *handler) UploadFile(c *gin.Context) {
	uploadType := c.Param("type")
	if uploadType != "blog" {
		if uploadType == "webdav" {
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

	userID := userIDFromContext(c)
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

	c.JSON(http.StatusOK, response.SuccessWithData(filepath.ToSlash(savedFile.StoragePath)))
}

func userIDFromContext(c *gin.Context) uint64 {
	userID, exists := c.Get("userID")
	if !exists {
		return 0
	}
	switch id := userID.(type) {
	case float64:
		return uint64(id)
	case uint64:
		return id
	default:
		return 0
	}
}
