package logging

import (
	"fmt"
	"sync"
	"time"

	"dh-blog/internal/dhcache"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	ipBlacklistCacheKeyPrefix = "ip:blacklist:"
	ipBlacklistCacheExpire    = 10 * time.Minute
	defaultAccessLogBatchSize = 100
)

// Repository owns access-log and IP-blacklist persistence.
type Repository struct {
	db              *gorm.DB
	cache           dhcache.Cache
	mu              sync.Mutex
	accessLogBuffer []AccessLog
	batchSize       int
}

func newRepository(db *gorm.DB, cache dhcache.Cache) *Repository {
	return &Repository{
		db:        db,
		cache:     cache,
		batchSize: defaultAccessLogBatchSize,
	}
}

// SaveAccessLog buffers access logs and writes a batch when it reaches the
// configured size. A failed batch is restored to the front of the buffer.
func (r *Repository) SaveAccessLog(log *AccessLog) error {
	if log == nil {
		return nil
	}

	r.mu.Lock()
	r.accessLogBuffer = append(r.accessLogBuffer, *log)
	if len(r.accessLogBuffer) < r.batchSize {
		r.mu.Unlock()
		return nil
	}

	batch := make([]AccessLog, len(r.accessLogBuffer))
	copy(batch, r.accessLogBuffer)
	r.accessLogBuffer = r.accessLogBuffer[:0]
	r.mu.Unlock()

	if err := r.flushAccessLogs(batch); err != nil {
		r.mu.Lock()
		r.accessLogBuffer = append(batch, r.accessLogBuffer...)
		r.mu.Unlock()
		return err
	}

	return nil
}

// SaveAccessLogAsync persists an access log without blocking its caller.
func (r *Repository) SaveAccessLogAsync(log *AccessLog) {
	go func() {
		if err := r.SaveAccessLog(log); err != nil {
			logrus.Errorf("异步保存访问日志失败: %v", err)
		}
	}()
}

func (r *Repository) flushAccessLogs(logs []AccessLog) error {
	if len(logs) == 0 {
		return nil
	}
	if err := r.db.Create(&logs).Error; err != nil {
		logrus.Errorf("批量保存访问日志失败: %v", err)
		return err
	}
	logrus.Debugf("批量保存访问日志成功，数量: %d", len(logs))
	return nil
}

func (r *Repository) GetVisitLogs(page, pageSize int) ([]AccessLog, int64, error) {
	var logs []AccessLog
	var total int64

	r.db.Model(&AccessLog{}).Count(&total)
	err := r.db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&logs).Error
	return logs, total, err
}

func (r *Repository) GetIPVisitStats(page, pageSize int, startDate, endDate time.Time) ([]map[string]interface{}, int64, error) {
	var result []map[string]interface{}

	banSubQuery := r.db.Unscoped().Model(&IPBlacklist{}).
		Select("ip_address, COUNT(*) as banned_count").
		Group("ip_address")
	currentBanSubQuery := r.db.Model(&IPBlacklist{}).
		Select("ip_address, 1 as ban_status").
		Where("(expire_time IS NULL OR expire_time > ?)", time.Now())

	query := r.db.Model(&AccessLog{}).
		Select("access_logs.ip_address as ipAddress, MAX(access_logs.city) as city, "+
			"COUNT(access_logs.id) as accessCount, "+
			"IFNULL(ban_stats.banned_count, 0) as bannedCount, "+
			"IFNULL(current_ban.ban_status, 0) as banStatus").
		Joins("LEFT JOIN (?) as ban_stats ON access_logs.ip_address = ban_stats.ip_address", banSubQuery).
		Joins("LEFT JOIN (?) as current_ban ON access_logs.ip_address = current_ban.ip_address", currentBanSubQuery).
		Where("access_logs.access_date BETWEEN ? AND ?", startDate, endDate).
		Group("access_logs.ip_address").
		Order("accessCount DESC")

	var countResult []map[string]interface{}
	if err := query.Find(&countResult).Error; err != nil {
		return nil, 0, err
	}
	total := int64(len(countResult))
	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&result).Error
	return result, total, err
}

func (r *Repository) GetVisitStatistics() (map[string]int64, error) {
	stats := make(map[string]int64)
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var todayVisits int64
	if err := r.db.Model(&AccessLog{}).Where("access_date BETWEEN ? AND ?", today, tomorrow).Count(&todayVisits).Error; err != nil {
		return nil, err
	}
	stats["todayVisits"] = todayVisits

	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	var weekVisits int64
	if err := r.db.Model(&AccessLog{}).Where("access_date BETWEEN ? AND ?", weekStart, tomorrow).Count(&weekVisits).Error; err != nil {
		return nil, err
	}
	stats["weekVisits"] = weekVisits

	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	var monthVisits int64
	if err := r.db.Model(&AccessLog{}).Where("access_date BETWEEN ? AND ?", monthStart, tomorrow).Count(&monthVisits).Error; err != nil {
		return nil, err
	}
	stats["monthVisits"] = monthVisits

	var totalVisits int64
	if err := r.db.Model(&AccessLog{}).Count(&totalVisits).Error; err != nil {
		return nil, err
	}
	stats["totalVisits"] = totalVisits
	return stats, nil
}

