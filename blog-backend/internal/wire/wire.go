package wire

import (
	"dh-blog/internal/config"
	"dh-blog/internal/database"
	"dh-blog/internal/handler"
	"dh-blog/internal/repository"
	"dh-blog/internal/router"
	"github.com/gin-gonic/gin"
)

// InitApp 初始化整个应用程序的依赖
func InitApp(conf *config.Config) *gin.Engine {
	// 1. 初始化数据库
	db := database.Init(conf)

	// 2. 初始化 Repository 层
	articleRepo := repository.NewArticleRepository(db)

	// 3. 初始化 Handler 层，直接注入 Repository
	articleHandler := handler.NewArticleHandler(articleRepo)

	// 4. 初始化路由器并注册路由
	appRouter := router.Init(articleHandler)

	return appRouter
}
