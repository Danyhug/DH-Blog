package middleware

import (
	"net/http"
	"strings"

	"dh-blog/internal/response"
	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
)

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取token
		authHeader := c.Request.Header.Get("Authorization")
		var tokenString string

		if authHeader != "" {
			// 检查是否已经包含Bearer前缀
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = authHeader[7:] // 去掉"Bearer "前缀
			} else {
				tokenString = authHeader // 直接使用
			}
		}

		// 如果请求头中没有token，尝试从URL参数获取
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		// 如果没有找到token，返回未授权错误
		if tokenString == "" {
			response.FailWithCode(c, http.StatusUnauthorized, "请求未携带token，无权限访问")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			response.FailWithCode(c, http.StatusUnauthorized, "无效的Token")
			c.Abort()
			return
		}

		// 将当前请求的claims信息保存到请求的上下文c上
		c.Set("claims", claims)
		c.Set("userID", uint64(1))
		c.Next()
	}
}
