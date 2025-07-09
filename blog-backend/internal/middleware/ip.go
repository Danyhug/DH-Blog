package middleware

import (
	"dh-blog/internal/utils"
	"github.com/gin-gonic/gin"
)

// IPMiddleware 客户端 IP 中间件
func IPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端IP
		ip := utils.GetClientIP(c.Request)

		// 将 IP 设置到上下文中，以便后续处理使用
		c.Set("client_ip", ip)

		// 继续执行下一个处理器
		c.Next()
	}
}
