package system

import "strconv"

const (
	ConfigTypeBlog    = "blog"
	ConfigTypeEmail   = "email"
	ConfigTypeAI      = "ai"
	ConfigTypeStorage = "storage"
)

const (
	SettingKeyBlogTitle           = "blog_title"
	SettingKeySignature           = "signature"
	SettingKeyAvatar              = "avatar"
	SettingKeyGithubLink          = "github_link"
	SettingKeyBilibiliLink        = "bilibili_link"
	SettingKeyOpenBlog            = "open_blog"
	SettingKeyOpenComment         = "open_comment"
	SettingKeyCommentEmailNotify  = "comment_email_notify"
	SettingKeySmtpHost            = "smtp_host"
	SettingKeySmtpPort            = "smtp_port"
	SettingKeySmtpUser            = "smtp_user"
	SettingKeySmtpPass            = "smtp_pass"
	SettingKeySmtpSender          = "smtp_sender"
	SettingKeyAIAPIURL            = "ai_api_url"
	SettingKeyAIAPIKey            = "ai_api_key"
	SettingKeyAIModel             = "ai_model"
	SettingKeyAIPromptGetTags     = "ai_prompt_get_tags"
	SettingKeyAIPromptGetAbstract = "ai_prompt_get_abstract"
	SettingKeyFileStoragePath     = "file_storage_path"
	SettingKeyWebDAVChunkSize     = "webdav_chunk_size"
)

type Setting struct {
	ID           uint   `gorm:"primaryKey"`
	SettingKey   string `gorm:"unique;not null"`
	SettingValue string `gorm:"not null"`
	ConfigType   string `gorm:"not null;default:'blog'"`
}

func (Setting) TableName() string { return "system_settings" }

type Config struct {
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
	AIAPIURL           string `json:"ai_api_url"`
	AIAPIKey           string `json:"ai_api_key"`
	AIModel            string `json:"ai_model"`
	FileStoragePath    string `json:"file_storage_path"`
	WebDAVChunkSize    int    `json:"webdav_chunk_size"`
}

type BlogConfig struct {
	BlogTitle    string `json:"blog_title"`
	Signature    string `json:"signature"`
	Avatar       string `json:"avatar"`
	GithubLink   string `json:"github_link"`
	BilibiliLink string `json:"bilibili_link"`
	OpenBlog     bool   `json:"open_blog"`
	OpenComment  bool   `json:"open_comment"`
}

type EmailConfig struct {
	CommentEmailNotify bool   `json:"comment_email_notify"`
	SmtpHost           string `json:"smtp_host"`
	SmtpPort           int    `json:"smtp_port"`
	SmtpUser           string `json:"smtp_user"`
	SmtpPass           string `json:"smtp_pass"`
	SmtpSender         string `json:"smtp_sender"`
}

type AIConfig struct {
	APIURL string `json:"ai_api_url"`
	APIKey string `json:"ai_api_key"`
	Model  string `json:"ai_model"`
}

type StorageConfig struct {
	FileStoragePath string `json:"file_storage_path"`
	WebDAVChunkSize int    `json:"webdav_chunk_size"`
}

func configFrom(values map[string]string) Config {
	boolValue := func(key string) bool { value, _ := strconv.ParseBool(values[key]); return value }
	intValue := func(key string) int { value, _ := strconv.Atoi(values[key]); return value }
	chunkSize := intValue(SettingKeyWebDAVChunkSize)
	if chunkSize <= 0 {
		chunkSize = 5120
	}
	return Config{
		BlogTitle: values[SettingKeyBlogTitle], Signature: values[SettingKeySignature], Avatar: values[SettingKeyAvatar],
		GithubLink: values[SettingKeyGithubLink], BilibiliLink: values[SettingKeyBilibiliLink],
		OpenBlog: boolValue(SettingKeyOpenBlog), OpenComment: boolValue(SettingKeyOpenComment),
		CommentEmailNotify: boolValue(SettingKeyCommentEmailNotify), SmtpHost: values[SettingKeySmtpHost],
		SmtpPort: intValue(SettingKeySmtpPort), SmtpUser: values[SettingKeySmtpUser], SmtpPass: values[SettingKeySmtpPass],
		SmtpSender: values[SettingKeySmtpSender], AIAPIURL: values[SettingKeyAIAPIURL], AIAPIKey: values[SettingKeyAIAPIKey],
		AIModel: values[SettingKeyAIModel], FileStoragePath: values[SettingKeyFileStoragePath], WebDAVChunkSize: chunkSize,
	}
}

func (c Config) values() map[string]string {
	return map[string]string{
		SettingKeyBlogTitle: c.BlogTitle, SettingKeySignature: c.Signature, SettingKeyAvatar: c.Avatar,
		SettingKeyGithubLink: c.GithubLink, SettingKeyBilibiliLink: c.BilibiliLink,
		SettingKeyOpenBlog: strconv.FormatBool(c.OpenBlog), SettingKeyOpenComment: strconv.FormatBool(c.OpenComment),
		SettingKeyCommentEmailNotify: strconv.FormatBool(c.CommentEmailNotify), SettingKeySmtpHost: c.SmtpHost,
		SettingKeySmtpPort: strconv.Itoa(c.SmtpPort), SettingKeySmtpUser: c.SmtpUser, SettingKeySmtpPass: c.SmtpPass,
		SettingKeySmtpSender: c.SmtpSender, SettingKeyAIAPIURL: c.AIAPIURL, SettingKeyAIAPIKey: c.AIAPIKey,
		SettingKeyAIModel: c.AIModel, SettingKeyFileStoragePath: c.FileStoragePath,
		SettingKeyWebDAVChunkSize: strconv.Itoa(c.WebDAVChunkSize),
	}
}
