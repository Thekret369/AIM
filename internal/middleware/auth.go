// Package middleware 提供 JWT 鉴权中间件
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT Claims
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthRequired JWT 鉴权中间件
// 从 Authorization: Bearer <token> 提取并验证 token
// 验证通过后将 user_id 和 username 写入 gin.Context
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
		// 浏览器页面导航不带 Header，从 Cookie 回退取 token
		if tokenStr == "" {
			if cookie, err := c.Cookie("aim_token"); err == nil {
				tokenStr = cookie
			}
		}
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证令牌"})
			c.Abort()
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌无效或已过期"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
