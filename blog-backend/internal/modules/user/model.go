package user

import "dh-blog/internal/model"

// User is the administrator account persisted in the users table.
type User struct {
	model.BaseModel `gorm:"embedded"`
	Username        string `gorm:"column:username;not null;uniqueIndex" json:"username"`
	Password        string `gorm:"column:password;not null" json:"password"`
}

func (User) TableName() string {
	return "users"
}
