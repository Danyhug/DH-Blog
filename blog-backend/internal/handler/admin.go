package handler

import (
	"net/http"

	"dh-blog/internal/response"
	"dh-blog/internal/service"
	"github.com/gin-gonic/gin"
)

// AdminHandler 负责处理后台管理相关的请求
type AdminHandler struct {
	UploadService *service.UploadService
	AIService     service.AIService // 添加 AIService 依赖
}

func NewAdminHandler(uploadService *service.UploadService, aiService service.AIService) *AdminHandler {
	return &AdminHandler{
		UploadService: uploadService,
		AIService:     aiService,
	}
}

// UploadFile 处理文件上传
func (h *AdminHandler) UploadFile(c *gin.Context) {
	// 从 URL 参数中获取上传类型
	uploadTypeStr := c.Param("type")
	uploadType := service.UploadType(uploadTypeStr)

	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "获取文件失败")
		return
	}

	url, err := h.UploadService.UploadFile(file, uploadType)
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(url))
}
