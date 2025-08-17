package repository

import (
	"fmt"
	"time"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 缓存相关常量
const (
	IPBlacklistCacheKeyPrefix = "ip:blacklist:"
	IPBlacklistCacheExpire    = time.Minute * 10 // 10分钟过期
)

// LogRepository 定义日志仓库
type LogRepository struct {
	db    *gorm.DB
	cache dhcache.Cache
}

// NewLogRepository 创建新的日志仓库
func NewLogRepository(db *gorm.DB, cache dhcache.Cache) *LogRepository {
	return &LogRepository{
		db:    db,
		cache: cache,
	}
}

// SaveAccessLog 保存访问日志
func (r *LogRepository) SaveAccessLog(log *model.AccessLog) error {
	return r.db.Create(log).Error
}

// SaveAccessLogAsync 异步保存访问日志（不阻塞主流程）
func (r *LogRepository) SaveAccessLogAsync(log *model.AccessLog) {
	go func() {
		if err := r.SaveAccessLog(log); err != nil {
			logrus.Errorf("异步保存访问日志失败: %v", err)
		}
	}()
}

// GetVisitLogs 获取访问日志（带分页）
func (r *LogRepository) GetVisitLogs(page, pageSize int) ([]model.AccessLog, int64, error) {
	var logs []model.AccessLog
	var total int64

	r.db.Model(&model.AccessLog{}).Count(&total)
	err := r.db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&logs).Error
	return logs, total, err
}

// GetIPVisitStats 获取按IP分组的访问统计和封禁次数
func (r *LogRepository) GetIPVisitStats(page, pageSize int, startDate, endDate time.Time) ([]map[string]interface{}, int64, error) {
	var result []map[string]interface{}
	var total int64

	// 子查询：获取每个IP的封禁次数（包括已删除的记录）
	// 使用Unscoped()来包含软删除的记录，这样能计算总封禁次数
	banSubQuery := r.db.Unscoped().Model(&model.IPBlacklist{}).
		Select("ip_address, COUNT(*) as banned_count").
		Group("ip_address")

	// 子查询：获取每个IP当前是否被封禁
	currentBanSubQuery := r.db.Model(&model.IPBlacklist{}).
		Select("ip_address, 1 as ban_status").
		Where("(expire_time IS NULL OR expire_time > ?)", time.Now())

	// 主查询：获取每个IP的访问统计
	query := r.db.Model(&model.AccessLog{}).
		Select("access_logs.ip_address as ipAddress, MAX(access_logs.city) as city, "+
			"COUNT(access_logs.id) as accessCount, "+
			"IFNULL(ban_stats.banned_count, 0) as bannedCount, "+
			"IFNULL(current_ban.ban_status, 0) as banStatus").
		Joins("LEFT JOIN (?) as ban_stats ON access_logs.ip_address = ban_stats.ip_address", banSubQuery).
		Joins("LEFT JOIN (?) as current_ban ON access_logs.ip_address = current_ban.ip_address", currentBanSubQuery).
		Where("access_logs.access_date BETWEEN ? AND ?", startDate, endDate).
		Group("access_logs.ip_address").
		Order("accessCount DESC")

	// 获取总记录数
	var countResult []map[string]interface{}
	err := query.Find(&countResult).Error
	if err != nil {
		return nil, 0, err
	}
	total = int64(len(countResult))

	// 分页查询
	err = query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&result).Error
	return result, total, err
}

// GetVisitStatistics 获取访问统计信息
func (r *LogRepository) GetVisitStatistics() (map[string]int64, error) {
	stats := make(map[string]int64)

	// 获取今日访问次数
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	var todayVisits int64
	err := r.db.Model(&model.AccessLog{}).
		Where("access_date BETWEEN ? AND ?", today, tomorrow).
		Count(&todayVisits).Error
	if err != nil {
		return nil, err
	}
	stats["todayVisits"] = todayVisits

	// 获取本周访问次数
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	var weekVisits int64
	err = r.db.Model(&model.AccessLog{}).
		Where("access_date BETWEEN ? AND ?", weekStart, tomorrow).
		Count(&weekVisits).Error
	if err != nil {
		return nil, err
	}
	stats["weekVisits"] = weekVisits

	// 获取本月访问次数
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	var monthVisits int64
	err = r.db.Model(&model.AccessLog{}).
		Where("access_date BETWEEN ? AND ?", monthStart, tomorrow).
		Count(&monthVisits).Error
	if err != nil {
		return nil, err
	}
	stats["monthVisits"] = monthVisits

	// 获取总访问次数
	var totalVisits int64
	err = r.db.Model(&model.AccessLog{}).
		Count(&totalVisits).Error
	if err != nil {
		return nil, err
	}
	stats["totalVisits"] = totalVisits

	return stats, nil
}

// 生成IP黑名单缓存键
func getIPBlacklistCacheKey(ip string) string {
	return IPBlacklistCacheKeyPrefix + ip
}

// BanIP 将IP添加到黑名单
func (r *LogRepository) BanIP(ip, reason string, expireTime time.Time) error {
	// 创建新的封禁记录
	blacklist := &model.IPBlacklist{
		IPAddress:  ip,
		BanReason:  reason,
		ExpireTime: expireTime,
	}

	err := r.db.Create(blacklist).Error
	if err == nil {
		// 更新缓存
		cacheKey := getIPBlacklistCacheKey(ip)
		_ = r.cache.Set(cacheKey, true, IPBlacklistCacheExpire)
		logrus.Debugf("IP %s 已加入黑名单并缓存", ip)
	} else {
		logrus.Errorf("将IP添加到黑名单失败: %s, 错误: %v", ip, err)
	}
	return err
}

