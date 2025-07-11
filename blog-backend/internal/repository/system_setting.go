package repository

import (
	"errors"
	"fmt"
	"time"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 缓存键前缀
const (
	PrefixSystemSetting = "system:setting:"
	ExpireMedium        = time.Minute * 30 // 中期缓存，30分钟
)

type SystemSettingRepository interface {
	GetAllSettings() ([]model.SystemSetting, error)
	UpdateSetting(key, value string) error
	BatchUpdateSettings(settings map[string]string) error
	GetSetting(key string) (string, error)
	ClearCache() // 新增：清除缓存
}

type systemSettingRepository struct {
	db    *gorm.DB
	cache dhcache.Cache
}

// NewSystemSettingRepository 创建系统设置仓库
func NewSystemSettingRepository(db *gorm.DB, cache dhcache.Cache) SystemSettingRepository {
	return &systemSettingRepository{
		db:    db,
		cache: cache,
	}
}

// settingsCacheKey 系统设置缓存的键名
const settingsCacheKey = "system:all_settings"

// GetAllSettings 获取所有系统设置，优先从缓存获取
func (r *systemSettingRepository) GetAllSettings() ([]model.SystemSetting, error) {
	// 尝试从缓存获取
	if cached, found := r.cache.Get(settingsCacheKey); found {
		if settings, ok := cached.([]model.SystemSetting); ok {
			logrus.Debug("从缓存获取系统设置")
			return settings, nil
		}
	}

	// 缓存未命中，从数据库获取
	var settings []model.SystemSetting
	err := r.db.Find(&settings).Error
	if err != nil {
		return nil, err
	}

	// 存入缓存，设置10分钟过期
	_ = r.cache.Set(settingsCacheKey, settings, time.Minute*10)
	logrus.Debug("系统设置已缓存")

	return settings, nil
}

// GetSetting 获取单个系统设置
func (r *systemSettingRepository) GetSetting(key string) (string, error) {
	// 缓存键名
	cacheKey := PrefixSystemSetting + key

	// 尝试从缓存获取
	if cached, found := r.cache.Get(cacheKey); found {
		if value, ok := cached.(string); ok {
			return value, nil
		}
	}

	// 缓存未命中，从数据库获取
	var setting model.SystemSetting
	err := r.db.Where("setting_key = ?", key).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("设置不存在: %s", key)
		}
		return "", err
	}

	// 存入缓存
	_ = r.cache.Set(cacheKey, setting.SettingValue, ExpireMedium)

	return setting.SettingValue, nil
}

func (r *systemSettingRepository) UpdateSetting(key, value string) error {
	var setting model.SystemSetting
	// 尝试查找现有设置
	err := r.db.Where("setting_key = ?", key).First(&setting).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果未找到，则创建新设置
			setting = model.SystemSetting{
				SettingKey:   key,
				SettingValue: value,
			}

			// 使用短事务创建新设置
			var createErr error
			for i := 0; i < 3; i++ {
				createErr = r.db.Create(&setting).Error
				if createErr == nil {
					// 更新成功后清除相关缓存
					r.clearSettingCache(key)
					return nil
				}
				time.Sleep(50 * time.Millisecond)
			}
			return createErr
		} else {
			return err
		}
	}

	// 如果找到，则更新现有设置
	setting.SettingValue = value

	// 使用短事务更新设置
	var saveErr error
	for i := 0; i < 3; i++ {
		saveErr = r.db.Save(&setting).Error
		if saveErr == nil {
			// 更新成功后清除相关缓存
			r.clearSettingCache(key)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return saveErr
}

func (r *systemSettingRepository) BatchUpdateSettings(settings map[string]string) error {
	// 对于每个设置项，单独执行更新操作
	for key, value := range settings {
		err := r.UpdateSetting(key, value)
		if err != nil {
			return fmt.Errorf("更新设置 %s 失败: %w", key, err)
		}
	}

	// 所有设置更新成功后，清除全局设置缓存
	r.cache.Delete(settingsCacheKey)
	logrus.Debug("批量更新设置后清除缓存")

	return nil
}

// ClearCache 清除所有系统设置缓存
func (r *systemSettingRepository) ClearCache() {
	r.cache.Delete(settingsCacheKey)
	logrus.Info("已清除系统设置缓存")
}

// clearSettingCache 清除单个设置的缓存
func (r *systemSettingRepository) clearSettingCache(key string) {
	// 清除全局设置缓存
	r.cache.Delete(settingsCacheKey)
	// 清除单个设置缓存
	r.cache.Delete(PrefixSystemSetting + key)
	logrus.Debugf("已清除设置缓存: %s", key)
}
