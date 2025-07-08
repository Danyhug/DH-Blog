package model

import (
	"gorm.io/gorm"
	"strconv"
)

// SystemSetting 映射到数据库中的 system_settings 表
type SystemSetting struct {
	ID           uint   `gorm:"primaryKey"`
	SettingKey   string `gorm:"unique;not null"`
	SettingValue string `gorm:"not null"`
}

// SystemConfig 用于前端交互，不直接映射数据库表
type SystemConfig struct {
	BlogTitle          string `json:"blog_title"`
	Signature          string `json:"signature"`
	Avatar             string `json:"avatar"`
	GithubLink         string `json:"github_link"`
	BilibiliLink       string `json:"bilibili_link"`
	OpenBlog           bool   `json:"open_blog"`
	OpenComment        bool   `json:"open_comment"`
	CommentEmailNotify bool   `json:"comment_email_notify"`
	SmtpHost           string `json:"smtp_host"`
	SmtpPort           int    `json:"smtp_port"`
	SmtpUser           string `json:"smtp_user"`
	SmtpPass           string `json:"smtp_pass"`
	SmtpSender         string `json:"smtp_sender"`
	AiApiURL           string `json:"ai_api_url"`
	AiApiKey           string `json:"ai_api_key"`
	AiModel            string `json:"ai_model"`
	AiPrompt           string `json:"ai_prompt"`
	// ... 其他配置项
}

// ToSettingsMap 将SystemConfig转换为map[string]string
func (c *SystemConfig) ToSettingsMap() map[string]string {
	return map[string]string{
		"blog_title":           c.BlogTitle,
		"signature":            c.Signature,
		"avatar":               c.Avatar,
		"github_link":          c.GithubLink,
		"bilibili_link":        c.BilibiliLink,
		"open_blog":            strconv.FormatBool(c.OpenBlog),
		"open_comment":         strconv.FormatBool(c.OpenComment),
		"comment_email_notify": strconv.FormatBool(c.CommentEmailNotify),
		"smtp_host":            c.SmtpHost,
		"smtp_port":            strconv.Itoa(c.SmtpPort),
		"smtp_user":            c.SmtpUser,
		"smtp_pass":            c.SmtpPass,
		"smtp_sender":          c.SmtpSender,
		"ai_api_url":           c.AiApiURL,
		"ai_api_key":           c.AiApiKey,
		"ai_model":             c.AiModel,
		"ai_prompt":            c.AiPrompt,
	}
}

// FromSettingsMap 从map[string]string创建SystemConfig
func FromSettingsMap(settings map[string]string) *SystemConfig {
	config := &SystemConfig{}

	// 字符串字段
	config.BlogTitle = settings["blog_title"]
	config.Signature = settings["signature"]
	config.Avatar = settings["avatar"]
	config.GithubLink = settings["github_link"]
	config.BilibiliLink = settings["bilibili_link"]
	config.SmtpHost = settings["smtp_host"]
	config.SmtpUser = settings["smtp_user"]
	config.SmtpPass = settings["smtp_pass"]
	config.SmtpSender = settings["smtp_sender"]
	config.AiApiURL = settings["ai_api_url"]
	config.AiApiKey = settings["ai_api_key"]
	config.AiModel = settings["ai_model"]
	config.AiPrompt = settings["ai_prompt"]

	// 布尔字段
	config.OpenBlog, _ = strconv.ParseBool(settings["open_blog"])
	config.OpenComment, _ = strconv.ParseBool(settings["open_comment"])
	config.CommentEmailNotify, _ = strconv.ParseBool(settings["comment_email_notify"])

	// 整数字段
	config.SmtpPort, _ = strconv.Atoi(settings["smtp_port"])

	return config
}

// BeforeSave 钩子，用于在保存前处理数据，例如加密敏感信息
func (s *SystemSetting) BeforeSave(tx *gorm.DB) (err error) {
	// 可以在这里添加对敏感信息的加密处理
	return
}

// BeforeDelete 钩子，用于在删除前处理数据
func (s *SystemSetting) BeforeDelete(tx *gorm.DB) (err error) {
	// 可以在这里添加删除前的逻辑
	return
}
