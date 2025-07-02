package model

import (
	"gorm.io/gorm"
)

// Comment 表示数据库中的评论实体。
type Comment struct {
	gorm.Model        // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	ArticleID  int    `gorm:"column:article_id;not null" json:"article_id"`   // 文章 ID
	Author     string `gorm:"column:author;not null" json:"author"`           // 作者名称
	Email      string `gorm:"column:email;not null" json:"email"`             // 邮箱
	Content    string `gorm:"column:content;not null" json:"content"`         // 内容
	IsPublic   bool   `gorm:"column:is_public;default:true" json:"is_public"` // 是否公开
	ParentID   *int   `gorm:"column:parent_id" json:"parent_id"`              // 父评论 ID
	UA         string `gorm:"column:ua;not null" json:"ua"`                   // 用户代理
	IsAdmin    bool   `gorm:"column:is_admin;default:false" json:"is_admin"`  // 是否管理员
}

// TableName 指定 Comment 模型对应的表名。
func (Comment) TableName() string {
	return "comments" // 显式设置表名为 'comments'
}
