package frontend

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"dh-blog/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//go:embed all:dist
var distFS embed.FS

// RegisterFrontendRoutes 注册前端静态文件路由
func RegisterFrontendRoutes(router *gin.Engine, conf *config.Config) {
	// 检查嵌入的文件系统
	dist := distFS
	// 在开发模式下，如果嵌入的文件系统为空，尝试从文件系统加载
	if files, err := dist.ReadDir("."); err != nil || len(files) == 0 {
		// 如果嵌入失败，尝试从文件系统加载（开发模式）
		if _, statErr := os.Stat("internal/frontend/dist"); statErr == nil {
			logrus.Info("使用文件系统模式加载前端文件")
			router.Static("/", "internal/frontend/dist")
			return
		}

		logrus.Warn("无法加载前端文件")
		return
	}

	// 为assets目录创建带缓存控制的静态文件服务
	assetsFS, err := fs.Sub(distFS, "dist/assets")
	if err == nil {
		// 使用带缓存控制的处理器
		router.Use(cacheControlMiddleware())
		router.StaticFS("/assets", http.FS(assetsFS))
	}

	// 为其他静态文件创建带缓存控制的处理函数
	staticFiles := map[string]string{
		"/vite.svg":      "dist/vite.svg",
		"/favicon.ico":   "dist/favicon.ico",
		"/robots.txt":    "dist/robots.txt",
		"/manifest.json": "dist/manifest.json",
	}

	for route, filePath := range staticFiles {
		route := route // 创建局部变量
		filePath := filePath
		router.GET(route, func(c *gin.Context) {
			serveCachedFile(c, filePath, "public, max-age=86400") // 24小时缓存
		})
	}

	// 处理首页和前端路由
	indexHandler := func(c *gin.Context) {
		// 读取index.html文件
		content, err := fs.ReadFile(distFS, "dist/index.html")
		if err != nil {
			logrus.Errorf("读取index.html失败: %v", err)
			c.String(http.StatusInternalServerError, "无法读取index.html文件")
			return
		}

		// 注入服务器配置，使用相对路径避免硬编码localhost
		serverConfig := fmt.Sprintf(`
<script>
window.__SERVER_CONFIG__ = {
  SERVER_URL: "%s"
};
</script>`, "/api")

		htmlContent := strings.Replace(string(content), "</head>", serverConfig+"</head>", 1)

		// 设置不缓存的HTTP头
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Header("Content-Type", "text/html; charset=utf-8")

		// 返回修改后的HTML
		c.String(http.StatusOK, htmlContent)
	}

	// 注册路由
	router.GET("/", indexHandler)
	router.GET("/index.html", indexHandler)

	// 处理API请求和其他路由
	router.NoRoute(func(c *gin.Context) {
		// 先尝试API路由
		path := c.Request.URL.Path
		if len(path) >= 4 && strings.HasPrefix(path, "/api") {
			c.Next()
			return
		}

		// 对于前端路由，返回index.html
		indexHandler(c)
	})

	logrus.Info("前端静态文件路由注册成功（已启用缓存优化）")
}

// cacheControlMiddleware 为静态文件添加缓存控制头的中间件
func cacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只为assets目录下的文件添加缓存控制
		if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			// 为构建后的JS/CSS文件设置长期缓存（1年），因为这些文件有内容哈希
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	}
}

// serveCachedFile 服务单个带缓存控制的文件
func serveCachedFile(c *gin.Context, filePath string, cacheControl string) {
	content, err := fs.ReadFile(distFS, filePath)
	if err != nil {
		c.String(http.StatusNotFound, "文件不存在")
		return
	}

	// 设置缓存控制头
	c.Header("Cache-Control", cacheControl)
	
	// 根据文件扩展名设置正确的Content-Type
	contentType := getContentType(filePath)
	c.Data(http.StatusOK, contentType, content)
}

// getContentType 根据文件扩展名获取Content-Type
func getContentType(filePath string) string {
	ext := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(ext, ".js"):
		return "application/javascript"
	case strings.HasSuffix(ext, ".css"):
		return "text/css"
	case strings.HasSuffix(ext, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(ext, ".png"):
		return "image/png"
	case strings.HasSuffix(ext, ".jpg"), strings.HasSuffix(ext, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(ext, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(ext, ".json"):
		return "application/json"
	case strings.HasSuffix(ext, ".txt"):
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
