package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *handler) getConfigs(c *gin.Context) {
	config, err := h.service.config(c.Request.Context())
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, config)
}

func (h *handler) updateConfigs(c *gin.Context) {
	var patch map[string]json.RawMessage
	if err := c.ShouldBindJSON(&patch); err != nil {
		failure(c, 400, err)
		return
	}
	old, err := h.service.config(c.Request.Context())
	if err != nil {
		failure(c, 500, err)
		return
	}
	config, err := mergeConfigPatch(old, patch)
	if err != nil {
		failure(c, 400, err)
		return
	}
	storageChanged := config.FileStoragePath != old.FileStoragePath || config.WebDAVChunkSize != old.WebDAVChunkSize
	if storageChanged {
		if err := validateStorageConfig(StorageConfig{FileStoragePath: config.FileStoragePath, WebDAVChunkSize: config.WebDAVChunkSize}); err != nil {
			failure(c, 400, err)
			return
		}
	}
	if err := h.service.updateConfig(c.Request.Context(), config); err != nil {
		failure(c, 500, err)
		return
	}
	if storageChanged {
		if err := h.storage.ApplyStorageConfig(c.Request.Context(), config.FileStoragePath, config.WebDAVChunkSize); err != nil {
			if rollbackErr := h.service.updateConfig(context.Background(), old); rollbackErr != nil {
				failure(c, 500, fmt.Errorf("应用配置失败: %v；回滚设置失败: %w", err, rollbackErr))
				return
			}
			failure(c, 500, fmt.Errorf("应用配置失败，设置已回滚: %w", err))
			return
		}
	}
	success(c)
}

func mergeConfigPatch(current Config, patch map[string]json.RawMessage) (Config, error) {
	encoded, err := json.Marshal(current)
	if err != nil {
		return Config{}, err
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &merged); err != nil {
		return Config{}, err
	}
	for key, value := range patch {
		if _, known := merged[key]; known {
			merged[key] = value
		}
	}
	encoded, err = json.Marshal(merged)
	if err != nil {
		return Config{}, err
	}
	var result Config
	if err := json.Unmarshal(encoded, &result); err != nil {
		return Config{}, fmt.Errorf("无效的配置字段: %w", err)
	}
	return result, nil
}

func (h *handler) getBlogConfig(c *gin.Context) {
	config, err := h.service.configByType(c.Request.Context(), ConfigTypeBlog)
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, BlogConfig{config.BlogTitle, config.Signature, config.Avatar, config.GithubLink, config.BilibiliLink, config.OpenBlog, config.OpenComment})
}
func (h *handler) updateBlogConfig(c *gin.Context) {
	var config BlogConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		failure(c, 400, err)
		return
	}
	values := map[string]string{SettingKeyBlogTitle: config.BlogTitle, SettingKeySignature: config.Signature, SettingKeyAvatar: config.Avatar, SettingKeyGithubLink: config.GithubLink, SettingKeyBilibiliLink: config.BilibiliLink, SettingKeyOpenBlog: strconv.FormatBool(config.OpenBlog), SettingKeyOpenComment: strconv.FormatBool(config.OpenComment)}
	if err := h.service.settings.updateBatch(c.Request.Context(), values, ConfigTypeBlog); err != nil {
		failure(c, 500, err)
		return
	}
	success(c)
}

func (h *handler) getEmailConfig(c *gin.Context) {
	config, err := h.service.configByType(c.Request.Context(), ConfigTypeEmail)
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, EmailConfig{config.CommentEmailNotify, config.SmtpHost, config.SmtpPort, config.SmtpUser, config.SmtpPass, config.SmtpSender})
}
func (h *handler) updateEmailConfig(c *gin.Context) {
	var config EmailConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		failure(c, 400, err)
		return
	}
	values := map[string]string{SettingKeyCommentEmailNotify: strconv.FormatBool(config.CommentEmailNotify), SettingKeySmtpHost: config.SmtpHost, SettingKeySmtpPort: strconv.Itoa(config.SmtpPort), SettingKeySmtpUser: config.SmtpUser, SettingKeySmtpPass: config.SmtpPass, SettingKeySmtpSender: config.SmtpSender}
	if err := h.service.settings.updateBatch(c.Request.Context(), values, ConfigTypeEmail); err != nil {
		failure(c, 500, err)
		return
	}
	success(c)
}

func (h *handler) getAIConfig(c *gin.Context) {
	config, err := h.service.configByType(c.Request.Context(), ConfigTypeAI)
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, AIConfig{APIURL: config.AIAPIURL, APIKey: config.AIAPIKey, Model: config.AIModel})
}
func (h *handler) updateAIConfig(c *gin.Context) {
	var config AIConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		failure(c, 400, err)
		return
	}
	values := map[string]string{SettingKeyAIAPIURL: config.APIURL, SettingKeyAIAPIKey: config.APIKey, SettingKeyAIModel: config.Model}
	if err := h.service.settings.updateBatch(c.Request.Context(), values, ConfigTypeAI); err != nil {
		failure(c, 500, err)
		return
	}
	success(c)
}

func (h *handler) getAIPromptTags(c *gin.Context) {
	tags, err := h.service.settings.value(c.Request.Context(), SettingKeyAIPromptGetTags)
	if err != nil {
		failure(c, 500, err)
		return
	}
	abstract, err := h.service.settings.value(c.Request.Context(), SettingKeyAIPromptGetAbstract)
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, []gin.H{{"label": "文章标签提取", "prompt": tags}, {"label": "文章摘要生成", "prompt": abstract}})
}
