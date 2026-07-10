package user

import (
	"errors"
	"fmt"
	"net/http"

	"dh-blog/internal/response"
	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
)

var (
	ErrPasswordMismatch = errors.New("用户名或密码错误")
	ErrLoginFailed      = errors.New("登录失败")
	ErrGenerateToken    = errors.New("生成 token 失败")
)

type Handler struct {
	repository *Repository
	tokens     TokenGenerator
}

type TokenGenerator interface {
	GenerateJWT(username string) (string, error)
}

func NewHandler(repository *Repository, tokens TokenGenerator) *Handler {
	return &Handler{repository: repository, tokens: tokens}
}

func (h *Handler) Login(c *gin.Context) {
	var credentials User
	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	foundUser, err := h.repository.GetByUsername(credentials.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, response.Error(ErrUserNotFound.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrLoginFailed.Error(), err)))
		return
	}

	if !utils.CheckPasswordHash(credentials.Password, foundUser.Password) {
		c.JSON(http.StatusUnauthorized, response.Error(ErrPasswordMismatch.Error()))
		return
	}

	if h.tokens == nil {
		c.JSON(http.StatusInternalServerError, response.Error(ErrGenerateToken.Error()))
		return
	}
	token, err := h.tokens.GenerateJWT(foundUser.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrGenerateToken.Error(), err)))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(token))
}

func (h *Handler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success())
}

func (h *Handler) Heart(c *gin.Context) {
	c.JSON(http.StatusOK, response.SuccessWithData("咚咚咚 ~ 咚咚咚 ~1"))
}
