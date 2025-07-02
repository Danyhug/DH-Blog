package model

import (
	"gorm.io/gorm"
)

// Article 表示数据库中的文章实体。
type Article struct {
	gorm.Model          // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	Title        string `gorm:"column:title;not null" json:"title"`              // 文章标题
	Content      string `gorm:"column:content;not null" json:"content"`          // 文章内容
	CategoryID   int    `gorm:"column:category_id" json:"category_id"`           // 分类 ID，关联 categories 表
	Views        int    `gorm:"column:views;default:0" json:"views"`             // 浏览次数
	WordNum      int    `gorm:"column:word_num" json:"word_num"`                 // 字数统计
	ThumbnailURL string `gorm:"column:thumbnail_url" json:"thumbnail_url"`       // 缩略图 URL
	IsLocked     bool   `gorm:"column:is_locked;default:false" json:"is_locked"` // 是否锁定
	LockPassword string `gorm:"column:lock_password" json:"lock_password"`       // 锁定密码
}

// TableName 指定 Article 模型对应的表名。
func (Article) TableName() string {
	return "articles" // 显式设置表名为 'articles'
}
