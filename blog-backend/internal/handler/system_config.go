package handler

import (
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

type SystemConfigHandler interface {
	GetConfigs(c *gin.Context)
	UpdateConfigs(c *gin.Context)
}

type systemConfigHandler struct {
	configRepo repository.SystemConfigRepository
}

func NewSystemConfigHandler(configRepo repository.SystemConfigRepository) SystemConfigHandler {
	return &systemConfigHandler{configRepo: configRepo}
}

func (h *systemConfigHandler) GetConfigs(c *gin.Context) {
	config, err := h.configRepo.Get()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果没有找到记录，则创建默认配置
			defaultConfig := &model.SystemConfig{
				BlogTitle:          "DH-Blog",
				Signature:          "保持热爱，奔赴山海",
				Avatar:             "https://www.danyhug.com/avatar.jpg",
				GithubLink:         "https://github.com/danyhug",
				BilibiliLink:       "https://space.bilibili.com/your_bilibili_link",
				OpenBlog:           true,
				OpenComment:        true,
				CommentEmailNotify: false,
				SmtpHost:           "",
				SmtpPort:           0,
				SmtpUser:           "",
				SmtpPass:           "",
				SmtpSender:         "",
				AiPrompt:           "",
			}
			err = h.configRepo.Update(defaultConfig)
			if err != nil {
				c.JSON(http.StatusOK, response.Error("创建默认配置失败: "+err.Error()))
				return
			}
			c.JSON(http.StatusOK, response.SuccessWithData(defaultConfig))
			return
		}
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(config))
}

func (h *systemConfigHandler) UpdateConfigs(c *gin.Context) {
	var config model.SystemConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusOK, response.Error("参数错误"))
		return
	}

	err := h.configRepo.Update(&config)
	if err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success())
}
