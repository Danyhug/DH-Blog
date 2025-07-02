package model

import (
	"gorm.io/gorm"
)

// DailyStat 表示数据库中的每日统计实体。
type DailyStat struct {
	gorm.Model          // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	Date         string `gorm:"column:date;uniqueIndex" json:"date"`                 // 日期，唯一
	VisitCount   int    `gorm:"column:visit_count;default:0" json:"visit_count"`     // 访问量
	ArticleCount int    `gorm:"column:article_count;default:0" json:"article_count"` // 文章数量
	CommentCount int    `gorm:"column:comment_count;default:0" json:"comment_count"` // 评论数量
	TagCount     int    `gorm:"column:tag_count;default:0" json:"tag_count"`         // 标签数量
}

// TableName 指定 DailyStat 模型对应的表名。
func (DailyStat) TableName() string {
	return "daily_stats" // 显式设置表名为 'daily_stats'
}
