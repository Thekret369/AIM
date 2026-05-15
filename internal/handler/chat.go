package handler

import (
	"net/http"
	"strconv"

	"AIM/internal/middleware"
	"AIM/internal/service"
	"AIM/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	Svc        *service.ChatService
	Hub        *ws.Hub
	JWTSecret  string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 开发阶段允许跨域
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleWS 升级 WebSocket 连接
// 浏览器 WebSocket API 不支持自定义 Header，token 通过查询参数 ?token=xxx 传递
func (h *ChatHandler) HandleWS(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 token"})
		return
	}

	claims, err := middleware.ParseToken(tokenStr, h.JWTSecret)
	if err != nil || claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token 无效或已过期"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(claims.UserID, claims.Username, h.Hub, conn)
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
	afterID, _ := strconv.ParseUint(c.Query("after_id"), 10, 64)

	msgs, total, err := h.Svc.GetHistory(userID, uint(peerID), page, pageSize, uint(afterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	info := h.Svc.GetReadInfo(userID, uint(peerID), nil)
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "total": total, "read_info": info})
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
	afterID, _ := strconv.ParseUint(c.Query("after_id"), 10, 64)

	msgs, total, err := h.Svc.GetGroupHistory(userID, uint(groupID), page, pageSize, uint(afterID))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	gid := uint(groupID)
	info := h.Svc.GetReadInfo(userID, 0, &gid)
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "total": total, "read_info": info})
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

// GetGroupReads 获取群内各用户的最后已读消息 ID，用于前端重建已读扇形图
func (h *ChatHandler) GetGroupReads(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	reads, err := h.Svc.GetGroupReads(uint(groupID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reads": reads})
}
