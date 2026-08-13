package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type TokenParser interface {
	ParseToken(token string) (*jwt.Token, error)
}

// JWTMiddleware JWT中间件，用于拦截越权请求
func JWTMiddleware(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)

		// 如果没有找到token，返回未授权错误
		if tokenString == "" {
			c.Set("isLogin", false)
			response.FailWithCode(c, http.StatusUnauthorized, "请求未携带token，无权限访问")
			c.Abort()
			return
		}

		if parser == nil {
			c.Set("isLogin", false)
			response.FailWithCode(c, http.StatusUnauthorized, "Token 服务未配置")
			c.Abort()
			return
		}

		token, err := parser.ParseToken(tokenString)
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
func ValidLoginMiddleware(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)

		if parser == nil {
			c.Set("isLogin", false)
			c.Next()
			return
		}
		token, err := parser.ParseToken(tokenString)
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

// extractToken reads the token from the Authorization header, falling back to
// the query string for transports that cannot set headers — file downloads and
// the admin event WebSocket, since the browser's WebSocket API has no way to
// send one.
//
// Both sources are normalised the same way. The frontend stores the token with
// the "Bearer " prefix the login endpoint returns, so a query carrying it
// verbatim used to fail while the identical header succeeded; call sites papered
// over that by stripping the prefix themselves, one copy at a time.
func extractToken(c *gin.Context) string {
	if authHeader := c.Request.Header.Get("Authorization"); authHeader != "" {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return strings.TrimPrefix(c.Query("token"), "Bearer ")
}

func setJWTContext(c *gin.Context, token *jwt.Token) {
	c.Set("isLogin", true)
	c.Set("jwtToken", token)

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		c.Set("jwtClaims", claims)

		if rawID, exists := claims["userID"]; exists {
			switch v := rawID.(type) {
			case float64:
				c.Set("userID", uint64(v))
			case string:
				if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
					c.Set("userID", parsed)
				}
			}
		}
	}

	if _, exists := c.Get("userID"); !exists {
		c.Set("userID", uint64(1))
	}
}
