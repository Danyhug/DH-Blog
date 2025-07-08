package handler

import (
	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"dh-blog/internal/response"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Handler 统一的处理器接口
type Handler interface {
	// RegisterRoutes 注册路由
	RegisterRoutes(router *gin.RouterGroup)
}

// BaseHandler 基础处理器，提供通用方法
type BaseHandler struct{}

// Success 返回成功响应
func (h *BaseHandler) Success(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success())
}

// SuccessWithData 返回带数据的成功响应
func (h *BaseHandler) SuccessWithData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response.SuccessWithData(data))
}

// Error 返回错误响应
func (h *BaseHandler) Error(c *gin.Context, err error) {
	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
}

// BindJSON 绑定JSON请求体
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return errs.BadRequest("无效的请求参数", err)
	}
	return nil
}

// BindQuery 绑定查询参数
func (h *BaseHandler) BindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return errs.BadRequest("无效的查询参数", err)
	}
	return nil
}

// GetPageRequest 获取分页请求参数
func (h *BaseHandler) GetPageRequest(c *gin.Context) (*model.PageRequest, error) {
	var req model.PageRequest
	if err := h.BindQuery(c, &req); err != nil {
		// 如果绑定失败，使用默认值
		req = model.PageRequest{
			Page:     1,
			PageSize: 6, // 设置默认值为6
		}
		return &req, nil
	}

	// 如果绑定成功但值为0，设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 6 // 设置默认值为6
	}

	return &req, nil
}

// SuccessWithPage 返回分页结果
func (h *BaseHandler) SuccessWithPage(c *gin.Context, list interface{}, total int64, page int) {
	result := response.Page(total, int64(page), list)
	h.SuccessWithData(c, result)
}
