package wire

import (
	"dh-blog/internal/config"
	"dh-blog/internal/handler"
	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"dh-blog/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"path/filepath"
	"os"
	"fmt"
	"time"
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

// WithConfig 设置配置
func WithConfig(conf *config.Config) AppOption {
	return func(app *App) {
		app.Config = conf
	}
}

// WithDB 设置数据库
func WithDB(db *gorm.DB) AppOption {
	return func(app *App) {
		app.DB = db
	}
}

// WithHandler 添加处理器
func WithHandler(h handler.Handler) AppOption {
	return func(app *App) {
		app.Handlers = append(app.Handlers, h)
	}
}

// WithStaticFilesPath 设置静态文件路径
func WithStaticFilesPath(path string) AppOption {
	return func(app *App) {
		app.StaticFilesPath = path
	}
}

// NewApp 创建应用程序
func NewApp(options ...AppOption) *App {
	app := &App{
		Router:   gin.Default(),
		Handlers: make([]handler.Handler, 0),
	}

	for _, option := range options {
		option(app)
	}

	return app
}

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
	
	// 初始化处理器
	articleHandler := handler.NewArticleHandler(articleRepo, tagRepo, categoryRepo)
	userHandler := handler.NewUserHandler(userRepo)
	commentHandler := handler.NewCommentHandler(commentRepo)
	logHandler := handler.NewLogHandler(logRepo)
	adminHandler := handler.NewAdminHandler(uploadService, aiService)
	systemConfigHandler := handler.NewSystemConfigHandler(systemSettingRepo)

	// 使用原始的路由配置
	router := gin.Default()

	// 配置 CORS 中间件
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有源，生产环境请限制
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 公共 API 路由组
	publicAPI := router.Group("/api")
	{
		// 文章公共 API
		publicAPI.GET("/article/:id", articleHandler.GetArticleDetail)
		publicAPI.GET("/article/title/:id", articleHandler.GetArticleTitle)
		publicAPI.GET("/article/unlock/:id/:password", articleHandler.UnlockArticle)
		publicAPI.POST("/article/list", articleHandler.GetArticleList)
		publicAPI.GET("/article/overview", articleHandler.GetOverview)
		publicAPI.GET("/article/tag", articleHandler.GetAllTags)
		publicAPI.GET("/article/category", articleHandler.GetAllCategories)

		// 用户公共 API
		publicAPI.POST("/user/login", userHandler.Login)
		publicAPI.POST("/user/check", userHandler.Check)
		publicAPI.GET("/user/heart", userHandler.Heart)

		// 评论公共 API
		publicAPI.POST("/comment", commentHandler.AddComment)
		publicAPI.GET("/comment/:articleId", commentHandler.GetCommentsByArticleID)
	}

	// 管理 API 路由组
	adminAPI := router.Group("/api/admin")
	adminAPI.Use(middleware.JWTMiddleware())
	{
		// 文章管理 API
		adminAPI.GET("/article/:id", articleHandler.GetArticleDetail)
		adminAPI.POST("/article", articleHandler.SaveArticle)
		adminAPI.PUT("/article", articleHandler.UpdateArticle)
		adminAPI.POST("/article/list", articleHandler.GetArticleList)
		adminAPI.POST("/upload/:type", adminHandler.UploadFile)

		// 标签管理 API
		adminAPI.POST("/tag", articleHandler.CreateTag)
		adminAPI.PUT("/tag", articleHandler.UpdateTag)
		adminAPI.DELETE("/tag/:id", articleHandler.DeleteTag)

		// 分类管理 API
		adminAPI.POST("/category", articleHandler.CreateCategory)
		adminAPI.GET("/category/:id", articleHandler.GetCategoryByID)
		adminAPI.PUT("/category", articleHandler.UpdateCategory)
		adminAPI.DELETE("/category/:id", articleHandler.DeleteCategory)
		adminAPI.GET("/category/:id/tags", articleHandler.GetCategoryDefaultTags)
		adminAPI.POST("/category/:id/tags", articleHandler.SaveCategoryDefaultTags)

		// 评论管理 API
		adminAPI.GET("/comment/:pageSize/:pageNum", commentHandler.GetAllComments)
		adminAPI.PUT("/comment", commentHandler.UpdateComment)
		adminAPI.POST("/comment/reply", commentHandler.ReplyComment)
		adminAPI.DELETE("/comment/:id", commentHandler.DeleteComment)

		// 日志管理 API
		adminAPI.GET("/log/overview/visitLog", logHandler.GetVisitLogs)
		adminAPI.POST("/ip/ban", logHandler.BanIP)
		adminAPI.POST("/ip/unban", logHandler.UnbanIP)
		adminAPI.GET("/stats/daily", logHandler.GetDailyStats)

		// 系统配置 API
		adminAPI.GET("/config", systemConfigHandler.GetConfigs)
		adminAPI.PUT("/config", systemConfigHandler.UpdateConfigs)
	}

	// 开放静态文件服务
	publicAPI.Static("/uploads", staticFilesAbsPath)
	fmt.Printf("静态文件服务路径: /uploads -> %s\n", staticFilesAbsPath)
	
	return router
} 