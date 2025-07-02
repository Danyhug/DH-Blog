package model

import (
	"gorm.io/gorm"
)

// IPStat 表示数据库中的 IP 统计实体。
type IPStat struct {
	gorm.Model         // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	IPAddress   string `gorm:"column:ip_address;uniqueIndex" json:"ip_address"`   // IP 地址，唯一
	City        string `gorm:"column:city" json:"city"`                           // 城市
	AccessCount int    `gorm:"column:access_count;default:0" json:"access_count"` // 访问次数
	BannedCount int    `gorm:"column:banned_count;default:0" json:"banned_count"` // 封禁次数
	BanStatus   bool   `gorm:"column:ban_status;default:false" json:"ban_status"` // 封禁状态
}

// TableName 指定 IPStat 模型对应的表名。
func (IPStat) TableName() string {
	return "ip_stats" // 显式设置表名为 'ip_stats'
}
