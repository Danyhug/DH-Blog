package handler

import (
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SystemConfigHandler 系统配置处理器
type SystemConfigHandler struct {
	BaseHandler
	settingRepo repository.SystemSettingRepository
	db          *gorm.DB
	fileService service.IFileService // 添加文件服务
}

// NewSystemConfigHandler 创建系统配置处理器
func NewSystemConfigHandler(settingRepo repository.SystemSettingRepository, db *gorm.DB, fileService service.IFileService) *SystemConfigHandler {
	return &SystemConfigHandler{
		settingRepo: settingRepo,
		db:          db,
		fileService: fileService,
	}
}

// RegisterRoutes 注册路由
func (h *SystemConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	configGroup := router.Group("/config")
	{
		// 全局配置接口
		configGroup.GET("", h.GetConfigs)
		configGroup.PUT("", h.UpdateConfigs)

		// 博客基本配置接口
		configGroup.GET("/blog", h.GetBlogConfig)
		configGroup.PUT("/blog", h.UpdateBlogConfig)

		// 邮件配置接口
		configGroup.GET("/email", h.GetEmailConfig)
		configGroup.PUT("/email", h.UpdateEmailConfig)

		// AI配置接口
		configGroup.GET("/ai", h.GetAIConfig)
		configGroup.PUT("/ai", h.UpdateAIConfig)

		// 存储配置接口
		configGroup.GET("/storage", h.GetStorageConfig)
		configGroup.PUT("/storage", h.UpdateStorageConfig)

		// 兼容旧版API
		configGroup.GET("/storage-path", h.GetStoragePath)
		configGroup.PUT("/storage-path", h.UpdateStoragePath)
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

// GetBlogConfig 获取博客基本配置
func (h *SystemConfigHandler) GetBlogConfig(c *gin.Context) {
	// 获取博客类型的配置
	settings, err := h.settingRepo.GetSettingsByType(model.ConfigTypeBlog)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// 从全局设置中提取博客配置
	allConfig := model.FromSettingsMap(settingsMap)
	blogConfig := allConfig.GetBlogConfig()

	h.SuccessWithData(c, blogConfig)
}

// UpdateBlogConfig 更新博客基本配置
func (h *SystemConfigHandler) UpdateBlogConfig(c *gin.Context) {
	var blogConfig model.BlogConfig
	if err := h.BindJSON(c, &blogConfig); err != nil {
		h.Error(c, err)
		return
	}

	// 转换为配置映射
	settings := map[string]string{
		"blog_title":    blogConfig.BlogTitle,
		"signature":     blogConfig.Signature,
		"avatar":        blogConfig.Avatar,
		"github_link":   blogConfig.GithubLink,
		"bilibili_link": blogConfig.BilibiliLink,
		"open_blog":     boolToString(blogConfig.OpenBlog),
		"open_comment":  boolToString(blogConfig.OpenComment),
	}

	// 使用带类型的批量更新
	if err := h.settingRepo.BatchUpdateSettingsWithType(settings, model.ConfigTypeBlog); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c)
}

// GetEmailConfig 获取邮件配置
func (h *SystemConfigHandler) GetEmailConfig(c *gin.Context) {
	// 获取邮件类型的配置
	settings, err := h.settingRepo.GetSettingsByType(model.ConfigTypeEmail)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// 从全局设置中提取邮件配置
	allConfig := model.FromSettingsMap(settingsMap)
	emailConfig := allConfig.GetEmailConfig()

	h.SuccessWithData(c, emailConfig)
}

// UpdateEmailConfig 更新邮件配置
func (h *SystemConfigHandler) UpdateEmailConfig(c *gin.Context) {
	var emailConfig model.EmailConfig
	if err := h.BindJSON(c, &emailConfig); err != nil {
		h.Error(c, err)
		return
	}

	// 转换为配置映射
	settings := map[string]string{
		"comment_email_notify": boolToString(emailConfig.CommentEmailNotify),
		"smtp_host":            emailConfig.SmtpHost,
		"smtp_port":            intToString(emailConfig.SmtpPort),
		"smtp_user":            emailConfig.SmtpUser,
		"smtp_pass":            emailConfig.SmtpPass,
		"smtp_sender":          emailConfig.SmtpSender,
	}

	// 使用带类型的批量更新
	if err := h.settingRepo.BatchUpdateSettingsWithType(settings, model.ConfigTypeEmail); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c)
}

// GetAIConfig 获取AI配置
func (h *SystemConfigHandler) GetAIConfig(c *gin.Context) {
	// 获取AI类型的配置
	settings, err := h.settingRepo.GetSettingsByType(model.ConfigTypeAI)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// 从全局设置中提取AI配置
	allConfig := model.FromSettingsMap(settingsMap)
	aiConfig := allConfig.GetAIConfig()

	h.SuccessWithData(c, aiConfig)
}

// UpdateAIConfig 更新AI配置
func (h *SystemConfigHandler) UpdateAIConfig(c *gin.Context) {
	var aiConfig model.AIConfig
	if err := h.BindJSON(c, &aiConfig); err != nil {
		h.Error(c, err)
		return
	}

	// 转换为配置映射
	settings := map[string]string{
		"ai_api_url": aiConfig.AiApiURL,
		"ai_api_key": aiConfig.AiApiKey,
		"ai_model":   aiConfig.AiModel,
		"ai_prompt":  aiConfig.AiPrompt,
	}

	// 使用带类型的批量更新
	if err := h.settingRepo.BatchUpdateSettingsWithType(settings, model.ConfigTypeAI); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c)
}

