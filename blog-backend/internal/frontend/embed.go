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

	// 为assets目录创建静态文件服务
	assetsFS, err := fs.Sub(distFS, "dist/assets")
	if err == nil {
		router.StaticFS("/assets", http.FS(assetsFS))
	}

	// 为其他静态文件创建处理函数
	router.GET("/vite.svg", func(c *gin.Context) {
		content, err := fs.ReadFile(distFS, "dist/vite.svg")
		if err != nil {
			c.String(http.StatusNotFound, "文件不存在")
			return
		}
		c.Data(http.StatusOK, "image/svg+xml", content)
	})

	// 处理首页和前端路由
	indexHandler := func(c *gin.Context) {
		// 读取index.html文件
		content, err := fs.ReadFile(distFS, "dist/index.html")
		if err != nil {
			logrus.Errorf("读取index.html失败: %v", err)
			c.String(http.StatusInternalServerError, "无法读取index.html文件")
			return
		}

		// 注入服务器配置
		serverPort := conf.Server.HttpPort
		serverAddress := conf.Server.Address
		if serverAddress == "0.0.0.0" {
			serverAddress = "localhost"
		}

		serverConfig := fmt.Sprintf(`
<script>
window.__SERVER_CONFIG__ = {
  SERVER_URL: "http://%s:%d/api"
};
</script>`, serverAddress, serverPort)

		htmlContent := strings.Replace(string(content), "</head>", serverConfig+"</head>", 1)

		// 设置Content-Type
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

	logrus.Info("前端静态文件路由注册成功")
}
