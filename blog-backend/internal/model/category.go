package model

import (
	"gorm.io/gorm"
)

// Category 表示数据库中的分类实体。
type Category struct {
	gorm.Model        // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	Name       string `gorm:"column:name;not null;uniqueIndex" json:"name"` // 分类名称，唯一
	Slug       string `gorm:"column:slug;not null;uniqueIndex" json:"slug"` // URL 友好的别名，唯一
}

// TableName 指定 Category 模型对应的表名。
func (Category) TableName() string {
	return "categories" // 显式设置表名为 'categories'
}
