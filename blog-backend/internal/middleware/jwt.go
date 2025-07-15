package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"dh-blog/internal/response"
	"dh-blog/internal/utils"
	"github.com/gin-gonic/gin"
)

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			response.FailWithCode(c, http.StatusUnauthorized, "请求未携带token，无权限访问")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			fmt.Println(parts)
			response.FailWithCode(c, http.StatusUnauthorized, "请求头中auth格式有误")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1])
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
