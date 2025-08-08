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

// 博客基本配置键名常量
const (
	SettingKeyBlogTitle    = "blog_title"    // 博客标题
	SettingKeySignature    = "signature"     // 个性签名
	SettingKeyAvatar       = "avatar"        // 头像
	SettingKeyGithubLink   = "github_link"   // GitHub链接
	SettingKeyBilibiliLink = "bilibili_link" // B站链接
	SettingKeyOpenBlog     = "open_blog"     // 是否开放博客
	SettingKeyOpenComment  = "open_comment"  // 是否开放评论
)

// 邮件配置键名常量
const (
	SettingKeyCommentEmailNotify = "comment_email_notify" // 评论邮件通知
	SettingKeySmtpHost           = "smtp_host"            // SMTP主机
	SettingKeySmtpPort           = "smtp_port"            // SMTP端口
	SettingKeySmtpUser           = "smtp_user"            // SMTP用户名
	SettingKeySmtpPass           = "smtp_pass"            // SMTP密码
	SettingKeySmtpSender         = "smtp_sender"          // SMTP发件人
)

// AI配置键名常量
const (
	SettingKeyAiApiURL            = "ai_api_url"             // AI API URL
	SettingKeyAiApiKey            = "ai_api_key"             // AI API Key
	SettingKeyAiModel             = "ai_model"               // AI模型
	SettingKeyAiPromptGetTags     = "ai_prompt_get_tags"     // 获取标签的提示词
	SettingKeyAiPromptGetAbstract = "ai_prompt_get_abstract" // 获取摘要的提示词
)

// 存储配置键名常量
const (
	SettingKeyFileStoragePath = "file_storage_path" // 文件存储路径
)

// WebDAV配置键名常量
const (
	SettingKeyWebdavChunkSize = "webdav_chunk_size" // WebDAV分片大小(KB)
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

	// 存储配置
	FileStoragePath string `json:"file_storage_path"` // 文件存储路径
	// WebDAV配置
	WebdavChunkSize int `json:"webdav_chunk_size"` // WebDAV分片大小(KB)
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
}

// StorageConfig 存储配置
type StorageConfig struct {
	FileStoragePath string `json:"file_storage_path"`
	WebdavChunkSize int    `json:"webdav_chunk_size"`
}

// ToSettingsMap 将SystemConfig转换为map[string]string
func (c *SystemConfig) ToSettingsMap() map[string]string {
	return map[string]string{
		SettingKeyBlogTitle:          c.BlogTitle,
		SettingKeySignature:          c.Signature,
		SettingKeyAvatar:             c.Avatar,
		SettingKeyGithubLink:         c.GithubLink,
		SettingKeyBilibiliLink:       c.BilibiliLink,
		SettingKeyOpenBlog:           strconv.FormatBool(c.OpenBlog),
		SettingKeyOpenComment:        strconv.FormatBool(c.OpenComment),
		SettingKeyCommentEmailNotify: strconv.FormatBool(c.CommentEmailNotify),
		SettingKeySmtpHost:           c.SmtpHost,
		SettingKeySmtpPort:           strconv.Itoa(c.SmtpPort),
		SettingKeySmtpUser:           c.SmtpUser,
		SettingKeySmtpPass:           c.SmtpPass,
		SettingKeySmtpSender:         c.SmtpSender,
		SettingKeyAiApiURL:           c.AiApiURL,
		SettingKeyAiApiKey:           c.AiApiKey,
		SettingKeyAiModel:            c.AiModel,
		SettingKeyFileStoragePath:    c.FileStoragePath,
		SettingKeyWebdavChunkSize:    strconv.Itoa(c.WebdavChunkSize),
	}
}

// FromSettingsMap 从map[string]string创建SystemConfig
func FromSettingsMap(settings map[string]string) *SystemConfig {
	config := &SystemConfig{}

	// 字符串字段
	config.BlogTitle = settings[SettingKeyBlogTitle]
	config.Signature = settings[SettingKeySignature]
	config.Avatar = settings[SettingKeyAvatar]
	config.GithubLink = settings[SettingKeyGithubLink]
	config.BilibiliLink = settings[SettingKeyBilibiliLink]
	config.SmtpHost = settings[SettingKeySmtpHost]
	config.SmtpUser = settings[SettingKeySmtpUser]
	config.SmtpPass = settings[SettingKeySmtpPass]
	config.SmtpSender = settings[SettingKeySmtpSender]
	config.AiApiURL = settings[SettingKeyAiApiURL]
	config.AiApiKey = settings[SettingKeyAiApiKey]
	config.AiModel = settings[SettingKeyAiModel]
	config.FileStoragePath = settings[SettingKeyFileStoragePath]

	// 布尔字段
	config.OpenBlog, _ = strconv.ParseBool(settings[SettingKeyOpenBlog])
	config.OpenComment, _ = strconv.ParseBool(settings[SettingKeyOpenComment])
	config.CommentEmailNotify, _ = strconv.ParseBool(settings[SettingKeyCommentEmailNotify])

	// 整数字段
	config.SmtpPort, _ = strconv.Atoi(settings[SettingKeySmtpPort])
	if chunkSize, ok := settings[SettingKeyWebdavChunkSize]; ok && chunkSize != "" {
		config.WebdavChunkSize, _ = strconv.Atoi(chunkSize)
	} else {
		config.WebdavChunkSize = 5120 // 默认5MB (5120KB)
	}

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
	}
}

// GetStorageConfig 获取存储配置
func (c *SystemConfig) GetStorageConfig() *StorageConfig {
	return &StorageConfig{
		FileStoragePath: c.FileStoragePath,
		WebdavChunkSize: c.WebdavChunkSize,
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
}

// UpdateStorageConfig 更新存储配置
func (c *SystemConfig) UpdateStorageConfig(storageConfig *StorageConfig) {
	c.FileStoragePath = storageConfig.FileStoragePath
	c.WebdavChunkSize = storageConfig.WebdavChunkSize
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
