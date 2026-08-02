package aigateway

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dh-blog/internal/platform/search"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func newRepository(db *gorm.DB) *repository { return &repository{db: db} }

// defaultProviders seeds both upstreams disabled and keyless so the admin page
// has rows to edit on a fresh install.
func defaultProviders() []Provider {
	return []Provider{
		{
			Name: search.ProviderBrave, DisplayName: "Brave Search", Enabled: false,
			// Brave's free tier is one request per second and 2000 per month.
			Priority: 100, Weight: 1, RPS: 1, MonthlyQuota: 2000, Extra: "{}",
		},
		{
			Name: search.ProviderTavily, DisplayName: "Tavily", Enabled: false,
			Priority: 100, Weight: 1, RPS: 5, MonthlyQuota: 1000,
			Extra: `{"search_depth":"basic","chunks_per_source":3}`,
		},
		{
			Name: search.ProviderExa, DisplayName: "Exa", Enabled: false,
			// Exa bills by amount, not by call: a request costs whatever it
			// costs, so a request cap cannot express the budget that actually
			// runs out. The ceiling is the free tier's $10 a month instead.
			// Existing installs keep whatever they had — silently starting to
			// block traffic on an upgrade would be worse than an open tap.
			Priority: 100, Weight: 1, RPS: 5, MonthlyQuota: 0,
			MonthlyCostLimit: 10_000_000,
			Extra:            `{"search_type":"auto"}`,
		},
	}
}

// ensureDefaults inserts the seed provider rows and gateway settings, leaving
// existing ones alone.
func (r *repository) ensureDefaults(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, provider := range defaultProviders() {
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoNothing: true,
			}).Create(&provider).Error
			if err != nil {
				return fmt.Errorf("补齐搜索供应商 %s: %w", provider.Name, err)
			}
		}
		for key, value := range defaultSettings() {
			setting := Setting{SettingKey: key, SettingValue: value}
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "setting_key"}},
				DoNothing: true,
			}).Create(&setting).Error
			if err != nil {
				return fmt.Errorf("补齐网关设置 %s: %w", key, err)
			}
		}
		return migrateLegacyProviderKeys(tx)
	})
}

// migrateLegacyProviderKeys moves the single credential that used to live on
// the provider row into the credential table, so there is exactly one place a
// key can come from. The column is cleared on the way out; leaving a copy
// behind would mean two sources of truth that silently drift apart.
func migrateLegacyProviderKeys(tx *gorm.DB) error {
	var providers []Provider
	if err := tx.Where("api_key <> ''").Find(&providers).Error; err != nil {
		return fmt.Errorf("读取旧版供应商密钥: %w", err)
	}
	for _, provider := range providers {
		key := ProviderKey{
			Provider: provider.Name,
			Label:    "默认",
			APIKey:   provider.APIKey,
			Enabled:  true,
			Status:   ProviderKeyActive,
		}
		if err := tx.Create(&key).Error; err != nil {
			return fmt.Errorf("迁移供应商 %s 的密钥: %w", provider.Name, err)
		}
		if err := tx.Model(&Provider{}).Where("name = ?", provider.Name).
			Update("api_key", "").Error; err != nil {
			return fmt.Errorf("清理供应商 %s 的旧密钥列: %w", provider.Name, err)
		}
		logrus.Infof("已将供应商 %s 的密钥迁移到多密钥表", provider.Name)
	}
	return nil
}

func (r *repository) listProviderKeys(ctx context.Context) ([]ProviderKey, error) {
	var keys []ProviderKey
	err := r.db.WithContext(ctx).Order("provider, id").Find(&keys).Error
	return keys, err
}

func (r *repository) providerKeyByID(ctx context.Context, id int) (ProviderKey, error) {
	var key ProviderKey
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ProviderKey{}, ErrProviderKeyNotFound
	}
	return key, err
}

func (r *repository) createProviderKey(ctx context.Context, key *ProviderKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *repository) updateProviderKey(ctx context.Context, id int, updates map[string]any) error {
	result := r.db.WithContext(ctx).Model(&ProviderKey{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderKeyNotFound
	}
	return nil
}

func (r *repository) deleteProviderKey(ctx context.Context, id int) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ProviderKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderKeyNotFound
	}
	return nil
}

// parkProviderKey records why a credential left the rotation. Writing the
// reason matters: "都不通" without a cause sends the operator digging through logs.
func (r *repository) parkProviderKey(ctx context.Context, id int, status, reason string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&ProviderKey{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"last_error":  truncateQuery(reason),
		"disabled_at": now,
	}).Error
}

// reviveProviderKey puts a credential back into rotation and clears the reason.
func (r *repository) reviveProviderKey(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Model(&ProviderKey{}).Where("id = ?", id).Updates(map[string]any{
		"status":      ProviderKeyActive,
		"last_error":  "",
		"disabled_at": nil,
	}).Error
}

func (r *repository) touchProviderKey(ctx context.Context, id int, now time.Time) error {
	return r.db.WithContext(ctx).Model(&ProviderKey{}).Where("id = ?", id).
		Update("last_used_at", now).Error
}

func defaultSettings() map[string]string {
	return map[string]string{
		SettingKeyRoutingStrategy: string(StrategyBalanced),
	}
}

