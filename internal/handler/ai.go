package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"AIM/internal/service"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	Svc *service.AIService
}

type aiBotCreateReq struct {
	Name             string          `json:"name" binding:"required"`
	Avatar           string          `json:"avatar"`
	APISource        string          `json:"api_source"`
	BaseURL          string          `json:"base_url"`
	APIKey           string          `json:"api_key"`
	Model            string          `json:"model"`
	SystemPrompt     string          `json:"system_prompt"`
	ContextLimit     int             `json:"context_limit"`
	Temperature      float64         `json:"temperature"`
	MaxTokens        int             `json:"max_tokens"`
	KnowledgeBaseIDs []uint          `json:"knowledge_base_ids"`
	PluginConfig     json.RawMessage `json:"plugin_config"`
}

type aiBotUpdateReq struct {
	Name             *string          `json:"name"`
	Avatar           *string          `json:"avatar"`
	APISource        *string          `json:"api_source"`
	BaseURL          *string          `json:"base_url"`
	APIKey           *string          `json:"api_key"`
	Model            *string          `json:"model"`
	SystemPrompt     *string          `json:"system_prompt"`
	ContextLimit     *int             `json:"context_limit"`
	Temperature      *float64         `json:"temperature"`
	MaxTokens        *int             `json:"max_tokens"`
	Status           *string          `json:"status"`
	KnowledgeBaseIDs *[]uint          `json:"knowledge_base_ids"`
	PluginConfig     *json.RawMessage `json:"plugin_config"`
}

type aiKnowledgeBaseReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type aiKnowledgeDocumentReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type aiContextResetReq struct {
	BotUserID uint  `json:"bot_user_id" binding:"required"`
	GroupID   *uint `json:"group_id"`
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
		Name:             req.Name,
		Avatar:           req.Avatar,
		APISource:        req.APISource,
		BaseURL:          req.BaseURL,
		APIKey:           req.APIKey,
		Model:            req.Model,
		SystemPrompt:     req.SystemPrompt,
		ContextLimit:     req.ContextLimit,
		Temperature:      req.Temperature,
		MaxTokens:        req.MaxTokens,
		KnowledgeBaseIDs: req.KnowledgeBaseIDs,
		PluginConfig:     req.PluginConfig,
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
		Name:             req.Name,
		Avatar:           req.Avatar,
		APISource:        req.APISource,
		BaseURL:          req.BaseURL,
		APIKey:           req.APIKey,
		Model:            req.Model,
		SystemPrompt:     req.SystemPrompt,
		ContextLimit:     req.ContextLimit,
		Temperature:      req.Temperature,
		MaxTokens:        req.MaxTokens,
		Status:           req.Status,
		KnowledgeBaseIDs: req.KnowledgeBaseIDs,
		PluginConfig:     req.PluginConfig,
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

func (h *AIHandler) ResetContext(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}

	var req aiContextResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	reset, err := h.Svc.ResetContext(c.GetUint("user_id"), service.AIContextResetInput{
		BotUserID: req.BotUserID,
		GroupID:   req.GroupID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "蓝妹上下文已清除", "reset": reset})
}

func (h *AIHandler) ListTokenUsages(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	page, pageSize := parsePage(c)
	usages, total, err := h.Svc.ListTokenUsages(c.GetUint("user_id"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"usages": usages, "total": total, "page": page, "page_size": pageSize})
}

func (h *AIHandler) ListKnowledgeBases(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	bases, err := h.Svc.ListKnowledgeBases(c.GetUint("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"knowledge_bases": bases})
}

func (h *AIHandler) CreateKnowledgeBase(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	var req aiKnowledgeBaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	base, err := h.Svc.CreateKnowledgeBase(c.GetUint("user_id"), service.AIKnowledgeBaseInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"knowledge_base": base})
}

func (h *AIHandler) UpdateKnowledgeBase(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	baseID, ok := parseUintParam(c, "id", "无效的知识库 ID")
	if !ok {
		return
	}
	var req aiKnowledgeBaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	base, err := h.Svc.UpdateKnowledgeBase(c.GetUint("user_id"), uint(baseID), service.AIKnowledgeBaseInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"knowledge_base": base})
}

func (h *AIHandler) DeleteKnowledgeBase(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	baseID, ok := parseUintParam(c, "id", "无效的知识库 ID")
	if !ok {
		return
	}
	if err := h.Svc.DeleteKnowledgeBase(c.GetUint("user_id"), uint(baseID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "知识库已删除"})
}

func (h *AIHandler) AddKnowledgeDocument(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	baseID, ok := parseUintParam(c, "id", "无效的知识库 ID")
	if !ok {
		return
	}
	var req aiKnowledgeDocumentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	doc, err := h.Svc.AddKnowledgeDocument(c.GetUint("user_id"), uint(baseID), service.AIKnowledgeDocumentInput{
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"document": doc})
}

func (h *AIHandler) UpdateKnowledgeDocument(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	docID, ok := parseUintParam(c, "id", "无效的知识库文档 ID")
	if !ok {
		return
	}
	var req aiKnowledgeDocumentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	doc, err := h.Svc.UpdateKnowledgeDocument(c.GetUint("user_id"), uint(docID), service.AIKnowledgeDocumentInput{
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"document": doc})
}

func (h *AIHandler) DeleteKnowledgeDocument(c *gin.Context) {
	if h.Svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 功能未启用"})
		return
	}
	docID, ok := parseUintParam(c, "id", "无效的知识库文档 ID")
	if !ok {
		return
	}
	if err := h.Svc.DeleteKnowledgeDocument(c.GetUint("user_id"), uint(docID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "知识库文档已删除"})
}

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return page, pageSize
}

func parseUintParam(c *gin.Context, name, errMsg string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return 0, false
	}
	return id, true
}
