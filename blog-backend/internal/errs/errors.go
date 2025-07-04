package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// 定义常见的业务哨兵错误
var (
	ErrNotFound         = errors.New("资源未找到")
	ErrInvalidParameter = errors.New("无效的请求参数")
	ErrUnauthorized     = errors.New("未授权访问")
	ErrForbidden        = errors.New("禁止访问")
	ErrInternalServer   = errors.New("服务器内部错误")

	// ErrUserNotFound 针对特定业务场景的哨兵错误
	ErrUserNotFound          = errors.New("用户不存在")
	ErrPasswordMismatch      = errors.New("用户名或密码错误")
	ErrTagAlreadyExists      = errors.New("标签已存在")
	ErrCategoryAlreadyExists = errors.New("分类已存在")
	ErrCommentNotFound       = errors.New("评论不存在")
	ErrArticleNotFound       = errors.New("文章不存在")
)

// HTTPError 是一个自定义错误类型，用于携带 HTTP 状态码
type HTTPError struct {
	StatusCode int
	Message    string
	Err        error // 包装底层错误
}

// Error 实现 error 接口
func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 实现 errors.Wrapper 接口，支持错误链
func (e *HTTPError) Unwrap() error {
	return e.Err
}

// NewHTTPError 创建一个 HTTPError 实例
func NewHTTPError(statusCode int, message string, err error) error {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

func BadRequest(msg string, err error) error {
	return NewHTTPError(http.StatusBadRequest, msg, err)
}

func Unauthorized(msg string, err error) error {
	return NewHTTPError(http.StatusUnauthorized, msg, err)
}

func Forbidden(msg string, err error) error {
	return NewHTTPError(http.StatusForbidden, msg, err)
}

func NotFound(msg string, err error) error {
	return NewHTTPError(http.StatusNotFound, msg, err)
}

func InternalServerError(msg string, err error) error {
	return NewHTTPError(http.StatusInternalServerError, msg, err)
}
