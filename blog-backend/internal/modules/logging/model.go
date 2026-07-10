package logging

import (
	"time"

	"dh-blog/internal/model"
)

// AccessLog records one request handled by the blog API.
type AccessLog struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	IPAddress    string    `gorm:"column:ip_address;not null" json:"ipAddress"`
	AccessDate   time.Time `gorm:"column:access_date;index" json:"accessDate"`
	UserAgent    string    `gorm:"column:user_agent" json:"userAgent"`
	RequestURL   string    `gorm:"column:request_url" json:"requestUrl"`
	City         string    `gorm:"column:city" json:"city"`
	ResourceType string    `gorm:"column:resource_type" json:"resourceType"`
}

// TableName keeps the existing access log table compatible.
func (AccessLog) TableName() string {
	return "access_logs"
}

// IPBlacklist records an IP ban. Soft-deleted rows remain available for ban
// history statistics.
type IPBlacklist struct {
	model.BaseModel `gorm:"embedded"`
	IPAddress       string    `gorm:"column:ip_address;index" json:"ipAddress"`
	BanReason       string    `gorm:"column:ban_reason" json:"banReason"`
	ExpireTime      time.Time `gorm:"column:expire_time" json:"expireTime"`
}

// TableName keeps the existing blacklist table compatible.
func (IPBlacklist) TableName() string {
	return "ip_blacklist"
}
