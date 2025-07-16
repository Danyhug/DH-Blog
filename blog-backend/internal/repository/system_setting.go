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
	UpdateSettingWithType(key, value, configType string) error
	BatchUpdateSettings(settings map[string]string) error
	BatchUpdateSettingsWithType(settings map[string]string, configType string) error
	GetSetting(key string) (string, error)
	GetSettingsByType(configType string) ([]model.SystemSetting, error)
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

// GetSettingsByType 按类型获取系统设置
func (r *systemSettingRepository) GetSettingsByType(configType string) ([]model.SystemSetting, error) {
	// 缓存键名
	cacheKey := fmt.Sprintf("%stype:%s", PrefixSystemSetting, configType)

	// 尝试从缓存获取
	if cached, found := r.cache.Get(cacheKey); found {
		if settings, ok := cached.([]model.SystemSetting); ok {
			logrus.Debugf("从缓存获取类型[%s]的系统设置", configType)
			return settings, nil
		}
	}

	// 缓存未命中，从数据库获取
	var settings []model.SystemSetting
	err := r.db.Where("config_type = ?", configType).Find(&settings).Error
	if err != nil {
		return nil, err
	}

	// 存入缓存
	_ = r.cache.Set(cacheKey, settings, ExpireMedium)
	logrus.Debugf("类型[%s]的系统设置已缓存", configType)

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
	return r.UpdateSettingWithType(key, value, "")
}

func (r *systemSettingRepository) UpdateSettingWithType(key, value, configType string) error {
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

			// 如果指定了配置类型，则设置
			if configType != "" {
				setting.ConfigType = configType
			} else {
				// 默认为blog类型
				setting.ConfigType = model.ConfigTypeBlog
			}

			// 使用短事务创建新设置
			var createErr error
			for i := 0; i < 3; i++ {
				createErr = r.db.Create(&setting).Error
				if createErr == nil {
					// 更新成功后清除相关缓存
					r.clearSettingCache(key)
					r.clearTypeCache(setting.ConfigType)
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

	// 如果指定了配置类型且与当前不同，则更新
	if configType != "" && configType != setting.ConfigType {
		oldType := setting.ConfigType
		setting.ConfigType = configType
		// 后续清除旧类型缓存
		defer r.clearTypeCache(oldType)
	}

	// 使用短事务更新设置
	var saveErr error
	for i := 0; i < 3; i++ {
		saveErr = r.db.Save(&setting).Error
		if saveErr == nil {
			// 更新成功后清除相关缓存
			r.clearSettingCache(key)
			r.clearTypeCache(setting.ConfigType)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return saveErr
}

func (r *systemSettingRepository) BatchUpdateSettings(settings map[string]string) error {
	return r.BatchUpdateSettingsWithType(settings, "")
}

func (r *systemSettingRepository) BatchUpdateSettingsWithType(settings map[string]string, configType string) error {
	// 对于每个设置项，单独执行更新操作
	for key, value := range settings {
		err := r.UpdateSettingWithType(key, value, configType)
		if err != nil {
			return fmt.Errorf("更新设置 %s 失败: %w", key, err)
		}
	}

	// 所有设置更新成功后，清除全局设置缓存
	r.cache.Delete(settingsCacheKey)
	if configType != "" {
		r.clearTypeCache(configType)
	}
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

// clearTypeCache 清除某个类型的设置缓存
func (r *systemSettingRepository) clearTypeCache(configType string) {
	cacheKey := fmt.Sprintf("%stype:%s", PrefixSystemSetting, configType)
	r.cache.Delete(cacheKey)
	logrus.Debugf("已清除类型[%s]的设置缓存", configType)
}
