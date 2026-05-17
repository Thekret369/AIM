package handler

import (
	"net/http"
	"strconv"

	"AIM/internal/service"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	Svc *service.AIService
}

type aiBotCreateReq struct {
	Name         string  `json:"name" binding:"required"`
	Avatar       string  `json:"avatar"`
	BaseURL      string  `json:"base_url" binding:"required"`
	APIKey       string  `json:"api_key"`
	Model        string  `json:"model" binding:"required"`
	SystemPrompt string  `json:"system_prompt"`
	ContextLimit int     `json:"context_limit"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
}

type aiBotUpdateReq struct {
	Name         *string  `json:"name"`
	Avatar       *string  `json:"avatar"`
	BaseURL      *string  `json:"base_url"`
	APIKey       *string  `json:"api_key"`
	Model        *string  `json:"model"`
	SystemPrompt *string  `json:"system_prompt"`
	ContextLimit *int     `json:"context_limit"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int     `json:"max_tokens"`
	Status       *string  `json:"status"`
}

func (h *AIHandler) ListBots(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}

	bots, err := h.Svc.ListUserBots(c.GetUint("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bots": bots})
}

func (h *AIHandler) CreateBot(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}

	var req aiBotCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	bot, err := h.Svc.CreateUserBot(c.GetUint("user_id"), service.AIBotInput{
		Name:         req.Name,
		Avatar:       req.Avatar,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		ContextLimit: req.ContextLimit,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"bot": bot})
}

func (h *AIHandler) UpdateBot(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}

	botID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || botID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 AI 助手 ID"})
		return
	}

	var req aiBotUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	bot, err := h.Svc.UpdateUserBot(c.GetUint("user_id"), uint(botID), service.AIBotUpdateInput{
		Name:         req.Name,
		Avatar:       req.Avatar,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		ContextLimit: req.ContextLimit,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		Status:       req.Status,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bot": bot})
}

func (h *AIHandler) DeleteBot(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}

	botID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || botID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 AI 助手 ID"})
		return
	}
	if err := h.Svc.DeleteUserBot(c.GetUint("user_id"), uint(botID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "AI 助手已删除"})
}