func (r *repository) setting(ctx context.Context, key string) (string, error) {
	var setting Setting
	err := r.db.WithContext(ctx).Where("setting_key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return setting.SettingValue, err
}

func (r *repository) saveSetting(ctx context.Context, key, value string) error {
	setting := Setting{SettingKey: key, SettingValue: value}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"setting_value"}),
	}).Create(&setting).Error
}

func (r *repository) listProviders(ctx context.Context) ([]Provider, error) {
	var providers []Provider
	err := r.db.WithContext(ctx).Order("priority, name").Find(&providers).Error
	return providers, err
}

func (r *repository) providerByName(ctx context.Context, name string) (Provider, error) {
	var provider Provider
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Provider{}, ErrProviderNotFound
	}
	return provider, err
}

// updateProvider applies an admin patch. An empty api_key means "leave the
// stored credential alone", so the masked value the UI shows can be sent back
// unchanged without wiping the key.
func (r *repository) updateProvider(ctx context.Context, name string, updates map[string]any) error {
	result := r.db.WithContext(ctx).Model(&Provider{}).Where("name = ?", name).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}

func (r *repository) listAPIKeys(ctx context.Context) ([]APIKey, error) {
	var keys []APIKey
	err := r.db.WithContext(ctx).Order("id desc").Find(&keys).Error
	return keys, err
}

func (r *repository) apiKeyByPrefix(ctx context.Context, prefix string) (APIKey, error) {
	var key APIKey
	err := r.db.WithContext(ctx).Where("key_prefix = ?", prefix).First(&key).Error
	return key, err
}

func (r *repository) apiKeyByID(ctx context.Context, id int) (APIKey, error) {
	var key APIKey
	err := r.db.WithContext(ctx).First(&key, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return APIKey{}, errors.New("API Key 不存在")
	}
	return key, err
}

func (r *repository) createAPIKey(ctx context.Context, key *APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *repository) updateAPIKey(ctx context.Context, id int, updates map[string]any) error {
	result := r.db.WithContext(ctx).Model(&APIKey{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("API Key 不存在")
	}
	return nil
}

func (r *repository) deleteAPIKey(ctx context.Context, id int) error {
	result := r.db.WithContext(ctx).Delete(&APIKey{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("API Key 不存在")
	}
	return nil
}

func (r *repository) touchAPIKey(ctx context.Context, id int, at time.Time) error {
	return r.db.WithContext(ctx).Model(&APIKey{}).Where("id = ?", id).
		Update("last_used_at", at).Error
}

func (r *repository) writeLog(ctx context.Context, entry *RequestLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

type logFilter struct {
	Provider string
	Status   string
	Page     int
	PageSize int
}

func (r *repository) listLogs(ctx context.Context, filter logFilter) ([]RequestLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&RequestLog{})
	if filter.Provider != "" {
		query = query.Where("provider = ?", filter.Provider)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	var logs []RequestLog
	err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

// deleteLogsBefore prunes the request log; the gateway keeps 90 days.
func (r *repository) deleteLogsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&RequestLog{})
	return result.RowsAffected, result.Error
}

// addUsage increments a monthly counter atomically.
func (r *repository) addUsage(ctx context.Context, subject, period string, count, credits, costMicroUSD int) error {
	usage := Usage{Subject: subject, Period: period, Count: count, Credits: credits, CostMicroUSD: costMicroUSD}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "subject"}, {Name: "period"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":          gorm.Expr("count + ?", count),
			"credits":        gorm.Expr("credits + ?", credits),
			"cost_micro_usd": gorm.Expr("cost_micro_usd + ?", costMicroUSD),
		}),
	}).Create(&usage).Error
}

// usageFor returns the counters for the given subjects in one query, keyed by
// subject. Missing subjects simply have no entry.
func (r *repository) usageFor(ctx context.Context, period string, subjects []string) (map[string]Usage, error) {
	usage := make(map[string]Usage, len(subjects))
	if len(subjects) == 0 {
		return usage, nil
	}
	var rows []Usage
	err := r.db.WithContext(ctx).Where("period = ? AND subject IN ?", period, subjects).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		usage[row.Subject] = row
	}
	return usage, nil
}

// statsRow is one provider's aggregate over a time span.
type statsRow struct {
	Provider     string  `json:"provider"`
	Total        int64   `json:"total"`
	Succeeded    int64   `json:"succeeded"`
	Cached       int64   `json:"cached"`
	Credits      int64   `json:"credits"`
	CostMicroUSD int64   `json:"costMicroUsd"`
	AvgLatency   float64 `json:"avgLatency"`
}

func (r *repository) statsSince(ctx context.Context, since time.Time) ([]statsRow, error) {
	var rows []statsRow
	err := r.db.WithContext(ctx).Model(&RequestLog{}).
		Select(`provider,
		        COUNT(*) AS total,
		        SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS succeeded,
		        SUM(CASE WHEN cached THEN 1 ELSE 0 END) AS cached,
		        COALESCE(SUM(credits), 0) AS credits,
		        COALESCE(SUM(cost_micro_usd), 0) AS cost_micro_usd,
		        COALESCE(AVG(latency_ms), 0) AS avg_latency`, StatusOK).
		Where("created_at >= ?", since).
		Group("provider").
		Scan(&rows).Error
	return rows, err
}
