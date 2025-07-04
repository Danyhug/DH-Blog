package model

// AccessLog 对应于数据库中的 `access_logs` 表
type AccessLog struct {
	BaseModel  `gorm:"embedded"`
	IPAddress  string `gorm:"column:ip_address;not null" json:"ipAddress"` // IP 地址
	UserAgent  string `gorm:"column:user_agent" json:"userAgent"`          // 用户代理
	RequestURL string `gorm:"column:request_url" json:"requestUrl"`        // 请求 URL
	City       string `gorm:"-" json:"city,omitempty"`                     // 城市 (不映射到数据库，仅用于数据传输)
}

// TableName 指定 GORM 使用的表名
func (AccessLog) TableName() string {
	return "access_logs"
}
