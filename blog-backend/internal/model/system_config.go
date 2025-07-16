package model

import (
	"strconv"

	"gorm.io/gorm"
)

// 配置类型常量
const (
	ConfigTypeBlog    = "blog"    // 博客基本配置
	ConfigTypeEmail   = "email"   // 邮件配置
	ConfigTypeAI      = "ai"      // AI相关配置
	ConfigTypeStorage = "storage" // 存储配置
)

// SystemSetting 映射到数据库中的 system_settings 表
type SystemSetting struct {
	ID           uint   `gorm:"primaryKey"`
	SettingKey   string `gorm:"unique;not null"`
	SettingValue string `gorm:"not null"`
	ConfigType   string `gorm:"not null;default:'blog'"` // 新增配置类型字段，默认为blog
}

// SystemConfig 用于前端交互，不直接映射数据库表
type SystemConfig struct {
	// 博客基本配置
	BlogTitle    string `json:"blog_title"`
	Signature    string `json:"signature"`
	Avatar       string `json:"avatar"`
	GithubLink   string `json:"github_link"`
	BilibiliLink string `json:"bilibili_link"`
	OpenBlog     bool   `json:"open_blog"`
	OpenComment  bool   `json:"open_comment"`

	// 邮件配置
	CommentEmailNotify bool   `json:"comment_email_notify"`
	SmtpHost           string `json:"smtp_host"`
	SmtpPort           int    `json:"smtp_port"`
	SmtpUser           string `json:"smtp_user"`
	SmtpPass           string `json:"smtp_pass"`
	SmtpSender         string `json:"smtp_sender"`

	// AI配置
	AiApiURL string `json:"ai_api_url"`
	AiApiKey string `json:"ai_api_key"`
	AiModel  string `json:"ai_model"`
	AiPrompt string `json:"ai_prompt"`

	// 存储配置
	FileStoragePath string `json:"file_storage_path"` // 文件存储路径
	// ... 其他配置项
}

// BlogConfig 博客基本配置
type BlogConfig struct {
	BlogTitle    string `json:"blog_title"`
	Signature    string `json:"signature"`
	Avatar       string `json:"avatar"`
	GithubLink   string `json:"github_link"`
	BilibiliLink string `json:"bilibili_link"`
	OpenBlog     bool   `json:"open_blog"`
	OpenComment  bool   `json:"open_comment"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	CommentEmailNotify bool   `json:"comment_email_notify"`
	SmtpHost           string `json:"smtp_host"`
	SmtpPort           int    `json:"smtp_port"`
	SmtpUser           string `json:"smtp_user"`
	SmtpPass           string `json:"smtp_pass"`
	SmtpSender         string `json:"smtp_sender"`
}

// AIConfig AI配置
type AIConfig struct {
	AiApiURL string `json:"ai_api_url"`
	AiApiKey string `json:"ai_api_key"`
	AiModel  string `json:"ai_model"`
	AiPrompt string `json:"ai_prompt"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	FileStoragePath string `json:"file_storage_path"`
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
		"file_storage_path":    c.FileStoragePath,
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
	config.FileStoragePath = settings["file_storage_path"]

	// 布尔字段
	config.OpenBlog, _ = strconv.ParseBool(settings["open_blog"])
	config.OpenComment, _ = strconv.ParseBool(settings["open_comment"])
	config.CommentEmailNotify, _ = strconv.ParseBool(settings["comment_email_notify"])

	// 整数字段
	config.SmtpPort, _ = strconv.Atoi(settings["smtp_port"])

	return config
}

// GetBlogConfig 获取博客基本配置
func (c *SystemConfig) GetBlogConfig() *BlogConfig {
	return &BlogConfig{
		BlogTitle:    c.BlogTitle,
		Signature:    c.Signature,
		Avatar:       c.Avatar,
		GithubLink:   c.GithubLink,
		BilibiliLink: c.BilibiliLink,
		OpenBlog:     c.OpenBlog,
		OpenComment:  c.OpenComment,
	}
}

// GetEmailConfig 获取邮件配置
func (c *SystemConfig) GetEmailConfig() *EmailConfig {
	return &EmailConfig{
		CommentEmailNotify: c.CommentEmailNotify,
		SmtpHost:           c.SmtpHost,
		SmtpPort:           c.SmtpPort,
		SmtpUser:           c.SmtpUser,
		SmtpPass:           c.SmtpPass,
		SmtpSender:         c.SmtpSender,
	}
}

// GetAIConfig 获取AI配置
func (c *SystemConfig) GetAIConfig() *AIConfig {
	return &AIConfig{
		AiApiURL: c.AiApiURL,
		AiApiKey: c.AiApiKey,
		AiModel:  c.AiModel,
		AiPrompt: c.AiPrompt,
	}
}

// GetStorageConfig 获取存储配置
func (c *SystemConfig) GetStorageConfig() *StorageConfig {
	return &StorageConfig{
		FileStoragePath: c.FileStoragePath,
	}
}

// UpdateBlogConfig 更新博客基本配置
func (c *SystemConfig) UpdateBlogConfig(blogConfig *BlogConfig) {
	c.BlogTitle = blogConfig.BlogTitle
	c.Signature = blogConfig.Signature
	c.Avatar = blogConfig.Avatar
	c.GithubLink = blogConfig.GithubLink
	c.BilibiliLink = blogConfig.BilibiliLink
	c.OpenBlog = blogConfig.OpenBlog
	c.OpenComment = blogConfig.OpenComment
}

// UpdateEmailConfig 更新邮件配置
func (c *SystemConfig) UpdateEmailConfig(emailConfig *EmailConfig) {
	c.CommentEmailNotify = emailConfig.CommentEmailNotify
	c.SmtpHost = emailConfig.SmtpHost
	c.SmtpPort = emailConfig.SmtpPort
	c.SmtpUser = emailConfig.SmtpUser
	c.SmtpPass = emailConfig.SmtpPass
	c.SmtpSender = emailConfig.SmtpSender
}

// UpdateAIConfig 更新AI配置
func (c *SystemConfig) UpdateAIConfig(aiConfig *AIConfig) {
	c.AiApiURL = aiConfig.AiApiURL
	c.AiApiKey = aiConfig.AiApiKey
	c.AiModel = aiConfig.AiModel
	c.AiPrompt = aiConfig.AiPrompt
}

// UpdateStorageConfig 更新存储配置
func (c *SystemConfig) UpdateStorageConfig(storageConfig *StorageConfig) {
	c.FileStoragePath = storageConfig.FileStoragePath
}

// BeforeSave 钩子，用于在保存前处理数据，例如加密敏感信息
func (s *SystemSetting) BeforeSave(*gorm.DB) (err error) {
	// 可以在这里添加对敏感信息的加密处理
	return
}

// BeforeDelete 钩子，用于在删除前处理数据
func (s *SystemSetting) BeforeDelete(*gorm.DB) (err error) {
	// 可以在这里添加删除前的逻辑
	return
}