func getIPBlacklistCacheKey(ip string) string {
	return ipBlacklistCacheKeyPrefix + ip
}

func (r *Repository) BanIP(ip, reason string, expireTime time.Time) error {
	blacklist := &IPBlacklist{IPAddress: ip, BanReason: reason, ExpireTime: expireTime}
	if err := r.db.Create(blacklist).Error; err != nil {
		logrus.Errorf("将IP添加到黑名单失败: %s, 错误: %v", ip, err)
		return err
	}
	_ = r.cache.Set(getIPBlacklistCacheKey(ip), true, ipBlacklistCacheExpire)
	logrus.Debugf("IP %s 已加入黑名单并缓存", ip)
	return nil
}

func (r *Repository) UnbanIP(ip string) error {
	if err := r.db.Where("ip_address = ?", ip).Delete(&IPBlacklist{}).Error; err != nil {
		logrus.Errorf("从黑名单移除IP失败: %s, 错误: %v", ip, err)
		return err
	}
	_ = r.cache.Set(getIPBlacklistCacheKey(ip), false, ipBlacklistCacheExpire)
	logrus.Debugf("IP %s 已从黑名单移除并更新缓存", ip)
	return nil
}

func (r *Repository) IsIPBanned(ip string) (bool, error) {
	cacheKey := getIPBlacklistCacheKey(ip)
	if cached, found := r.cache.Get(cacheKey); found {
		if banned, ok := cached.(bool); ok {
			logrus.Debugf("从缓存获取IP %s 的黑名单状态: %v", ip, banned)
			return banned, nil
		}
		logrus.Warnf("IP黑名单状态缓存类型转换失败: %s, 将从数据库重新获取", ip)
	}

	var count int64
	if err := r.db.Model(&IPBlacklist{}).
		Where("ip_address = ? AND (expire_time IS NULL OR expire_time > ?)", ip, time.Now()).
		Count(&count).Error; err != nil {
		logrus.Errorf("查询IP黑名单状态失败: %s, 错误: %v", ip, err)
		return false, fmt.Errorf("查询IP黑名单状态失败: %w", err)
	}

	isBanned := count > 0
	_ = r.cache.Set(cacheKey, isBanned, ipBlacklistCacheExpire)
	logrus.Debugf("IP %s 的黑名单状态已缓存: %v", ip, isBanned)
	return isBanned, nil
}

func (r *Repository) ClearIPBlacklistCache(ip string) {
	if deleted := r.cache.Delete(getIPBlacklistCacheKey(ip)); !deleted {
		logrus.Warnf("清除IP黑名单缓存失败: %s, 缓存中未找到", ip)
	} else {
		logrus.Debugf("已清除IP %s 的黑名单缓存", ip)
	}
}

func (r *Repository) GetDailyVisitStats(startDate, endDate time.Time) ([]map[string]interface{}, error) {
	var stats []map[string]interface{}
	err := r.db.Model(&AccessLog{}).
		Select("strftime('%Y-%m-%d', access_date) as date, count(*) as visit_count").
		Where("access_date BETWEEN ? AND ?", startDate, endDate).
		Group("date").
		Order("date DESC").
		Find(&stats).Error
	return stats, err
}

func (r *Repository) GetMonthlyVisitStats(year int) ([]map[string]interface{}, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.Local)

	var stats []map[string]interface{}
	err := r.db.Model(&AccessLog{}).
		Select("strftime('%m', access_date) as month, count(*) as visit_count").
		Where("access_date BETWEEN ? AND ?", startDate, endDate).
		Group("month").
		Order("month ASC").
		Find(&stats).Error

	result := make([]map[string]interface{}, 12)
	for i := 0; i < 12; i++ {
		result[i] = map[string]interface{}{"month": i + 1, "visit_count": 0}
	}
	for _, stat := range stats {
		monthStr, ok := stat["month"].(string)
		if !ok {
			continue
		}
		monthInt := 0
		if _, scanErr := fmt.Sscanf(monthStr, "%d", &monthInt); scanErr != nil || monthInt < 1 || monthInt > 12 {
			continue
		}
		result[monthInt-1]["visit_count"] = stat["visit_count"]
	}
	return result, err
}

func (r *Repository) GetDailyVisitStatsForLastDays(days int) ([]map[string]interface{}, error) {
	if days <= 0 {
		days = 30
	}
	endDate := time.Now().Add(24 * time.Hour)
	startDate := endDate.AddDate(0, 0, -days)

	var stats []map[string]interface{}
	err := r.db.Model(&AccessLog{}).
		Select("strftime('%Y-%m-%d', access_date) as date, count(*) as visit_count").
		Where("access_date BETWEEN ? AND ?", startDate, endDate).
		Group("date").
		Order("date ASC").
		Find(&stats).Error

	today := time.Now()
	result := make([]map[string]interface{}, days)
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -days+1+i)
		result[i] = map[string]interface{}{"date": date.Format("2006-01-02"), "visit_count": 0}
	}
	for _, stat := range stats {
		dateStr, ok := stat["date"].(string)
		if !ok {
			continue
		}
		for i, item := range result {
			if item["date"] == dateStr {
				result[i]["visit_count"] = stat["visit_count"]
				break
			}
		}
	}
	return result, err
}
