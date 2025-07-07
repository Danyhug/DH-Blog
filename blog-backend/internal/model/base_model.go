package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 包含了所有模型通用的字段和 GORM 钩子
type BaseModel struct {
	ID        int            `gorm:"primarykey" json:"id"`
	CreatedAt JSONTime       `json:"createTime"`
	UpdatedAt JSONTime       `json:"updateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// BeforeCreate 是 GORM 的 BeforeCreate 钩子，用于在创建记录前设置创建时间和更新时间
func (base *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	now := JSONTime{time.Now()}
	base.CreatedAt = now
	base.UpdatedAt = now
	return
}

// BeforeUpdate 是 GORM 的 BeforeUpdate 钩子，用于在更新记录前设置更新时间
func (base *BaseModel) BeforeUpdate(tx *gorm.DB) (err error) {
	base.UpdatedAt = JSONTime{time.Now()}
	return
}
