package handler

import (
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"github.com/gin-gonic/gin"
)

// SystemConfigHandler 系统配置处理器
type SystemConfigHandler struct {
	BaseHandler
	settingRepo repository.SystemSettingRepository
}

// NewSystemConfigHandler 创建系统配置处理器
func NewSystemConfigHandler(settingRepo repository.SystemSettingRepository) *SystemConfigHandler {
	return &SystemConfigHandler{
		settingRepo: settingRepo,
	}
}

// RegisterRoutes 注册路由
func (h *SystemConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	configGroup := router.Group("/config")
	{
		configGroup.GET("", h.GetConfigs)
		configGroup.PUT("", h.UpdateConfigs)
	}
}

// GetConfigs 获取所有配置
func (h *SystemConfigHandler) GetConfigs(c *gin.Context) {
	settings, err := h.settingRepo.GetAllSettings()
	if err != nil {
		h.Error(c, err)
		return
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// 使用新的映射方法创建配置对象
	config := model.FromSettingsMap(settingsMap)
	h.SuccessWithData(c, config)
}

// UpdateConfigs 更新配置
func (h *SystemConfigHandler) UpdateConfigs(c *gin.Context) {
	var config model.SystemConfig
	if err := h.BindJSON(c, &config); err != nil {
		h.Error(c, err)
		return
	}

	// 使用新的映射方法获取设置映射
	settingsMap := config.ToSettingsMap()
	if err := h.settingRepo.BatchUpdateSettings(settingsMap); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c)
}

// BindJSON 绑定JSON数据
func (h *SystemConfigHandler) BindJSON(c *gin.Context, obj interface{}) error {
	return c.ShouldBindJSON(obj)
}
