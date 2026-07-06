package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"dh-blog/internal/config"
	"dh-blog/internal/controller"
	"dh-blog/internal/modules"
	"dh-blog/internal/repository"
	"dh-blog/internal/router"
	"dh-blog/internal/service"
	"dh-blog/internal/task"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// App 持有应用级依赖和生命周期组件。
type App struct {
	Config          *config.Config
	DB              *gorm.DB
	Router          *gin.Engine
	StaticFilesPath string
	TaskManager     *task.TaskManager
	CacheService    *service.CacheService
}

// New 初始化应用依赖、业务模块和路由。
func New(conf *config.Config, db *gorm.DB) *App {
	staticFilesPath := resolveStaticFilesPath()

	cacheService := service.NewCacheService()
	cache := cacheService.GetCache()
	logrus.Info("缓存服务初始化完成")

	userRepo := repository.NewUserRepository(db)
	tagRepo := repository.NewTagRepository(db, cache)
	categoryRepo := repository.NewCategoryRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	logRepo := repository.NewLogRepository(db, cache)
	articleRepo := repository.NewArticleRepository(db, categoryRepo, tagRepo, cache)
	systemSettingRepo := repository.NewSystemSettingRepository(db, cache)
	fileRepo := repository.NewFileRepository(db)
	shareRepo := repository.NewShareRepository(db)
	shareAccessLogRepo := repository.NewShareAccessLogRepository(db)

	aiService := service.NewAIService(systemSettingRepo, cache)
	ipService := service.NewIPService(logRepo)
	fileService := service.NewFileService(fileRepo, systemSettingRepo)
	if err := fileService.EnsureProtectedDirectories(context.Background()); err != nil {
		logrus.Warnf("创建固定目录失败: %v", err)
	}
	shareService := service.NewShareService(shareRepo, shareAccessLogRepo, fileService)
	configService := service.NewConfigService(systemSettingRepo)

	taskManager := task.NewTaskManager(db, aiService, tagRepo)
	taskManager.Start()

	logrus.Info("应用程序核心组件初始化完成")

	articleController := controller.NewArticleController(articleRepo, tagRepo, categoryRepo, commentRepo, aiService, taskManager)
	userController := controller.NewUserController(userRepo)
	commentController := controller.NewCommentController(commentRepo)
	logController := controller.NewLogController(logRepo)
	adminController := controller.NewAdminController(fileService, aiService)
	systemConfigController := controller.NewSystemConfigController(systemSettingRepo, db, fileService)
	fileController := controller.NewFileController(fileService)
	shareController := controller.NewShareController(shareService)
	systemSettingController := controller.NewSystemSettingController(systemSettingRepo, db)
	chunkUploadController := controller.NewChunkUploadController(fileController.GetFileService(), db, configService)
	webDAVController := controller.NewWebDAVController(userRepo, fileService, conf.WebDAVServer.Prefix)
	routeModules := []router.Module{
		modules.NewArticleModule(articleController),
		modules.NewUserModule(userController),
		modules.NewCommentModule(commentController),
		modules.NewAdminModule(adminController),
		modules.NewLoggingModule(logController),
		modules.NewSystemModule(systemConfigController, systemSettingController),
		modules.NewFilesModule(fileController, chunkUploadController, staticFilesPath, fileService),
		modules.NewShareModule(shareController),
		modules.NewWebDAVModule(conf.WebDAVServer.Enabled, conf.WebDAVServer.Prefix, webDAVController),
	}

	engine := router.Init(router.Options{
		Config:    conf,
		IPService: ipService,
	}, routeModules...)

	return &App{
		Config:          conf,
		DB:              db,
		Router:          engine,
		StaticFilesPath: staticFilesPath,
		TaskManager:     taskManager,
		CacheService:    cacheService,
	}
}

// Shutdown 停止应用持有的后台组件。
func (a *App) Shutdown() {
	if a.TaskManager != nil {
		a.TaskManager.Stop()
	}
	if a.CacheService != nil {
		a.CacheService.Shutdown()
	}
}

func resolveStaticFilesPath() string {
	exePath, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("获取可执行文件路径失败: %w", err))
	}

	dataDir := filepath.Join(filepath.Dir(exePath), "data")
	return filepath.Join(dataDir, "upload")
}
