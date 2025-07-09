package wire

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dh-blog/internal/config"
	"dh-blog/internal/handler"
	"dh-blog/internal/repository"
	"dh-blog/internal/router"
	"dh-blog/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App 应用程序
type App struct {
	Config          *config.Config
	DB              *gorm.DB
	Router          *gin.Engine
	Handlers        []handler.Handler
	StaticFilesPath string
}

// AppOption 应用程序选项
type AppOption func(*App)

// Init 初始化应用程序
func (a *App) Init() *gin.Engine {
	// 配置 CORS 中间件
	a.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有源，生产环境请限制
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 注册处理器路由
	publicAPI := a.Router.Group("/api")

	// 注册所有处理器的公共路由
	for _, h := range a.Handlers {
		h.RegisterRoutes(publicAPI)
	}

	// 开放静态文件服务
	if a.StaticFilesPath != "" {
		publicAPI.Static("/uploads", a.StaticFilesPath)
		fmt.Printf("静态文件服务路径: /uploads -> %s\n", a.StaticFilesPath)
	}

	return a.Router
}

// InitApp 初始化整个应用程序的依赖
func InitApp(conf *config.Config, db *gorm.DB) *gin.Engine {
	// 获取 data 目录的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("获取可执行文件路径失败: %w", err))
	}
	dataDir := filepath.Join(filepath.Dir(exePath), "data")
	staticFilesAbsPath := filepath.Join(dataDir, "upload")

	// 初始化存储库
	userRepo := repository.NewUserRepository(db)
	tagRepo := repository.NewTagRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	logRepo := repository.NewLogRepository(db)
	articleRepo := repository.NewArticleRepository(db, categoryRepo, tagRepo)
	systemSettingRepo := repository.NewSystemSettingRepository(db)

	// 初始化服务
	uploadService := service.NewUploadService(conf, dataDir)
	aiService := service.NewAIService(systemSettingRepo)
	ipService := service.NewIPService(logRepo)

	// 初始化处理器
	articleHandler := handler.NewArticleHandler(articleRepo, tagRepo, categoryRepo)
	userHandler := handler.NewUserHandler(userRepo)
	commentHandler := handler.NewCommentHandler(commentRepo)
	logHandler := handler.NewLogHandler(logRepo)
	adminHandler := handler.NewAdminHandler(uploadService, aiService)
	systemConfigHandler := handler.NewSystemConfigHandler(systemSettingRepo)

	return router.Init(articleHandler, userHandler, commentHandler, logHandler, adminHandler, systemConfigHandler, ipService, staticFilesAbsPath)
}
