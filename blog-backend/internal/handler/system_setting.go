package handler

import (
	"net/http"
	"strconv"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SystemSettingResponse 用于返回小写字段名的JSON响应
type SystemSettingResponse struct {
	ID           uint   `json:"id"`
	SettingKey   string `json:"settingKey"`
	SettingValue string `json:"settingValue"`
	ConfigType   string `json:"configType"`
}

// 将model.SystemSetting转换为SystemSettingResponse
func toSystemSettingResponse(setting model.SystemSetting) SystemSettingResponse {
	return SystemSettingResponse{
		ID:           setting.ID,
		SettingKey:   setting.SettingKey,
		SettingValue: setting.SettingValue,
		ConfigType:   setting.ConfigType,
	}
}

// 将多个model.SystemSetting转换为多个SystemSettingResponse
func toSystemSettingResponses(settings []model.SystemSetting) []SystemSettingResponse {
	responses := make([]SystemSettingResponse, len(settings))
	for i, setting := range settings {
		responses[i] = toSystemSettingResponse(setting)
	}
	return responses
}

type SystemSettingHandler struct {
	BaseHandler
	settingRepo repository.SystemSettingRepository
	db          *gorm.DB
}

func NewSystemSettingHandler(settingRepo repository.SystemSettingRepository, db *gorm.DB) *SystemSettingHandler {
	return &SystemSettingHandler{settingRepo: settingRepo, db: db}
}

// RegisterRoutes 注册路由
func (h *SystemSettingHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/system-setting")
	{
		group.GET("/list", h.List)
		group.POST("", h.Add)
		group.PUT("", h.Update)
		group.DELETE(":id", h.Delete)
	}
}

// List 获取所有系统配置项
func (h *SystemSettingHandler) List(c *gin.Context) {
	settings, err := h.settingRepo.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("获取失败"))
		return
	}
	// 转换为小写字段名的响应
	responseData := toSystemSettingResponses(settings)
	c.JSON(http.StatusOK, response.SuccessWithData(responseData))
}

// Add 新增配置项
func (h *SystemSettingHandler) Add(c *gin.Context) {
	var req struct {
		SettingKey   string `json:"settingKey" binding:"required"`
		SettingValue string `json:"settingValue" binding:"required"`
		ConfigType   string `json:"configType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("参数错误"))
		return
	}
	err := h.settingRepo.UpdateSettingWithType(req.SettingKey, req.SettingValue, req.ConfigType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("新增失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

// Update 更新配置项
func (h *SystemSettingHandler) Update(c *gin.Context) {
	var req struct {
		ID           uint   `json:"id" binding:"required"`
		SettingKey   string `json:"settingKey" binding:"required"`
		SettingValue string `json:"settingValue" binding:"required"`
		ConfigType   string `json:"configType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("参数错误"))
		return
	}
	// 查找原始key
	var setting model.SystemSetting
	if err := h.db.First(&setting, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, response.Error("配置不存在"))
		return
	}
	err := h.settingRepo.UpdateSettingWithType(req.SettingKey, req.SettingValue, req.ConfigType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("更新失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

// Delete 删除配置项
func (h *SystemSettingHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error("参数错误"))
		return
	}
	var setting model.SystemSetting
	if err := h.db.First(&setting, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, response.Error("配置不存在"))
		return
	}
	if err := h.db.Delete(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("删除失败"))
		return
	}
	h.settingRepo.ClearCache()
	c.JSON(http.StatusOK, response.Success())
}
