package router

import (
	"time"

	"dh-blog/internal/config"
	"dh-blog/internal/frontend"
	"dh-blog/internal/middleware"
	"dh-blog/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Module 表示一个可注册 HTTP 路由的业务模块。
type Module interface {
	RegisterRoutes(*Routes)
}

// Options 是初始化路由器所需的全局依赖。
type Options struct {
	Config    *config.Config
	IPService service.IPService
}

// Routes 汇总模块注册路由时可用的 Gin 分组。
type Routes struct {
	Engine    *gin.Engine
	PublicAPI *gin.RouterGroup
	AdminAPI  *gin.RouterGroup
	FileAPI   *gin.RouterGroup
}

// Init 初始化 Gin 路由器并挂载业务模块。
func Init(options Options, modules ...Module) *gin.Engine {
	engine := gin.Default()

	// 配置 CORS 中间件
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有源，生产环境请限制
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 添加 IP 中间件
	engine.Use(middleware.IPMiddleware(options.IPService), middleware.ValidLoginMiddleware())

	routes := &Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
		FileAPI:   engine.Group("/api/files"),
	}

	routes.AdminAPI.Use(middleware.JWTMiddleware())
	routes.FileAPI.Use(middleware.JWTMiddleware())

	for _, module := range modules {
		module.RegisterRoutes(routes)
	}

	// 注册前端静态文件路由
	frontend.RegisterFrontendRoutes(engine, options.Config)
	logrus.Info("前端静态文件路由已注册")

	return engine
}
