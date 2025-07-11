package model

import "time"

// AccessLog 对应于数据库中的 `access_logs` 表
type AccessLog struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	IPAddress    string    `gorm:"column:ip_address;not null" json:"ipAddress"` // IP 地址
	AccessDate   time.Time `gorm:"column:access_date;index" json:"accessDate"`  // 访问日期，用于按日统计
	UserAgent    string    `gorm:"column:user_agent" json:"userAgent"`          // 用户代理
	RequestURL   string    `gorm:"column:request_url" json:"requestUrl"`        // 请求 URL
	City         string    `gorm:"column:city" json:"city"`                     // 城市
	ResourceType string    `gorm:"column:resource_type" json:"resourceType"`    // 资源类型（article, tag, category等）
}

// TableName 指定 GORM 使用的表名
func (AccessLog) TableName() string {
	return "access_logs"
}

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
