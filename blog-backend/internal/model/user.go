package model

// User 对应于数据库中的 `users` 表
type User struct {
	BaseModel `gorm:"embedded"`
	Username  string `gorm:"column:username;not null;uniqueIndex" json:"username"` // 用户名，唯一
	Password  string `gorm:"column:password;not null" json:"password"`             // 密码，不暴露在 JSON 中
}

// TableName 指定 GORM 使用的表名
func (User) TableName() string {
	return "users"
}
