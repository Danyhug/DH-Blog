package model

import "time"

// IPBlacklist 对应于数据库中的 `ip_blacklist` 表
type IPBlacklist struct {
	BaseModel  `gorm:"embedded"`
	IPAddress  string    `gorm:"column:ip_address;index" json:"ipAddress"` // IP 地址，可重复
	BanReason  string    `gorm:"column:ban_reason" json:"banReason"`       // 封禁原因
	ExpireTime time.Time `gorm:"column:expire_time" json:"expireTime"`     // 封禁过期时间，NULL表示永久封禁
}

// TableName 指定 GORM 使用的表名
func (IPBlacklist) TableName() string {
	return "ip_blacklist"
}
