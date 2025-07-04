package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var JwtConfig struct {
	JwtSecret string
}

// InitJwtUtils 初始化 JWT 工具
func InitJwtUtils(secret string) {
	JwtConfig.JwtSecret = secret
}

// GenerateToken 生成 JWT Token
func GenerateToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":        time.Now().Add(time.Hour * 24).Unix(), // Token 有效期 24 小时
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(JwtConfig.JwtSecret))
	if err != nil {
		return "", fmt.Errorf("生成 Token 失败: %w", err)
	}

	return tokenString, nil
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 检查签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非法的签名方法: %v", token.Header["alg"])
		}
		return []byte(JwtConfig.JwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析 Token 失败: %w", err)
	}

	return token, nil
}

// CheckPasswordHash 验证密码哈希
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT 是 GenerateToken 的别名，为了兼容旧代码
func GenerateJWT(username string) (string, error) {
	return GenerateToken(username)
}

// HashPassword 对密码进行哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("哈希密码失败: %w", err)
	}
	return string(bytes), nil
}
