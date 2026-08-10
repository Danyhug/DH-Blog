package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *handler) getBlogConfig(c *gin.Context) {
	config, err := h.service.configByType(c.Request.Context(), ConfigTypeBlog)
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, BlogConfig{config.BlogTitle, config.Signature, config.Avatar, config.GithubLink, config.BilibiliLink, config.OpenComment})
}
func (h *handler) updateBlogConfig(c *gin.Context) {
	var config BlogConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		failure(c, 400, err)
		return
	}
	values := map[string]string{SettingKeyBlogTitle: config.BlogTitle, SettingKeySignature: config.Signature, SettingKeyAvatar: config.Avatar, SettingKeyGithubLink: config.GithubLink, SettingKeyBilibiliLink: config.BilibiliLink, SettingKeyOpenComment: strconv.FormatBool(config.OpenComment)}
	if err := h.service.settings.updateBatch(c.Request.Context(), values, ConfigTypeBlog); err != nil {
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

// editablePrompts 限定后台可编辑的提示词，避免任意键被写入。
var editablePrompts = []AIPrompt{
	{Key: SettingKeyAIPromptGetTags, Label: "文章标签提取"},
	{Key: SettingKeyAIPromptGetAbstract, Label: "文章摘要生成"},
}

func (h *handler) getAIPromptTags(c *gin.Context) {
	prompts := make([]AIPrompt, 0, len(editablePrompts))
	for _, item := range editablePrompts {
		value, err := h.service.settings.value(c.Request.Context(), item.Key)
		if err != nil {
			failure(c, 500, err)
			return
		}
		prompts = append(prompts, AIPrompt{Key: item.Key, Label: item.Label, Prompt: value})
	}
	success(c, prompts)
}

func (h *handler) updateAIPromptTags(c *gin.Context) {
	var prompts []AIPrompt
	if err := c.ShouldBindJSON(&prompts); err != nil {
		failure(c, 400, err)
		return
	}
	allowed := make(map[string]bool, len(editablePrompts))
	for _, item := range editablePrompts {
		allowed[item.Key] = true
	}
	values := make(map[string]string, len(prompts))
	for _, prompt := range prompts {
		if !allowed[prompt.Key] {
			failure(c, 400, fmt.Errorf("未知的提示词: %s", prompt.Key))
			return
		}
		if strings.TrimSpace(prompt.Prompt) == "" {
			failure(c, 400, fmt.Errorf("提示词内容不能为空: %s", prompt.Label))
			return
		}
		values[prompt.Key] = prompt.Prompt
	}
	if len(values) == 0 {
		success(c)
		return
	}
	if err := h.service.settings.updateBatch(c.Request.Context(), values, ConfigTypeAI); err != nil {
		failure(c, 500, err)
		return
	}
	success(c)
}

// getSiteConfig 暴露前台展示所需的站点信息，不需要鉴权。
func (h *handler) getSiteConfig(c *gin.Context) {
	config, err := h.service.configByType(c.Request.Context(), ConfigTypeBlog)
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, BlogConfig{config.BlogTitle, config.Signature, config.Avatar, config.GithubLink, config.BilibiliLink, config.OpenComment})
}
