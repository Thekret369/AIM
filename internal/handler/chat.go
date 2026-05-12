package handler

import (
	"net/http"
	"strconv"

	"AIM/internal/service"
	"AIM/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	Svc *service.ChatService
	Hub *ws.Hub
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 开发阶段允许跨域
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleWS 升级 WebSocket 连接，认证通过后注册到 Hub
func (h *ChatHandler) HandleWS(c *gin.Context) {
	userID := c.GetUint("user_id")
	username := c.GetString("username")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(userID, username, h.Hub, conn)
	client.Start()
}

// GetHistory 获取单聊历史消息
func (h *ChatHandler) GetHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	peerID, err := strconv.ParseUint(c.Query("peer_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 peer_id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	msgs, total, err := h.Svc.GetHistory(userID, uint(peerID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "total": total})
}

// GetGroupHistory 获取群聊历史消息
func (h *ChatHandler) GetGroupHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	msgs, total, err := h.Svc.GetGroupHistory(userID, uint(groupID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "total": total})
}

// GetBroadcastHistory 获取广播消息历史
func (h *ChatHandler) GetBroadcastHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	msgs, total, err := h.Svc.GetBroadcastHistory(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "total": total})
}
