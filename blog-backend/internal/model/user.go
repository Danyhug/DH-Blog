package model

import (
	"gorm.io/gorm"
)

// User 表示数据库中的用户实体。
type User struct {
	gorm.Model        // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	Username   string `gorm:"column:username;not null;uniqueIndex" json:"username"` // 用户名，唯一
	Password   string `gorm:"column:password;not null" json:"-"`                    // 密码，不暴露在 JSON 中
}

// TableName 指定 User 模型对应的表名。
func (User) TableName() string {
	return "users" // 显式设置表名为 'users'
}
