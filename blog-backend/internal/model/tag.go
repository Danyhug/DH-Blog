package model

import (
	"gorm.io/gorm"
)

// Tag 表示数据库中的标签实体。
type Tag struct {
	gorm.Model        // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	Name       string `gorm:"column:name;not null;uniqueIndex" json:"name"` // 标签名，唯一
	Slug       string `gorm:"column:slug;not null;uniqueIndex" json:"slug"` // URL 友好的别名，唯一
}

// TableName 指定 Tag 模型对应的表名。
func (Tag) TableName() string {
	return "tags" // 显式设置表名为 'tags'
}
