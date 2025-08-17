package handler

import (
	"errors"
	"fmt"
	"net/http"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
)

// 用户相关错误常量
var (
	ErrUserNotFound     = errors.New("用户不存在")
	ErrPasswordMismatch = errors.New("用户名或密码错误")
	ErrLoginFailed      = errors.New("登录失败")
	ErrGenerateToken    = errors.New("生成 token 失败")
)

type UserHandler struct {
	BaseHandler
	repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	foundUser, err := h.repo.GetUserByUsername(user.Username)
	if err != nil {
		// 使用 errors.Is 检查是否是特定的哨兵错误
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, response.Error(ErrUserNotFound.Error()))
			return
		}
		// 如果是其他类型的错误（例如数据库错误），返回通用错误
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrLoginFailed.Error(), err)))
		return
	}

	// 验证密码
	if !utils.CheckPasswordHash(user.Password, foundUser.Password) {
		c.JSON(http.StatusUnauthorized, response.Error(ErrPasswordMismatch.Error()))
		return
	}

	// 生成 JWT Token
	token, err := utils.GenerateJWT(foundUser.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrGenerateToken.Error(), err)))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(token))
}

// Check 用户校验 (假设 JWT 认证中间件已经验证了 token)
func (h *UserHandler) Check(c *gin.Context) {
	// 假设 JWT 认证中间件已经验证了 token
	// 如果能到达这里，说明 token 有效
	c.JSON(http.StatusOK, response.Success()) // 返回成功，不带数据
}

func (h *UserHandler) Heart(c *gin.Context) {
	// TODO SSE或者WebSocket实现
	c.JSON(http.StatusOK, response.SuccessWithData("咚咚咚 ~ 咚咚咚 ~1"))
}

// RegisterRoutes 注册路由
func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup) {
	// 用户公共 API
	router.POST("/user/login", h.Login)
	router.POST("/user/check", h.Check)
	router.GET("/user/heart", h.Heart)
}
