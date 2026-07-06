package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"

	"github.com/sirupsen/logrus"
)

// SystemModule 注册系统配置和系统设置路由。
type SystemModule struct {
	configController  *controller.SystemConfigController
	settingController *controller.SystemSettingController
}

func NewSystemModule(configController *controller.SystemConfigController, settingController *controller.SystemSettingController) *SystemModule {
	return &SystemModule{
		configController:  configController,
		settingController: settingController,
	}
}

func (m *SystemModule) RegisterRoutes(routes *router.Routes) {
	logrus.Info("注册系统配置路由")

	configGroup := routes.AdminAPI.Group("/config")
	configGroup.GET("", m.configController.GetConfigs)
	configGroup.PUT("", m.configController.UpdateConfigs)
	configGroup.GET("/blog", m.configController.GetBlogConfig)
	configGroup.PUT("/blog", m.configController.UpdateBlogConfig)
	configGroup.GET("/email", m.configController.GetEmailConfig)
	configGroup.PUT("/email", m.configController.UpdateEmailConfig)
	configGroup.GET("/ai", m.configController.GetAIConfig)
	configGroup.PUT("/ai", m.configController.UpdateAIConfig)
	configGroup.GET("/ai/prompts", m.configController.GetAIPromptTags)
	configGroup.GET("/storage", m.configController.GetStorageConfig)
	configGroup.PUT("/storage", m.configController.UpdateStorageConfig)
	configGroup.GET("/backup/dirs", m.configController.GetBackupDirs)
	configGroup.GET("/backup", m.configController.BackupData)
	configGroup.GET("/storage-path", m.configController.GetStoragePath)
	configGroup.PUT("/storage-path", m.configController.UpdateStoragePath)

	m.settingController.RegisterRoutes(routes.AdminAPI)
}
