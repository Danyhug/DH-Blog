package model

import (
	"time"

	"gorm.io/gorm"
)

// CategoryDefaultTags 是分类和默认标签的关联表
type CategoryDefaultTags struct {
	CategoryID int            `gorm:"primaryKey"`
	TagID      int            `gorm:"primaryKey"`
	CreatedAt  JSONTime       `json:"createdAt"`
	UpdatedAt  JSONTime       `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate 是 GORM 的 BeforeCreate 钩子，用于在创建记录前设置创建时间和更新时间
func (cdt *CategoryDefaultTags) BeforeCreate(tx *gorm.DB) (err error) {
	now := JSONTime{time.Now()}
	cdt.CreatedAt = now
	cdt.UpdatedAt = now
	return
}

// BeforeUpdate 是 GORM 的 BeforeUpdate 钩子，用于在更新记录前设置更新时间
func (cdt *CategoryDefaultTags) BeforeUpdate(tx *gorm.DB) (err error) {
	cdt.UpdatedAt = JSONTime{time.Now()}
	return
}

// TableName 指定 GORM 使用的表名
func (CategoryDefaultTags) TableName() string {
	return "category_default_tags"
}
