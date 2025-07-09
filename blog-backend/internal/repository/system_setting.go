package repository

import (
	"errors"

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
			return r.db.Create(&setting).Error
		} else {
			return err
		}
	}

	// 如果找到，则更新现有设置
	setting.SettingValue = value
	return r.db.Save(&setting).Error
}

func (r *systemSettingRepository) BatchUpdateSettings(settings map[string]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range settings {
			var setting model.SystemSetting
			err := tx.Where("setting_key = ?", key).First(&setting).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					setting = model.SystemSetting{
						SettingKey:   key,
						SettingValue: value,
					}
					if createErr := tx.Create(&setting).Error; createErr != nil {
						return createErr
					}
				} else {
					return err
				}
			} else {
				setting.SettingValue = value
				if saveErr := tx.Save(&setting).Error; saveErr != nil {
					return saveErr
				}
			}
		}
		return nil
	})
}
