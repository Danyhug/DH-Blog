package model

import (
	"gorm.io/gorm"
)

// AccessLog 表示数据库中的访问日志实体。
type AccessLog struct {
	gorm.Model        // 嵌入 GORM 的 Model，默认字段包括 ID、CreatedAt、UpdatedAt、DeletedAt
	IPAddress  string `gorm:"column:ip_address;not null" json:"ip_address"` // IP 地址
	UserAgent  string `gorm:"column:user_agent" json:"user_agent"`          // 用户代理
	RequestURL string `gorm:"column:request_url" json:"request_url"`        // 请求 URL
}

// TableName 指定 AccessLog 模型对应的表名。
func (AccessLog) TableName() string {
	return "access_logs" // 显式设置表名为 'access_logs'
}
