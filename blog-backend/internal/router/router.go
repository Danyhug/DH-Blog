package router

import (
	"fmt"
	"time"

	"dh-blog/internal/handler"
	"dh-blog/internal/middleware"
	"dh-blog/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Init 初始化并配置 Gin 路由器
func Init(
	articleHandler *handler.ArticleHandler,
	userHandler *handler.UserHandler,
	commentHandler *handler.CommentHandler,
	logHandler *handler.LogHandler,
	adminHandler *handler.AdminHandler,
	systemConfigHandler *handler.SystemConfigHandler,
	fileHandler *handler.FileHandler,
	ipService service.IPService,
	staticFilesAbsPath string,
) *gin.Engine {

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

	// 添加 IP 中间件
	router.Use(middleware.IPMiddleware(ipService))

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
		adminAPI.GET("/stats/daily", logHandler.GetDailyStats)
		adminAPI.GET("/log/stats/visits", logHandler.GetVisitStatistics)                 // 添加访问统计接口
		adminAPI.GET("/log/stats/monthly", logHandler.GetMonthlyVisitStats)              // 添加月度访问统计接口
		adminAPI.GET("/log/stats/daily-chart", logHandler.GetDailyVisitStatsForLastDays) // 添加每日图表统计接口

		// IP封禁API - 与前端请求格式一致
		adminAPI.POST("/ip/ban/:ip/:status", logHandler.BanIP)

		// 使用SystemConfigHandler的RegisterRoutes方法注册系统配置相关路由
		logrus.Info("注册系统配置路由")

		// 添加系统配置路由
		configGroup := adminAPI.Group("/config")
		{
			// 全局配置接口
			configGroup.GET("", systemConfigHandler.GetConfigs)
			configGroup.PUT("", systemConfigHandler.UpdateConfigs)

			// 博客基本配置接口
			configGroup.GET("/blog", systemConfigHandler.GetBlogConfig)
			configGroup.PUT("/blog", systemConfigHandler.UpdateBlogConfig)

			// 邮件配置接口
			configGroup.GET("/email", systemConfigHandler.GetEmailConfig)
			configGroup.PUT("/email", systemConfigHandler.UpdateEmailConfig)

			// AI配置接口
			configGroup.GET("/ai", systemConfigHandler.GetAIConfig)
			configGroup.PUT("/ai", systemConfigHandler.UpdateAIConfig)

			// 存储配置接口
			configGroup.GET("/storage", systemConfigHandler.GetStorageConfig)
			configGroup.PUT("/storage", systemConfigHandler.UpdateStorageConfig)

			// 兼容旧版API
			configGroup.GET("/storage-path", systemConfigHandler.GetStoragePath)
			configGroup.PUT("/storage-path", systemConfigHandler.UpdateStoragePath)
		}
	}

	fileApi := router.Group("/api/files")
	fileApi.Use(middleware.JWTMiddleware())
	{
		fileApi.GET("/list", fileHandler.ListFiles)
		fileApi.POST("/upload", fileHandler.UploadFile)
		fileApi.POST("/folder", fileHandler.CreateFolder)
		fileApi.GET("/download/:id", fileHandler.DownloadFile)
		fileApi.PUT("/rename/:id", fileHandler.RenameFile)
		fileApi.DELETE("/:id", fileHandler.DeleteFile)
		fileApi.PUT("/storage-path", fileHandler.UpdateStoragePath) // 添加更新存储路径路由
	}

	// 开放静态文件服务
	publicAPI.Static("/uploads", staticFilesAbsPath)
	fmt.Printf("静态文件服务路径: /uploads -> %s\n", staticFilesAbsPath)

	return router
}
