// Package handler 实现 HTTP 请求处理
package handler

import (
	"net/http"

	"AIM/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	Svc *service.AuthService
}

// RegisterReq 注册请求体
type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=128"`
	Nickname string `json:"nickname"`
}

// LoginReq 登录请求体
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	user, err := h.Svc.Register(req.Username, req.Password, req.Nickname)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	token, user, err := h.Svc.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 种 Cookie，解决浏览器页面导航不携带 Authorization header 的问题
	c.SetCookie("aim_token", token, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// Logout 退出登录，清除 Cookie
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("aim_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "已退出"})
}
