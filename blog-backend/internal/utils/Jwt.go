package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JwtConfig JWT 配置结构
type JwtConfig struct {
	Secret string
}

var jwtConfig *JwtConfig

// InitJwtUtils 初始化 JWT 工具
func InitJwtUtils(secret string) {
	jwtConfig = &JwtConfig{
		Secret: secret,
	}
}

// UserClaims 自定义 JWT claims 结构
type UserClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// User 用户结构体
type User struct {
	Username string
}

// CreateToken 生成 JWT token
func CreateToken(user *User) (string, error) {
	if jwtConfig == nil {
		return "", errors.New("JWT utils not initialized")
	}

	// 设置过期时间为7天后
	expireTime := time.Now().Add(7 * 24 * time.Hour)

	claims := UserClaims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.Secret))
}

// ParseToken 解析 JWT token
func ParseToken(tokenString string) (*User, error) {
	if jwtConfig == nil {
		return nil, errors.New("JWT utils not initialized")
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return &User{
			Username: claims.Username,
		}, nil
	}

	return nil, errors.New("invalid token")
}

// EncodeByBCrypt 使用 BCrypt 加密字符串
func EncodeByBCrypt(str string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyByBCrypt 验证 BCrypt 加密的字符串
func VerifyByBCrypt(str, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(str))
	return err == nil
}
