package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// 预定义错误码
const (
	// 成功
	CodeSuccess = 200

	// 客户端错误 (4xx)
	CodeBadRequest          = 400
	CodeUnauthorized        = 401
	CodeForbidden           = 403
	CodeNotFound            = 404
	CodeMethodNotAllowed    = 405
	CodeConflict            = 409
	CodeTooManyRequests     = 429
	
	// 服务器错误 (5xx)
	CodeInternalServerError = 500
	CodeServiceUnavailable  = 503
)

// AppError 是一个自定义的错误类型，用于在应用程序中传递带有状态码的错误信息
type AppError struct {
	StatusCode int
	Message    string
	Err        error
}

// Error 实现了 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 返回内部错误，以支持 errors.Is 和 errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 创建一个新的 AppError
func NewAppError(statusCode int, message string, err error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// BadRequest 创建一个表示客户端错误的 AppError
func BadRequest(message string, err error) *AppError {
	if message == "" {
		message = "错误的请求"
	}
	return NewAppError(http.StatusBadRequest, message, err)
}

// Unauthorized 创建一个表示未授权的 AppError
func Unauthorized(message string, err error) *AppError {
	if message == "" {
		message = "未授权"
	}
	return NewAppError(http.StatusUnauthorized, message, err)
}

// Forbidden 创建一个表示禁止访问的 AppError
func Forbidden(message string, err error) *AppError {
	if message == "" {
		message = "禁止访问"
	}
	return NewAppError(http.StatusForbidden, message, err)
}

// NotFound 创建一个表示资源未找到的 AppError
func NotFound(message string, err error) *AppError {
	if message == "" {
		message = "资源未找到"
	}
	return NewAppError(http.StatusNotFound, message, err)
}

// InternalServerError 创建一个表示服务器内部错误的 AppError
func InternalServerError(message string, err error) *AppError {
	if message == "" {
		message = "服务器内部错误"
	}
	return NewAppError(http.StatusInternalServerError, message, err)
}

// 预定义错误
var (
	ErrNotFound            = errors.New("资源不存在")
	ErrBadRequest          = errors.New("无效的请求")
	ErrUnauthorized        = errors.New("未授权")
	ErrForbidden           = errors.New("禁止访问")
	ErrInternalServer      = errors.New("服务器内部错误")
	ErrTagAlreadyExists    = errors.New("标签已存在")
	ErrCategoryAlreadyExists = errors.New("分类已存在")
	ErrArticleNotFound     = errors.New("文章不存在")
	ErrCommentNotFound     = errors.New("评论不存在")
	ErrUserNotFound        = errors.New("用户不存在")
	ErrPasswordMismatch    = errors.New("用户名或密码错误")
)

// IsAppError 检查错误是否为AppError类型
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// GetStatusCode 从错误中获取HTTP状态码
func GetStatusCode(err error) int {
	if appErr, ok := IsAppError(err); ok {
		return appErr.StatusCode
	}
	
	// 根据预定义错误类型返回对应的HTTP状态码
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrArticleNotFound), errors.Is(err, ErrCommentNotFound), errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrPasswordMismatch):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrTagAlreadyExists), errors.Is(err, ErrCategoryAlreadyExists):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
