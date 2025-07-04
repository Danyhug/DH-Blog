package model

// IPStat 对应于数据库中的 `ip_stats` 表
type IPStat struct {
	BaseModel   `gorm:"embedded"`
	IPAddress   string `gorm:"column:ip_address;uniqueIndex" json:"ipAddress"`   // IP 地址，唯一
	City        string `gorm:"column:city" json:"city"`                          // 城市
	AccessCount int    `gorm:"column:access_count;default:0" json:"accessCount"` // 访问次数
	BannedCount int    `gorm:"column:banned_count;default:0" json:"bannedCount"` // 封禁次数
	BanStatus   bool   `gorm:"column:ban_status;default:false" json:"banStatus"` // 封禁状态
}

// TableName 指定 GORM 使用的表名
func (IPStat) TableName() string {
	return "ip_stats"
}
