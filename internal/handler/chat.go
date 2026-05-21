package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"AIM/internal/middleware"
	"AIM/internal/model"
	"AIM/internal/service"
	"AIM/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	Svc       *service.ChatService
	Hub       *ws.Hub
	JWTSecret string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return strings.EqualFold(u.Host, r.Host)
	},
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

// RecallMessage 撤回 2 分钟内由当前用户发送的消息。
func (h *ChatHandler) RecallMessage(c *gin.Context) {
	userID := c.GetUint("user_id")
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || messageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的消息 ID"})
		return
	}

	msg, err := h.Svc.RecallMessage(userID, uint(messageID))
	if err != nil {
		c.JSON(recallMessageStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func recallMessageStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrMessageNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrMessageRecallForbidden):
		return http.StatusForbidden
	default:
		return http.StatusBadRequest
	}
}

// SearchMessages 搜索当前用户可访问范围内的消息
func (h *ChatHandler) SearchMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	scope := strings.TrimSpace(c.DefaultQuery("type", "user"))
	keyword := strings.TrimSpace(c.Query("q"))
	startTime, err := parseSearchTime(c.Query("start_time"), false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 start_time"})
		return
	}
	endTime, err := parseSearchTime(c.Query("end_time"), true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 end_time"})
		return
	}
	if keyword == "" && startTime == nil && endTime == nil {
		c.JSON(http.StatusOK, gin.H{"messages": []interface{}{}, "total": 0})
		return
	}

	var targetID uint
	if scope == "user" || scope == "group" {
		id, err := strconv.ParseUint(c.Query("target_id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 target_id"})
			return
		}
		targetID = uint(id)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	msgs, total, err := h.Svc.SearchMessagesWithParams(userID, service.MessageSearchParams{
		Scope:     scope,
		TargetID:  targetID,
		Keyword:   keyword,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		status := http.StatusBadRequest
		if scope == "group" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	resp := gin.H{"messages": msgs, "total": total}
	if strings.EqualFold(c.Query("group_by"), "conversation") {
		resp["conversations"] = groupMessagesByConversation(userID, msgs)
	}
	c.JSON(http.StatusOK, resp)
}

func parseSearchTime(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}
	layouts := []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		return &t, nil
	}
	return nil, errors.New("invalid time")
}

type searchConversationGroup struct {
	ConversationType string          `json:"conversation_type"`
	ConversationID   uint            `json:"conversation_id"`
	Count            int             `json:"count"`
	Messages         []model.Message `json:"messages"`
}

func groupMessagesByConversation(userID uint, msgs []model.Message) []searchConversationGroup {
	groups := make([]searchConversationGroup, 0)
	index := make(map[string]int)
	for _, msg := range msgs {
		convType, convID := messageConversation(userID, msg)
		key := convType + ":" + strconv.FormatUint(uint64(convID), 10)
		pos, ok := index[key]
		if !ok {
			pos = len(groups)
			index[key] = pos
			groups = append(groups, searchConversationGroup{
				ConversationType: convType,
				ConversationID:   convID,
				Messages:         []model.Message{},
			})
		}
		groups[pos].Messages = append(groups[pos].Messages, msg)
		groups[pos].Count++
	}
	return groups
}

func messageConversation(userID uint, msg model.Message) (string, uint) {
	if msg.GroupID != nil && *msg.GroupID > 0 {
		return model.ConversationGroup, *msg.GroupID
	}
	if msg.ToUserID != nil && *msg.ToUserID > 0 {
		if msg.FromUserID == userID {
			return model.ConversationUser, *msg.ToUserID
		}
		return model.ConversationUser, msg.FromUserID
	}
	return "broadcast", 0
}

// GetGroupReads 获取群内各用户的最后已读消息 ID，用于前端重建已读扇形图
func (h *ChatHandler) GetGroupReads(c *gin.Context) {
	userID := c.GetUint("user_id")
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组 ID"})
		return
	}
	reads, err := h.Svc.GetGroupReads(userID, uint(groupID))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reads": reads})
}
