// Package middleware 提供 JWT 鉴权中间件（含 TokenVersion 校验以顶出旧登录）
package middleware

import (
	"net/http"
	"strings"

	"LanLine/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT Claims，TokenVersion 用于踢出并发登录的旧 session
type Claims struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"`
	jwt.RegisteredClaims
}

// AuthRequired JWT 鉴权中间件
// 校验 token 签名 + 有效期 + Token 版本号
// 版本号与 DB 不一致时拒绝（账号已在别处登录）
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证令牌"})
			c.Abort()
			return
		}

		claims, err := ParseToken(tokenStr, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌无效或已过期"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// ParseToken 解析 JWT 并校验签名 + 版本号，供中间件和 WebSocket handler 共用
func ParseToken(tokenStr string, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	// 校验 TokenVersion：与 DB 不一致说明账号已在别处重新登录
	var user model.User
	if err := model.DB.First(&user, claims.UserID).Error; err != nil {
		return nil, err
	}
	if claims.TokenVersion != user.TokenVersion {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// extractToken 从 Authorization Header 提取 token
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}
