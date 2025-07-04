package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"dh-blog/internal/response"
	"github.com/gin-gonic/gin"
)

// AdminHandler 负责处理后台管理相关的请求
type AdminHandler struct {
	// 这里可以注入 AdminService 或其他 Repository
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// UploadFile 处理文件上传
func (h *AdminHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error("获取文件失败"))
		return
	}

	// 确保上传目录存在
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, os.ModePerm)
	}

	// 生成唯一文件名
	filename := filepath.Base(file.Filename)
	extension := filepath.Ext(filename)
	newFilename := strconv.FormatInt(time.Now().UnixNano(), 10) + extension

	// 保存文件
	destination := filepath.Join(uploadDir, newFilename)
	if err := c.SaveUploadedFile(file, destination); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("保存文件失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{"filename": newFilename}))
}
