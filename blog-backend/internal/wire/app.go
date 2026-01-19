package wire

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"dh-blog/internal/config"
	"dh-blog/internal/handler"
	"dh-blog/internal/repository"
	"dh-blog/internal/router"
	"dh-blog/internal/service"
	"dh-blog/internal/task"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// App 应用程序
type App struct {
	Config          *config.Config
	DB              *gorm.DB
	Router          *gin.Engine
	Handlers        []handler.Handler
	StaticFilesPath string
	TaskManager     *task.TaskManager
	CacheService    *service.CacheService // 新增：缓存服务
}

// AppOption 应用程序选项
type AppOption func(*App)

// InitApp 初始化整个应用程序的依赖
func InitApp(conf *config.Config, db *gorm.DB) *gin.Engine {
	// 获取 data 目录的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("获取可执行文件路径失败: %w", err))
	}
	dataDir := filepath.Join(filepath.Dir(exePath), "data")
	staticFilesAbsPath := filepath.Join(dataDir, "upload")

	// 初始化缓存服务
	cacheService := service.NewCacheService()
	cache := cacheService.GetCache()
	logrus.Info("缓存服务初始化完成")

	// 初始化存储库
	userRepo := repository.NewUserRepository(db)
	tagRepo := repository.NewTagRepository(db, cache)
	categoryRepo := repository.NewCategoryRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	logRepo := repository.NewLogRepository(db, cache)
	articleRepo := repository.NewArticleRepository(db, categoryRepo, tagRepo, cache)
	systemSettingRepo := repository.NewSystemSettingRepository(db, cache)
	// 添加文件存储库
	fileRepo := repository.NewFileRepository(db)
	// 添加分享存储库
	shareRepo := repository.NewShareRepository(db)
	shareAccessLogRepo := repository.NewShareAccessLogRepository(db)

	// 初始化服务
	aiService := service.NewAIService(systemSettingRepo, cacheService.GetCache())
	ipService := service.NewIPService(logRepo)
	// 添加文件服务（需要在 uploadService 之前初始化）
	fileService := service.NewFileService(fileRepo, systemSettingRepo)
	// 确保固定目录存在（音乐、图片、视频）
	if err := fileService.EnsureProtectedDirectories(context.Background()); err != nil {
		logrus.Warnf("创建固定目录失败: %v", err)
	}
	// 初始化上传服务（依赖 fileService）
	uploadService := service.NewUploadService(conf, dataDir, fileService)
	// 添加分享服务
	shareService := service.NewShareService(shareRepo, shareAccessLogRepo, fileService)
	// 初始化配置服务
	configService := service.NewConfigService(systemSettingRepo)

	// 初始化任务管理器
	taskManager := task.NewTaskManager(db, aiService, tagRepo)

	// 启动任务管理器
	taskManager.Start()

	// 记录初始化信息
	logrus.Info("应用程序核心组件初始化完成")

	// 初始化处理器
	articleHandler := handler.NewArticleHandler(articleRepo, tagRepo, categoryRepo, aiService, taskManager)
	userHandler := handler.NewUserHandler(userRepo)
	commentHandler := handler.NewCommentHandler(commentRepo)
	logHandler := handler.NewLogHandler(logRepo)
	adminHandler := handler.NewAdminHandler(uploadService, aiService)
	systemConfigHandler := handler.NewSystemConfigHandler(systemSettingRepo, db, fileService)
	// 添加文件处理器
	fileHandler := handler.NewFileHandler(fileService)
	// 添加分享处理器
	shareHandler := handler.NewShareHandler(shareService)
	// 添加系统设置处理器
	systemSettingHandler := handler.NewSystemSettingHandler(systemSettingRepo, db)
	chunkUploadHandler := handler.NewChunkUploadHandler(fileHandler.GetFileService(), db, configService)

	return router.Init(
		articleHandler,
		userHandler,
		commentHandler,
		logHandler,
		adminHandler,
		systemConfigHandler,
		fileHandler,
		shareHandler,         // 添加分享处理器
		systemSettingHandler, // 添加系统设置处理器
		ipService,
		staticFilesAbsPath,
		chunkUploadHandler,
		conf,        // 添加配置参数
		fileService, // 添加文件服务参数（用于博客图片公开访问）
	)
}
