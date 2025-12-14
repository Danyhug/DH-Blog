package model

import (
	"time"

	"gorm.io/gorm"
)

// Share 分享记录
type Share struct {
	ID               int            `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID          string         `gorm:"type:varchar(32);uniqueIndex;not null" json:"share_id"`  // 分享短链ID
	FileKey          string         `gorm:"type:text;not null" json:"file_key"`                     // 文件标识（文件ID）
	Password         string         `gorm:"type:varchar(64)" json:"password,omitempty"`             // 访问密码（可选）
	ExpireAt         *time.Time     `json:"expire_at,omitempty"`                                    // 过期时间（可选）
	MaxDownloadCount *int           `json:"max_download_count,omitempty"`                           // 最大下载次数（可选）
	ViewCount        int64          `gorm:"default:0;not null" json:"view_count"`                   // 查看次数
	DownloadCount    int64          `gorm:"default:0;not null" json:"download_count"`               // 下载次数
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt        JSONTime       `json:"create_time"`
	UpdatedAt        JSONTime       `json:"update_time"`
}

// TableName 指定表名
func (Share) TableName() string {
	return "shares"
}

// IsExpired 检查分享是否已过期
func (s *Share) IsExpired() bool {
	if s.ExpireAt == nil {
		return false
	}
	return time.Now().After(*s.ExpireAt)
}

// IsDownloadLimitReached 检查是否达到下载次数限制
func (s *Share) IsDownloadLimitReached() bool {
	if s.MaxDownloadCount == nil {
		return false
	}
	return s.DownloadCount >= int64(*s.MaxDownloadCount)
}

// HasPassword 检查是否设置了密码
func (s *Share) HasPassword() bool {
	return s.Password != ""
}

// ShareAccessLog 分享访问日志
type ShareAccessLog struct {
	ID         int      `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID    string   `gorm:"type:varchar(32);index;not null" json:"share_id"` // 分享ID
	ActionType string   `gorm:"type:varchar(16);not null" json:"action_type"`    // view/download
	IP         string   `gorm:"type:varchar(64)" json:"ip"`
	UserAgent  string   `gorm:"type:text" json:"user_agent,omitempty"`
	Referer    string   `gorm:"type:text" json:"referer,omitempty"`
	CreatedAt  JSONTime `json:"create_time"`
}

// TableName 指定表名
func (ShareAccessLog) TableName() string {
	return "share_access_logs"
}

// ActionType 常量
const (
	ShareActionView     = "view"
	ShareActionDownload = "download"
)
