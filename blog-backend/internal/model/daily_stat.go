package model

import (
	"time"
)

// DailyStat 对应于数据库中的 `daily_stats` 表
type DailyStat struct {
	BaseModel    `gorm:"embedded"`
	Date         time.Time `gorm:"column:date;uniqueIndex;type:date" json:"date"`      // 日期，唯一
	VisitCount   int       `gorm:"column:visit_count;default:0" json:"visitCount"`     // 访问量
	ArticleCount int       `gorm:"column:article_count;default:0" json:"articleCount"` // 文章数量
	CommentCount int       `gorm:"column:comment_count;default:0" json:"commentCount"` // 评论数量
	TagCount     int       `gorm:"column:tag_count;default:0" json:"tagCount"`         // 标签数量
}

// TableName 指定 GORM 使用的表名
func (DailyStat) TableName() string {
	return "daily_stats"
}