// GetStorageConfig 获取存储配置
func (h *SystemConfigHandler) GetStorageConfig(c *gin.Context) {
	// 获取存储类型的配置
	settings, err := h.settingRepo.GetSettingsByType(model.ConfigTypeStorage)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// 从全局设置中提取存储配置
	allConfig := model.FromSettingsMap(settingsMap)
	storageConfig := allConfig.GetStorageConfig()

	h.SuccessWithData(c, storageConfig)
}

// UpdateStorageConfig 更新存储配置
func (h *SystemConfigHandler) UpdateStorageConfig(c *gin.Context) {
	var storageConfig model.StorageConfig
	if err := h.BindJSON(c, &storageConfig); err != nil {
		h.Error(c, err)
		return
	}

	// 验证路径是否存在
	if _, err := os.Stat(storageConfig.FileStoragePath); os.IsNotExist(err) {
		h.ErrorWithMessage(c, "存储路径不存在: "+err.Error())
		return
	}

	// 使用文件服务更新存储路径
	if err := h.fileService.UpdateStoragePath(storageConfig.FileStoragePath); err != nil {
		h.ErrorWithMessage(c, "更新存储路径失败: "+err.Error())
		return
	}

	// 转换为配置映射
	settings := map[string]string{
		"file_storage_path": storageConfig.FileStoragePath,
	}

	// 使用带类型的批量更新
	if err := h.settingRepo.BatchUpdateSettingsWithType(settings, model.ConfigTypeStorage); err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithMessage(c, "存储路径已更新，文件表已清空并重新扫描")
}

// GetStoragePath 获取文件存储路径 (兼容旧版API)
func (h *SystemConfigHandler) GetStoragePath(c *gin.Context) {
	path, err := h.settingRepo.GetSetting("file_storage_path")
	if err != nil {
		h.ErrorWithMessage(c, "获取文件存储路径失败: "+err.Error())
		return
	}

	h.SuccessWithData(c, gin.H{
		"path": path,
	})
}

// UpdateStoragePathRequest 更新存储路径请求结构
type UpdateStoragePathRequest struct {
	Path string `json:"path" binding:"required"`
}

// UpdateStoragePath 更新文件存储路径 (兼容旧版API)
func (h *SystemConfigHandler) UpdateStoragePath(c *gin.Context) {
	var req UpdateStoragePathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.ErrorWithMessage(c, "请求参数无效: "+err.Error())
		return
	}

	// 验证路径是否存在
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		h.ErrorWithMessage(c, "存储路径不存在: "+err.Error())
		return
	}

	// 使用文件服务更新存储路径
	if err := h.fileService.UpdateStoragePath(req.Path); err != nil {
		h.ErrorWithMessage(c, "更新存储路径失败: "+err.Error())
		return
	}

	// 清除设置缓存
	h.settingRepo.ClearCache()
	logrus.Infof("文件存储路径已更新为: %s", req.Path)

	h.SuccessWithMessage(c, "存储路径已更新，文件表已清空并重新扫描")
}

// BindJSON 绑定JSON数据
func (h *SystemConfigHandler) BindJSON(c *gin.Context, obj interface{}) error {
	return c.ShouldBindJSON(obj)
}

// 辅助函数：布尔值转字符串
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// 辅助函数：整数转字符串
func intToString(i int) string {
	return strconv.Itoa(i)
}
