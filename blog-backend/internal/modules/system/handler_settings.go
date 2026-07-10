package system

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func toSettingResponse(setting Setting) settingResponse {
	return settingResponse{setting.ID, setting.SettingKey, setting.SettingValue, setting.ConfigType}
}
func (h *handler) listSettings(c *gin.Context) {
	settings, err := h.service.settings.all(c.Request.Context())
	if err != nil {
		failure(c, 500, err)
		return
	}
	result := make([]settingResponse, len(settings))
	for i, setting := range settings {
		result[i] = toSettingResponse(setting)
	}
	success(c, result)
}
func (h *handler) addSetting(c *gin.Context) {
	var request struct {
		SettingKey   string `json:"settingKey" binding:"required"`
		SettingValue string `json:"settingValue" binding:"required"`
		ConfigType   string `json:"configType"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		failure(c, 400, err)
		return
	}
	if err := h.service.settings.update(c.Request.Context(), request.SettingKey, request.SettingValue, request.ConfigType); err != nil {
		failure(c, 500, err)
		return
	}
	success(c)
}
func (h *handler) updateSetting(c *gin.Context) {
	var request struct {
		ID           uint   `json:"id" binding:"required"`
		SettingKey   string `json:"settingKey" binding:"required"`
		SettingValue string `json:"settingValue" binding:"required"`
		ConfigType   string `json:"configType"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		failure(c, 400, err)
		return
	}
	if err := h.service.settings.updateID(c.Request.Context(), request.ID, request.SettingKey, request.SettingValue, request.ConfigType); errors.Is(err, gorm.ErrRecordNotFound) {
		failure(c, 404, fmt.Errorf("配置不存在"))
		return
	} else if err != nil {
		failure(c, 500, err)
		return
	}
	success(c)
}
func (h *handler) deleteSetting(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		failure(c, 400, fmt.Errorf("参数错误"))
		return
	}
	if err := h.service.settings.delete(c.Request.Context(), uint(id)); errors.Is(err, gorm.ErrRecordNotFound) {
		failure(c, 404, fmt.Errorf("配置不存在"))
		return
	} else if err != nil {
		failure(c, 500, err)
		return
	}
	success(c)
}

type backupDirInfo struct {
	Name        string `json:"name"`
	IsProtected bool   `json:"is_protected"`
}
