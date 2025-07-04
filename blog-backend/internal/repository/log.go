package repository

import (
	"fmt"

	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type LogRepository struct {
	DB *gorm.DB
}

func NewLogRepository(db *gorm.DB) *LogRepository {
	return &LogRepository{DB: db}
}

// CreateAccessLog 创建访问日志
func (r *LogRepository) CreateAccessLog(log *model.AccessLog) error {
	err := r.DB.Create(log).Error
	if err != nil {
		return fmt.Errorf("创建访问日志失败: %w", err)
	}
	return nil
}

// GetAccessLogs 获取访问日志列表（带分页和日期筛选）
func (r *LogRepository) GetAccessLogs(page, pageSize int, startDate, endDate string) ([]model.AccessLog, int64, error) {
	var logs []model.AccessLog
	var total int64

	offset := (page - 1) * pageSize
	query := r.DB.Model(&model.AccessLog{})

	// 如果提供了日期范围，则添加查询条件
	if startDate != "" && endDate != "" {
		// 确保结束日期包含当天所有时间
		endDate = endDate + " 23:59:59"
		query = query.Where("created_at BETWEEN ? AND ?", startDate, endDate)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询访问日志总数失败: %w", err)
	}

	// 查询日志列表
	if err := query.Offset(offset).Limit(pageSize).Order("created_at desc").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询访问日志列表失败: %w", err)
	}

	return logs, total, nil
}

// CreateOrUpdateIPStat 创建或更新 IP 统计
func (r *LogRepository) CreateOrUpdateIPStat(ipStat *model.IPStat) error {
	var existingStat model.IPStat
	// 尝试查找现有记录
	result := r.DB.Where("ip_address = ?", ipStat.IPAddress).First(&existingStat)

	if result.Error == gorm.ErrRecordNotFound {
		// 记录不存在，创建新记录
		ipStat.AccessCount = 1
		return r.DB.Create(ipStat).Error
	} else if result.Error != nil {
		// 其他错误
		return fmt.Errorf("查询 IP 统计失败: %w", result.Error)
	} else {
		// 记录存在，更新访问次数
		existingStat.AccessCount++
		// 更新城市信息，如果新的不为空
		if ipStat.City != "" {
			existingStat.City = ipStat.City
		}
		return r.DB.Save(&existingStat).Error
	}
}

// GetIPStats 获取 IP 统计列表（带分页）
func (r *LogRepository) GetIPStats(page, pageSize int) ([]model.IPStat, int64, error) {
	var stats []model.IPStat
	var total int64

	offset := (page - 1) * pageSize

	if err := r.DB.Model(&model.IPStat{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询 IP 统计总数失败: %w", err)
	}

	if err := r.DB.Offset(offset).Limit(pageSize).Order("access_count desc").Find(&stats).Error; err != nil {
		return nil, 0, fmt.Errorf("查询 IP 统计列表失败: %w", err)
	}

	return stats, total, nil
}

// UpdateIPBanStatus 更新 IP 封禁状态
func (r *LogRepository) UpdateIPBanStatus(ipAddress string, status bool) error {
	err := r.DB.Model(&model.IPStat{}).Where("ip_address = ?", ipAddress).Update("ban_status", status).Error
	if err != nil {
		return fmt.Errorf("更新 IP 封禁状态失败: %w", err)
	}
	return nil
}
