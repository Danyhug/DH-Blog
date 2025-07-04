package router

import (
	"time"

	"dh-blog/internal/handler"
	"dh-blog/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Init 初始化并配置 Gin 路由器
func Init(articleHandler *handler.ArticleHandler, userHandler *handler.UserHandler, commentHandler *handler.CommentHandler, logHandler *handler.LogHandler, adminHandler *handler.AdminHandler) *gin.Engine {
	// 初始化 Gin 路由器
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
		// 文章公共 API (对应 Java ArticleController)
		publicAPI.GET("/article/:id", articleHandler.GetArticleDetail)
		publicAPI.GET("/article/title/:id", articleHandler.GetArticleTitle)
		publicAPI.GET("/article/unlock/:id/:password", articleHandler.UnlockArticle)
		publicAPI.POST("/article/list", articleHandler.GetArticleList) // Java 使用 POST /article/list
		publicAPI.GET("/article/overview", articleHandler.GetOverview)
		publicAPI.GET("/article/tag", articleHandler.GetAllTags)
		publicAPI.GET("/article/category", articleHandler.GetAllCategories)

		// 用户公共 API (对应 Java UserController)
		publicAPI.POST("/user/login", userHandler.Login)
		publicAPI.POST("/user/check", userHandler.Check)
		publicAPI.GET("/user/heart", userHandler.Heart)

		// 评论公共 API (对应 Java CommentController)
		publicAPI.POST("/comment", commentHandler.AddComment)
		publicAPI.GET("/comment/:articleId", commentHandler.GetCommentsByArticleID)
	}

	// 管理 API 路由组
	adminAPI := router.Group("/api/admin")
	adminAPI.Use(middleware.JWTMiddleware())
	{
		// 文章管理 API (对应 Java AdminController)
		adminAPI.GET("/article/:id", articleHandler.GetArticleDetail) // 与公共 API 重复，但路径不同
		adminAPI.POST("/article", articleHandler.SaveArticle)
		adminAPI.PUT("/article", articleHandler.UpdateArticle)
		adminAPI.POST("/article/list", articleHandler.GetArticleList) // 与公共 API 重复，但路径不同
		adminAPI.POST("/upload", adminHandler.UploadFile)

		// 标签管理 API (对应 Java AdminController)
		adminAPI.POST("/tag", articleHandler.CreateTag)
		adminAPI.PUT("/tag", articleHandler.UpdateTag)
		adminAPI.DELETE("/tag/:id", articleHandler.DeleteTag)

		// 分类管理 API (对应 Java AdminController)
		adminAPI.POST("/category", articleHandler.CreateCategory)
		adminAPI.GET("/category/:id", articleHandler.GetCategoryByID)
		adminAPI.PUT("/category", articleHandler.UpdateCategory)
		adminAPI.DELETE("/category/:id", articleHandler.DeleteCategory)
		adminAPI.GET("/category/:id/tags", articleHandler.GetCategoryDefaultTags)
		adminAPI.POST("/category/:id/default-tags", articleHandler.SaveCategoryDefaultTags)

		// 评论管理 API (对应 Java AdminController)
		adminAPI.GET("/comment/:pageSize/:pageNum", commentHandler.GetAllComments)
		adminAPI.PUT("/comment", commentHandler.UpdateComment)
		adminAPI.POST("/comment/reply", commentHandler.ReplyComment)
		adminAPI.DELETE("/comment/:id", commentHandler.DeleteComment)

		// 日志管理 API (对应 Java LogController)
		adminAPI.GET("/log/overview/visitLog", logHandler.GetVisitLogs)
		adminAPI.POST("/ip/ban/:ip/:status", logHandler.BanIP)
	}

	return router
}
