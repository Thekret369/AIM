// Package ws 实现 WebSocket 连接管理（Hub + Client 模式）
// Hub 为全局连接中心，负责用户连接的注册、注销和消息路由
// 同时管理在线状态广播、输入状态和已读回执路由
package ws

import (
	"encoding/json"
	"AIM/internal/model"
	"sync"
)

// Hub 管理所有活跃的 WebSocket 连接
// 以用户 ID 为键，支持同一用户多端同时在线
type Hub struct {
	mu      sync.RWMutex
	// clients 用户ID -> 该用户的所有连接
	clients map[uint]map[*Client]bool
	// register 连接注册通道
	register chan *Client
	// unregister 连接注销通道
	unregister chan *Client
	// OnMessage 接收客户端发来的聊天消息，由 ChatService 消费
	OnMessage chan *model.Message
	// OnTyping 接收输入状态事件，由 ChatService 路由
	OnTyping chan *TypingPayload
	// OnReadReceipt 接收已读回执，由 ChatService 持久化并转发
	OnReadReceipt chan *ReadReceiptPayload
}

// NewHub 创建 Hub 实例并启动主循环
func NewHub() *Hub {
	h := &Hub{
		clients:      make(map[uint]map[*Client]bool),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		OnMessage:    make(chan *model.Message, 256),
		OnTyping:     make(chan *TypingPayload, 64),
		OnReadReceipt: make(chan *ReadReceiptPayload, 64),
	}
	go h.run()
	return h
}

// run Hub 主循环，处理连接注册、注销，并广播在线状态变更
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			wasOffline := len(h.clients[client.UserID]) == 0
			if _, ok := h.clients[client.UserID]; !ok {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()
			// 用户首个连接上线时广播在线状态
			if wasOffline {
				h.broadcastStatus(client.UserID, true)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.clients, client.UserID)
					// 用户最后一个连接断开时广播离线状态
					h.mu.Unlock()
					h.broadcastStatus(client.UserID, false)
					continue
				}
			}
			h.mu.Unlock()
		}
	}
}

// Register 将新连接注册到 Hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 从 Hub 移除连接
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// wrapChat 将聊天消息包裹为 WSMessage 格式
func wrapChat(msg *model.Message) json.RawMessage {
	w := WSMessage{
		Type:    WSMChat,
		Payload: mustMarshal(msg),
	}
	return mustMarshal(w)
}

// SendToUser 向指定用户的全部连接推送消息
func (h *Hub) SendToUser(userID uint, msg *model.Message) {
	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	data := wrapChat(msg)
	for client := range clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

// SendToUsers 向一组用户批量推送（群聊时使用）
func (h *Hub) SendToUsers(userIDs []uint, msg *model.Message) {
	data := wrapChat(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, uid := range userIDs {
		if clients, ok := h.clients[uid]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
				}
			}
		}
	}
}

// Broadcast 向所有在线用户广播消息
func (h *Hub) Broadcast(msg *model.Message) {
	data := wrapChat(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

// OnlineCount 返回当前在线用户数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// IsOnline 判断用户是否在线
func (h *Hub) IsOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// GetOnlineUserIDs 返回所有在线用户 ID 列表
func (h *Hub) GetOnlineUserIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uint, 0, len(h.clients))
	for uid := range h.clients {
		ids = append(ids, uid)
	}
	return ids
}

// SendTyping 向目标用户转发输入状态
func (h *Hub) SendTyping(toUserID uint, payload *TypingPayload) {
	h.mu.RLock()
	clients, ok := h.clients[toUserID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMTyping,
		Payload: mustMarshal(payload),
	})
	for client := range clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

// SendTypingToUsers 向群成员批量转发输入状态（排除发送者自己）
func (h *Hub) SendTypingToUsers(userIDs []uint, payload *TypingPayload) {
	data, _ := json.Marshal(WSMessage{
		Type:    WSMTyping,
		Payload: mustMarshal(payload),
	})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, uid := range userIDs {
		if uid == payload.FromUserID {
			continue
		}
		if clients, ok := h.clients[uid]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
				}
			}
		}
	}
}

// SendReadReceipt 向目标用户发送已读回执
func (h *Hub) SendReadReceipt(toUserID uint, payload *ReadReceiptPayload) {
	h.mu.RLock()
	clients, ok := h.clients[toUserID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMReadReceipt,
		Payload: mustMarshal(payload),
	})
	for client := range clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

// broadcastStatus 向所有在线用户广播某用户的上/下线状态
func (h *Hub) broadcastStatus(userID uint, online bool) {
	payload := StatusPayload{UserID: userID, IsOnline: online}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMStatus,
		Payload: mustMarshal(&payload),
	})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

// mustMarshal 序列化为 JSON，忽略错误（内部使用，数据可控）
func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
