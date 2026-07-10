package system

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *handler) getStorageConfig(c *gin.Context) {
	config, err := h.service.configByType(c.Request.Context(), ConfigTypeStorage)
	if err != nil {
		failure(c, 500, err)
		return
	}
	success(c, StorageConfig{FileStoragePath: config.FileStoragePath, WebDAVChunkSize: config.WebDAVChunkSize})
}
func (h *handler) updateStorageConfig(c *gin.Context) {
	var config StorageConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		failure(c, 400, err)
		return
	}
	if err := h.applyStorage(c.Request.Context(), config); err != nil {
		failure(c, 400, err)
		return
	}
	success(c, "存储路径已更新，文件表已清空并重新扫描")
}
func (h *handler) applyStorage(ctx context.Context, config StorageConfig) error {
	if err := validateStorageConfig(config); err != nil {
		return err
	}
	oldConfig, err := h.service.configByType(ctx, ConfigTypeStorage)
	if err != nil {
		return err
	}
	old := StorageConfig{FileStoragePath: oldConfig.FileStoragePath, WebDAVChunkSize: oldConfig.WebDAVChunkSize}
	values := map[string]string{SettingKeyFileStoragePath: config.FileStoragePath, SettingKeyWebDAVChunkSize: strconv.Itoa(config.WebDAVChunkSize)}
	if err := h.service.settings.updateBatch(ctx, values, ConfigTypeStorage); err != nil {
		return err
	}
	if err := h.storage.ApplyStorageConfig(ctx, config.FileStoragePath, config.WebDAVChunkSize); err != nil {
		rollback := map[string]string{SettingKeyFileStoragePath: old.FileStoragePath, SettingKeyWebDAVChunkSize: strconv.Itoa(old.WebDAVChunkSize)}
		if rollbackErr := h.service.settings.updateBatch(context.Background(), rollback, ConfigTypeStorage); rollbackErr != nil {
			return fmt.Errorf("应用存储配置失败: %v；回滚设置失败: %w", err, rollbackErr)
		}
		return fmt.Errorf("应用存储配置失败，设置已回滚: %w", err)
	}
	return nil
}

func validateStorageConfig(config StorageConfig) error {
	if config.FileStoragePath == "" {
		return fmt.Errorf("存储路径不能为空")
	}
	if config.WebDAVChunkSize <= 0 {
		return fmt.Errorf("WebDAV 分片大小必须大于 0")
	}
	info, err := os.Stat(config.FileStoragePath)
	if err != nil {
		return fmt.Errorf("存储路径不可用: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("存储路径不是目录: %s", config.FileStoragePath)
	}
	return nil
}

func (h *handler) getStoragePath(c *gin.Context) {
	success(c, gin.H{"path": h.storage.GetStoragePath()})
}
func (h *handler) updateStoragePath(c *gin.Context) {
	userID, ok := c.Get("userID")
	if !ok || !isAdminUserID(userID) {
		failure(c, http.StatusUnauthorized, fmt.Errorf("只有管理员可以更新存储路径"))
		return
	}
	var request struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		failure(c, 400, err)
		return
	}
	config, err := h.service.configByType(c.Request.Context(), ConfigTypeStorage)
	if err != nil {
		failure(c, 500, err)
		return
	}
	if err := h.applyStorage(c.Request.Context(), StorageConfig{FileStoragePath: request.Path, WebDAVChunkSize: config.WebDAVChunkSize}); err != nil {
		failure(c, 400, err)
		return
	}
	success(c, "存储路径已更新，文件表已清空并重新扫描")
}

func isAdminUserID(value any) bool {
	switch id := value.(type) {
	case uint64:
		return id == 1
	case uint:
		return id == 1
	case int:
		return id == 1
	case string:
		return id == "1"
	default:
		return false
	}
}

type settingResponse struct {
	ID           uint   `json:"id"`
	SettingKey   string `json:"settingKey"`
	SettingValue string `json:"settingValue"`
	ConfigType   string `json:"configType"`
}
