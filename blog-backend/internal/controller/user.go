package controller

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
	ErrPasswordMismatch = errors.New("用户名或密码错误")
	ErrLoginFailed      = errors.New("登录失败")
	ErrGenerateToken    = errors.New("生成 token 失败")
)

type UserController struct {
	repo *repository.UserRepository
}

func NewUserController(repo *repository.UserRepository) *UserController {
	return &UserController{repo: repo}
}

// Login 用户登录
func (h *UserController) Login(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	foundUser, err := h.repo.GetUserByUsername(user.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, response.Error(repository.ErrUserNotFound.Error()))
			return
		}

		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrLoginFailed.Error(), err)))
		return
	}

	if !utils.CheckPasswordHash(user.Password, foundUser.Password) {
		c.JSON(http.StatusUnauthorized, response.Error(ErrPasswordMismatch.Error()))
		return
	}

	token, err := utils.GenerateJWT(foundUser.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrGenerateToken.Error(), err)))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(token))
}

func (h *UserController) Check(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success())
}

func (h *UserController) Heart(c *gin.Context) {
	// TODO SSE或者WebSocket实现
	c.JSON(http.StatusOK, response.SuccessWithData("咚咚咚 ~ 咚咚咚 ~1"))
}
