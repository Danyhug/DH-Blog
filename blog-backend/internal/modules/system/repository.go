package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dh-blog/internal/dhcache"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const allSettingsCacheKey = "system:all_settings"

type settingRepository struct {
	db    *gorm.DB
	cache dhcache.Cache
}

func newSettingRepository(db *gorm.DB, cache dhcache.Cache) *settingRepository {
	return &settingRepository{db: db, cache: cache}
}

func (r *settingRepository) ensureDefaults(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range DefaultSettings() {
			setting := Setting{SettingKey: item.Key, SettingValue: item.Value, ConfigType: item.ConfigType}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "setting_key"}}, DoNothing: true}).Create(&setting).Error; err != nil {
				return fmt.Errorf("补齐系统设置 %s: %w", item.Key, err)
			}
			if err := tx.Model(&Setting{}).Where("setting_key = ? AND config_type <> ?", item.Key, item.ConfigType).Update("config_type", item.ConfigType).Error; err != nil {
				return fmt.Errorf("修复系统设置类型 %s: %w", item.Key, err)
			}
		}
		return nil
	})
}

func (r *settingRepository) all(ctx context.Context) ([]Setting, error) {
	if cached, ok := r.cache.Get(allSettingsCacheKey); ok {
		if settings, valid := cached.([]Setting); valid {
			return append([]Setting(nil), settings...), nil
		}
	}
	var settings []Setting
	if err := r.db.WithContext(ctx).Order("id").Find(&settings).Error; err != nil {
		return nil, err
	}
	_ = r.cache.Set(allSettingsCacheKey, settings, 10*time.Minute)
	return settings, nil
}

func (r *settingRepository) byType(ctx context.Context, configType string) ([]Setting, error) {
	var settings []Setting
	err := r.db.WithContext(ctx).Where("config_type = ?", configType).Order("id").Find(&settings).Error
	return settings, err
}

func (r *settingRepository) value(ctx context.Context, key string) (string, error) {
	var setting Setting
	if err := r.db.WithContext(ctx).Where("setting_key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("设置不存在: %s", key)
		}
		return "", err
	}
	return setting.SettingValue, nil
}

func (r *settingRepository) update(ctx context.Context, key, value, configType string) error {
	return r.updateBatch(ctx, map[string]string{key: value}, configType)
}

func (r *settingRepository) updateBatch(ctx context.Context, values map[string]string, configType string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			var setting Setting
			err := tx.Where("setting_key = ?", key).First(&setting).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				setting = Setting{SettingKey: key, SettingValue: value, ConfigType: configType}
				if setting.ConfigType == "" {
					setting.ConfigType = ConfigTypeBlog
				}
				if err := tx.Create(&setting).Error; err != nil {
					return fmt.Errorf("创建设置 %s: %w", key, err)
				}
				continue
			}
			if err != nil {
				return err
			}
			updates := map[string]any{"setting_value": value}
			if configType != "" {
				updates["config_type"] = configType
			}
			if err := tx.Model(&setting).Updates(updates).Error; err != nil {
				return fmt.Errorf("更新设置 %s: %w", key, err)
			}
		}
		return nil
	})
	if err == nil {
		r.clearCache()
	}
	return err
}

func (r *settingRepository) delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&Setting{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	r.clearCache()
	return nil
}

func (r *settingRepository) find(ctx context.Context, id uint) (Setting, error) {
	var setting Setting
	err := r.db.WithContext(ctx).First(&setting, id).Error
	return setting, err
}

func (r *settingRepository) updateID(ctx context.Context, id uint, key, value, configType string) error {
	updates := map[string]any{"setting_key": key, "setting_value": value}
	if configType != "" {
		updates["config_type"] = configType
	}
	result := r.db.WithContext(ctx).Model(&Setting{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	r.clearCache()
	return nil
}

func (r *settingRepository) clearCache() { r.cache.Delete(allSettingsCacheKey) }