// UnbanIP 从黑名单中移除IP
func (r *LogRepository) UnbanIP(ip string) error {
	// 将所有与该IP相关的记录标记为已删除（软删除）
	err := r.db.Where("ip_address = ?", ip).Delete(&model.IPBlacklist{}).Error
	if err == nil {
		// 更新缓存
		cacheKey := getIPBlacklistCacheKey(ip)
		_ = r.cache.Set(cacheKey, false, IPBlacklistCacheExpire)
		logrus.Debugf("IP %s 已从黑名单移除并更新缓存", ip)
	} else {
		logrus.Errorf("从黑名单移除IP失败: %s, 错误: %v", ip, err)
	}
	return err
}

// IsIPBanned 检查IP是否在黑名单中，优先从缓存获取
func (r *LogRepository) IsIPBanned(ip string) (bool, error) {
	cacheKey := getIPBlacklistCacheKey(ip)

	// 尝试从缓存获取
	if cached, found := r.cache.Get(cacheKey); found {
		if banned, ok := cached.(bool); ok {
			logrus.Debugf("从缓存获取IP %s 的黑名单状态: %v", ip, banned)
			return banned, nil
		} else {
			logrus.Warnf("IP黑名单状态缓存类型转换失败: %s, 将从数据库重新获取", ip)
		}
	}

	// 缓存未命中或类型转换失败，从数据库查询
	var count int64
	// 检查是否有该IP的有效封禁记录（未过期且未删除）
	err := r.db.Model(&model.IPBlacklist{}).
		Where("ip_address = ? AND (expire_time IS NULL OR expire_time > ?)", ip, time.Now()).
		Count(&count).Error

	if err != nil {
		logrus.Errorf("查询IP黑名单状态失败: %s, 错误: %v", ip, err)
		return false, fmt.Errorf("查询IP黑名单状态失败: %w", err)
	}

	isBanned := count > 0

	// 将结果存入缓存
	_ = r.cache.Set(cacheKey, isBanned, IPBlacklistCacheExpire)
	logrus.Debugf("IP %s 的黑名单状态已缓存: %v", ip, isBanned)

	return isBanned, nil
}

// ClearIPBlacklistCache 清除指定IP的黑名单缓存
func (r *LogRepository) ClearIPBlacklistCache(ip string) {
	cacheKey := getIPBlacklistCacheKey(ip)
	if deleted := r.cache.Delete(cacheKey); !deleted {
		logrus.Warnf("清除IP黑名单缓存失败: %s, 缓存中未找到", ip)
	} else {
		logrus.Debugf("已清除IP %s 的黑名单缓存", ip)
	}
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

// GetMonthlyVisitStats 获取每月访问统计
func (r *LogRepository) GetMonthlyVisitStats(year int) ([]map[string]interface{}, error) {
	if year == 0 {
		year = time.Now().Year()
	}

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.Local)

	var stats []map[string]interface{}
	err := r.db.Model(&model.AccessLog{}).
		Select("strftime('%m', access_date) as month, count(*) as visit_count").
		Where("access_date BETWEEN ? AND ?", startDate, endDate).
		Group("month").
		Order("month ASC").
		Find(&stats).Error

	// 确保12个月都有数据，如果某月没有数据，则设置为0
	result := make([]map[string]interface{}, 12)
	for i := 0; i < 12; i++ {
		result[i] = map[string]interface{}{
			"month":       i + 1,
			"visit_count": 0,
		}
	}

	// 填充实际数据
	for _, stat := range stats {
		monthStr, ok := stat["month"].(string)
		if !ok {
			continue
		}

		monthInt := 0
		_, err := fmt.Sscanf(monthStr, "%d", &monthInt)
		if err != nil || monthInt < 1 || monthInt > 12 {
			continue
		}

		result[monthInt-1]["visit_count"] = stat["visit_count"]
	}

	return result, err
}

// GetDailyVisitStatsForLastDays 获取最近n天的每日访问统计
func (r *LogRepository) GetDailyVisitStatsForLastDays(days int) ([]map[string]interface{}, error) {
	if days <= 0 {
		days = 30 // 默认30天
	}

	endDate := time.Now().Add(24 * time.Hour) // 加一天确保包含今天的数据
	startDate := endDate.AddDate(0, 0, -days)

	var stats []map[string]interface{}
	err := r.db.Model(&model.AccessLog{}).
		Select("strftime('%Y-%m-%d', access_date) as date, count(*) as visit_count").
		Where("access_date BETWEEN ? AND ?", startDate, endDate).
		Group("date").
		Order("date ASC").
		Find(&stats).Error

	// 确保所有日期都有数据，如果某天没有数据，则设置为0
	today := time.Now()
	result := make([]map[string]interface{}, days)
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -days+1+i) // 从 (今天-days+1) 到今天
		dateStr := date.Format("2006-01-02")
		result[i] = map[string]interface{}{
			"date":        dateStr,
			"visit_count": 0,
		}
	}

	// 填充实际数据
	for _, stat := range stats {
		dateStr, ok := stat["date"].(string)
		if !ok {
			continue
		}

		for i, r := range result {
			if r["date"] == dateStr {
				result[i]["visit_count"] = stat["visit_count"]
				break
			}
		}
	}

	return result, err
}
