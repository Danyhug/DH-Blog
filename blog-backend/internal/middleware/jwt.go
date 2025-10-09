package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"dh-blog/internal/response"
	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware JWT中间件，用于拦截越权请求
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)

		// 如果没有找到token，返回未授权错误
		if tokenString == "" {
			c.Set("isLogin", false)
			response.FailWithCode(c, http.StatusUnauthorized, "请求未携带token，无权限访问")
			c.Abort()
			return
		}

		token, err := utils.ParseToken(tokenString)
		if err != nil {
			c.Set("isLogin", false)
			response.FailWithCode(c, http.StatusUnauthorized, "无效的Token")
			c.Abort()
			return
		}

		setJWTContext(c, token)
		c.Next()
	}
}

// ValidLoginMiddleware 验证登录中间件，用于检查是否已经登录
func ValidLoginMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)

		token, err := utils.ParseToken(tokenString)
		if err == nil {
			// 验证成功
			fmt.Println("验证成功")
			setJWTContext(c, token)
		} else {
			c.Set("isLogin", false)
		}

		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return authHeader[7:]
		}
		return authHeader
	}

	return c.Query("token")
}

func setJWTContext(c *gin.Context, token *jwt.Token) {
	c.Set("isLogin", true)
	c.Set("jwtToken", token)

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		c.Set("jwtClaims", claims)
	}
}
