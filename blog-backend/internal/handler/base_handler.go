package handler

import (
	"errors"
	"net/http"
	"strconv"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler 统一的处理器接口
type Handler interface {
	RegisterRoutes(router *gin.RouterGroup)
}

// 通用错误常量，所有处理器可以共用
var (
	ErrInvalidID         = errors.New("无效的ID")
	ErrInvalidParams     = errors.New("无效的请求参数")
	ErrParamBinding      = errors.New("请求参数绑定失败")
	ErrPageParamBinding  = errors.New("分页参数绑定失败")
	ErrPasswordIncorrect = errors.New("密码错误")
)

// BaseHandler 提供基本的处理器功能
type BaseHandler struct{}

// Error 统一处理错误响应
func (h *BaseHandler) Error(c *gin.Context, err error) {
	// 根据错误类型设置不同的HTTP状态码
	statusCode := http.StatusInternalServerError

	// 检查特定的错误类型
	switch {
	case errors.Is(err, ErrInvalidID) ||
		errors.Is(err, ErrInvalidParams) ||
		errors.Is(err, ErrParamBinding) ||
		errors.Is(err, ErrPageParamBinding):
		statusCode = http.StatusBadRequest
	case errors.Is(err, ErrPasswordIncorrect):
		statusCode = http.StatusForbidden
	default:
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, response.Error(err.Error()))
}

// Success 返回成功响应
func (h *BaseHandler) Success(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success())
}

// SuccessWithData 返回带数据的成功响应
func (h *BaseHandler) SuccessWithData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response.SuccessWithData(data))
}

// SuccessWithMessage 返回带自定义消息的成功响应
func (h *BaseHandler) SuccessWithMessage(c *gin.Context, message string) {
	c.JSON(200, response.SuccessWithData(message))
}

// ErrorWithMessage 返回带自定义消息的错误响应
func (h *BaseHandler) ErrorWithMessage(c *gin.Context, message string) {
	c.JSON(400, response.Error(message))
}

// SuccessWithPage 返回带分页信息的成功响应
func (h *BaseHandler) SuccessWithPage(c *gin.Context, data interface{}, total int64, page int) {
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), data)))
}

// GetQueryInt 获取查询参数的整数值
func (h *BaseHandler) GetQueryInt(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
