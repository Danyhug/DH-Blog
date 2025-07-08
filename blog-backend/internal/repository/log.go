package repository

import (
	"dh-blog/internal/model"
	"gorm.io/gorm"
	"time"
)

// LogRepository 定义日志仓库
type LogRepository struct {
	db *gorm.DB
}

// NewLogRepository 创建新的日志仓库
func NewLogRepository(db *gorm.DB) *LogRepository {
	return &LogRepository{db: db}
}

// SaveAccessLog 保存访问日志
func (r *LogRepository) SaveAccessLog(log *model.AccessLog) error {
	return r.db.Create(log).Error
}

// GetVisitLogs 获取访问日志（带分页）
func (r *LogRepository) GetVisitLogs(page, pageSize int) ([]model.AccessLog, int64, error) {
	var logs []model.AccessLog
	var total int64

	r.db.Model(&model.AccessLog{}).Count(&total)
	err := r.db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&logs).Error
	return logs, total, err
}

// BanIP 将IP添加到黑名单
func (r *LogRepository) BanIP(ip, reason string, expireTime time.Time) error {
	blacklist := &model.IPBlacklist{
		IPAddress:  ip,
		BanReason:  reason,
		ExpireTime: expireTime,
	}
	return r.db.Create(blacklist).Error
}

// UnbanIP 从黑名单中移除IP
func (r *LogRepository) UnbanIP(ip string) error {
	return r.db.Where("ip_address = ?", ip).Delete(&model.IPBlacklist{}).Error
}

// IsIPBanned 检查IP是否在黑名单中
func (r *LogRepository) IsIPBanned(ip string) (bool, error) {
	var count int64
	err := r.db.Model(&model.IPBlacklist{}).
		Where("ip_address = ? AND (expire_time IS NULL OR expire_time > ?)", ip, time.Now()).
		Count(&count).Error
	return count > 0, err
}

// GetDailyVisitStats 获取每日访问统计
func (r *LogRepository) GetDailyVisitStats(startDate, endDate time.Time) ([]map[string]interface{}, error) {
	var stats []map[string]interface{}
	err := r.db.Model(&model.AccessLog{}).
		Select("strftime('%Y-%m-%d', access_date) as date, count(*) as visit_count").
		Where("access_date BETWEEN ? AND ?", startDate, endDate).
		Group("date").
		Order("date DESC").
		Find(&stats).Error
	return stats, err
}
