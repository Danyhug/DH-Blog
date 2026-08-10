package system

import "strconv"

const (
	ConfigTypeBlog    = "blog"
	ConfigTypeAI      = "ai"
	ConfigTypeStorage = "storage"
)

const (
	SettingKeyBlogTitle           = "blog_title"
	SettingKeySignature           = "signature"
	SettingKeyAvatar              = "avatar"
	SettingKeyGithubLink          = "github_link"
	SettingKeyBilibiliLink        = "bilibili_link"
	SettingKeyOpenComment         = "open_comment"
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
	BlogTitle       string `json:"blog_title"`
	Signature       string `json:"signature"`
	Avatar          string `json:"avatar"`
	GithubLink      string `json:"github_link"`
	BilibiliLink    string `json:"bilibili_link"`
	OpenComment     bool   `json:"open_comment"`
	AIAPIURL        string `json:"ai_api_url"`
	AIAPIKey        string `json:"ai_api_key"`
	AIModel         string `json:"ai_model"`
	FileStoragePath string `json:"file_storage_path"`
	WebDAVChunkSize int    `json:"webdav_chunk_size"`
}

// BlogConfig 同时用于后台编辑和前台公开展示，字段均可公开。
type BlogConfig struct {
	BlogTitle    string `json:"blog_title"`
	Signature    string `json:"signature"`
	Avatar       string `json:"avatar"`
	GithubLink   string `json:"github_link"`
	BilibiliLink string `json:"bilibili_link"`
	OpenComment  bool   `json:"open_comment"`
}

type AIConfig struct {
	APIURL string `json:"ai_api_url"`
	APIKey string `json:"ai_api_key"`
	Model  string `json:"ai_model"`
}

// AIPrompt 是可在后台编辑的一条提示词。Key 对应 system_settings 中的键。
type AIPrompt struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
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
		OpenComment: boolValue(SettingKeyOpenComment),
		AIAPIURL:    values[SettingKeyAIAPIURL], AIAPIKey: values[SettingKeyAIAPIKey],
		AIModel: values[SettingKeyAIModel], FileStoragePath: values[SettingKeyFileStoragePath], WebDAVChunkSize: chunkSize,
	}
}
