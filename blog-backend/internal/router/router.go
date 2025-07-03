package router

import (
	"dh-blog/internal/handler"
	"github.com/gin-gonic/gin"
)

// Init 初始化并配置 Gin 路由器
func Init(articleHandler *handler.ArticleHandler) *gin.Engine {
	// 初始化 Gin 路由器
	router := gin.Default()

	// 健康检查路由
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// API V1 路由组
	apiV1 := router.Group("/api/v1")
	{
		// 文章相关的路由
		apiV1.GET("/articles/:id", articleHandler.GetArticleDetail)
	}

	return router
}
