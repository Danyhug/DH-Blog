package share

import (
	"time"

	"dh-blog/internal/model"

	"gorm.io/gorm"
)

// Share is a persisted file-share link.
type Share struct {
	ID               int            `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID          string         `gorm:"type:varchar(32);uniqueIndex;not null" json:"share_id"`
	FileKey          string         `gorm:"type:text;not null" json:"file_key"`
	Password         string         `gorm:"type:varchar(64)" json:"password,omitempty"`
	ExpireAt         *time.Time     `json:"expire_at,omitempty"`
	MaxDownloadCount *int           `json:"max_download_count,omitempty"`
	ViewCount        int64          `gorm:"default:0;not null" json:"view_count"`
	DownloadCount    int64          `gorm:"default:0;not null" json:"download_count"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt        model.JSONTime `json:"create_time"`
	UpdatedAt        model.JSONTime `json:"update_time"`
}

func (Share) TableName() string {
	return "shares"
}

func (s *Share) IsExpired() bool {
	return s.ExpireAt != nil && time.Now().After(*s.ExpireAt)
}

func (s *Share) IsDownloadLimitReached() bool {
	return s.MaxDownloadCount != nil && s.DownloadCount >= int64(*s.MaxDownloadCount)
}

func (s *Share) HasPassword() bool {
	return s.Password != ""
}

// ShareAccessLog records a view or download of a share link.
type ShareAccessLog struct {
	ID         int            `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID    string         `gorm:"type:varchar(32);index;not null" json:"share_id"`
	ActionType string         `gorm:"type:varchar(16);not null" json:"action_type"`
	IP         string         `gorm:"type:varchar(64)" json:"ip"`
	UserAgent  string         `gorm:"type:text" json:"user_agent,omitempty"`
	Referer    string         `gorm:"type:text" json:"referer,omitempty"`
	CreatedAt  model.JSONTime `json:"create_time"`
}

func (ShareAccessLog) TableName() string {
	return "share_access_logs"
}

const (
	ShareActionView     = "view"
	ShareActionDownload = "download"
)
