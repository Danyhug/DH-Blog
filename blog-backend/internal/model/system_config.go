package model

type SystemConfig struct {
	BaseModel
	// 站点设置
	BlogTitle    string `gorm:"type:varchar(255);comment:博客标题" json:"blog_title"`
	Signature    string `gorm:"type:varchar(255);comment:个人签名" json:"signature"`
	Avatar       string `gorm:"type:varchar(255);comment:个人头像" json:"avatar"`
	GithubLink   string `gorm:"type:varchar(255);comment:github链接" json:"github_link"`
	BilibiliLink string `gorm:"type:varchar(255);comment:bilibili链接" json:"bilibili_link"`

	// 站点功能设置
	OpenBlog           bool `gorm:"type:bool;comment:开放博客" json:"open_blog"`
	OpenComment        bool `gorm:"type:bool;comment:开放评论" json:"open_comment"`
	CommentEmailNotify bool `gorm:"type:bool;comment:评论邮件通知" json:"comment_email_notify"`

	// 邮箱设置
	SmtpHost   string `gorm:"type:varchar(255);comment:smtp主机" json:"smtp_host"`
	SmtpPort   int    `gorm:"type:int;comment:smtp端口" json:"smtp_port"`
	SmtpUser   string `gorm:"type:varchar(255);comment:smtp用户" json:"smtp_user"`
	SmtpPass   string `gorm:"type:varchar(255);comment:smtp密码" json:"smtp_pass"`
	SmtpSender string `gorm:"type:varchar(255);comment:smtp发送者" json:"smtp_sender"`

	// AI设置
	AiPrompt string `gorm:"type:text;comment:AI提取文章标签的提示词" json:"ai_prompt"`
}

func (s SystemConfig) TableName() string {
	return "system_config"
}
