package handler

import (
	"errors"
	"net/http"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"dh-blog/internal/utils"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusOK, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	foundUser, err := h.repo.GetUserByUsername(user.Username)
	if err != nil {
		// 使用 errors.Is 检查是否是特定的哨兵错误
		if errors.Is(err, errs.ErrUserNotFound) {
			c.JSON(http.StatusOK, response.Error(errs.Unauthorized(errs.ErrUserNotFound.Error(), nil).Error()))
			return
		}
		// 如果是其他类型的错误（例如数据库错误），返回通用错误
		c.JSON(http.StatusOK, response.Error(errs.InternalServerError("登录失败", err).Error()))
		return
	}

	// 验证密码
	if !utils.CheckPasswordHash(user.Password, foundUser.Password) {
		c.JSON(http.StatusOK, response.Error(errs.Unauthorized(errs.ErrPasswordMismatch.Error(), nil).Error()))
		return
	}

	// 生成 JWT Token
	token, err := utils.GenerateJWT(foundUser.Username)
	if err != nil {
		c.JSON(http.StatusOK, response.Error(errs.InternalServerError("生成 token 失败", err).Error()))
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
