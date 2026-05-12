// Package ws 实现 WebSocket 连接管理（Hub + Client 模式）
// Hub 为全局连接中心，负责用户连接的注册、注销和消息路由
package ws

import (
	"encoding/json"
	"AIM/internal/model"
	"sync"

	"github.com/gorilla/websocket"
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
	// OnMessage 接收客户端发来的消息，由 ChatService 消费
	OnMessage chan *model.Message
}

// NewHub 创建 Hub 实例并启动主循环
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[uint]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		OnMessage:  make(chan *model.Message, 256),
	}
	go h.run()
	return h
}

// run Hub 主循环，处理连接注册与注销
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; !ok {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.clients, client.UserID)
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

// SendToUser 向指定用户的全部连接推送消息
func (h *Hub) SendToUser(userID uint, msg *model.Message) {
	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	data, _ := json.Marshal(msg)
	for client := range clients {
		select {
		case client.send <- data:
		default:
			// 发送缓冲区满则跳过
		}
	}
}

// SendToUsers 向一组用户批量推送（群聊时使用）
func (h *Hub) SendToUsers(userIDs []uint, msg *model.Message) {
	data, _ := json.Marshal(msg)
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
	data, _ := json.Marshal(msg)
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
