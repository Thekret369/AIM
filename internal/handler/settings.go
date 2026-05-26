// Package handler 实现 HTTP 请求处理
package handler

import (
	"net/http"

	"LanLine/internal/model"
	"LanLine/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingsHandler 处理用户设置相关请求
type SettingsHandler struct {
	Svc *service.SettingsService
}

// Get 返回当前用户的偏好设置
func (h *SettingsHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	settings, err := h.Svc.GetSettings(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

// Update 全量更新用户偏好设置
func (h *SettingsHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input model.UserSettings
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	settings, err := h.Svc.UpdateSettings(userID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}
