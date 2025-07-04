package model

// Comment 对应于数据库中的 `comments` 表
type Comment struct {
	BaseModel `gorm:"embedded"`
	ArticleID uint       `gorm:"column:article_id;not null" json:"articleId"`   // 文章 ID
	Author    string     `gorm:"column:author;not null" json:"author"`          // 作者名称
	Email     string     `gorm:"column:email;not null" json:"email"`            // 邮箱
	Content   string     `gorm:"column:content;not null" json:"content"`        // 内容
	IsPublic  bool       `gorm:"column:is_public;default:true" json:"isPublic"` // 是否公开
	ParentID  *uint      `gorm:"column:parent_id" json:"parentId"`              // 父评论 ID (可为空)
	UA        string     `gorm:"column:ua;not null" json:"ua"`                  // 用户代理
	IsAdmin   bool       `gorm:"column:is_admin;default:false" json:"isAdmin"`  // 是否管理员
	Children  []*Comment `gorm:"-" json:"children,omitempty"`                   // 子评论 (不映射到数据库)
}

// TableName 指定 GORM 使用的表名
func (Comment) TableName() string {
	return "comments"
}
