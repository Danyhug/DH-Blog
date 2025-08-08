package service

import (
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
)

// IConfigService 配置服务接口
type IConfigService interface {
	GetSystemConfig() (*model.SystemConfig, error)
	GetStorageConfig() (*model.StorageConfig, error)
	UpdateSystemConfig(config *model.SystemConfig) error
}

// configService 配置服务实现
type configService struct {
	settingRepo repository.SystemSettingRepository
}

// NewConfigService 创建配置服务
func NewConfigService(settingRepo repository.SystemSettingRepository) IConfigService {
	return &configService{
		settingRepo: settingRepo,
	}
}

// GetSystemConfig 获取系统配置
func (s *configService) GetSystemConfig() (*model.SystemConfig, error) {
	settings, err := s.settingRepo.GetAllSettings()
	if err != nil {
		return nil, err
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, setting := range settings {
		settingsMap[setting.SettingKey] = setting.SettingValue
	}

	// 使用新的映射方法创建配置对象
	config := model.FromSettingsMap(settingsMap)
	return config, nil
}

// GetStorageConfig 获取存储配置
func (s *configService) GetStorageConfig() (*model.StorageConfig, error) {
	config, err := s.GetSystemConfig()
	if err != nil {
		return nil, err
	}
	return config.GetStorageConfig(), nil
}

// UpdateSystemConfig 更新系统配置
func (s *configService) UpdateSystemConfig(config *model.SystemConfig) error {
	settingsMap := config.ToSettingsMap()
	return s.settingRepo.BatchUpdateSettings(settingsMap)
}