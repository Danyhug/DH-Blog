package repository

import (
	"errors"
	"fmt"
	"time"

	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type SystemSettingRepository interface {
	GetAllSettings() ([]model.SystemSetting, error)
	UpdateSetting(key, value string) error
	BatchUpdateSettings(settings map[string]string) error
}

type systemSettingRepository struct {
	db *gorm.DB
}

func NewSystemSettingRepository(db *gorm.DB) SystemSettingRepository {
	return &systemSettingRepository{db: db}
}

func (r *systemSettingRepository) GetAllSettings() ([]model.SystemSetting, error) {
	var settings []model.SystemSetting
	err := r.db.Find(&settings).Error
	return settings, err
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
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return saveErr
}

func (r *systemSettingRepository) BatchUpdateSettings(settings map[string]string) error {
	// 对于每个设置项，单独执行更新操作
	for key, value := range settings {
		// 添加重试逻辑
		var retryCount int
		for retryCount < 3 {
			err := r.UpdateSetting(key, value)
			if err == nil {
				break // 成功则跳出重试循环
			}

			// 如果失败，增加重试计数并等待一段时间
			retryCount++
			if retryCount >= 3 {
				return fmt.Errorf("更新设置 %s 失败，已重试 %d 次: %w", key, retryCount, err)
			}

			// 等待 100ms 后重试
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}
