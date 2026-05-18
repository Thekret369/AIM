package ws

import (
	"encoding/json"
	"sync"

	"AIM/internal/model"
)

// Hub manages active WebSocket connections. A user can have multiple clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[uint]map[*Client]bool

	register   chan *Client
	unregister chan *Client

	OnMessage     chan *model.Message
	OnTyping      chan *TypingPayload
	OnReadReceipt chan *ReadReceiptPayload
	OnUserOnline  chan uint
}

func NewHub() *Hub {
	h := &Hub{
		clients:       make(map[uint]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		OnMessage:     make(chan *model.Message, 256),
		OnTyping:      make(chan *TypingPayload, 64),
		OnReadReceipt: make(chan *ReadReceiptPayload, 64),
		OnUserOnline:  make(chan uint, 64),
	}
	go h.run()
	return h
}

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
			if wasOffline {
				h.broadcastStatus(client.UserID, true)
				h.notifyUserOnline(client.UserID)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.clients, client.UserID)
					h.mu.Unlock()
					h.broadcastStatus(client.UserID, false)
					continue
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) notifyUserOnline(userID uint) {
	select {
	case h.OnUserOnline <- userID:
	default:
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func wrapChat(msg *model.Message) json.RawMessage {
	w := WSMessage{
		Type:    WSMChat,
		Payload: mustMarshal(msg),
	}
	return mustMarshal(w)
}

func (h *Hub) snapshotUserClients(userID uint) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[userID]
	if !ok {
		return nil
	}
	result := make([]*Client, 0, len(clients))
	for client := range clients {
		result = append(result, client)
	}
	return result
}

func (h *Hub) snapshotUsersClients(userIDs []uint, skipUserID uint) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*Client, 0)
	seen := make(map[*Client]struct{})
	for _, uid := range userIDs {
		if skipUserID != 0 && uid == skipUserID {
			continue
		}
		clients, ok := h.clients[uid]
		if !ok {
			continue
		}
		for client := range clients {
			if _, exists := seen[client]; exists {
				continue
			}
			seen[client] = struct{}{}
			result = append(result, client)
		}
	}
	return result
}

func (h *Hub) snapshotAllClients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*Client, 0)
	for _, clients := range h.clients {
		for client := range clients {
			result = append(result, client)
		}
	}
	return result
}

func sendToClients(clients []*Client, data []byte) {
	for _, client := range clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

func (h *Hub) SendToUser(userID uint, msg *model.Message) {
	clients := h.snapshotUserClients(userID)
	if len(clients) == 0 {
		return
	}
	sendToClients(clients, wrapChat(msg))
}

func (h *Hub) SendToUsers(userIDs []uint, msg *model.Message) {
	clients := h.snapshotUsersClients(userIDs, 0)
	if len(clients) == 0 {
		return
	}
	sendToClients(clients, wrapChat(msg))
}

func (h *Hub) Broadcast(msg *model.Message) {
	clients := h.snapshotAllClients()
	if len(clients) == 0 {
		return
	}
	sendToClients(clients, wrapChat(msg))
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) IsOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

func (h *Hub) GetOnlineUserIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uint, 0, len(h.clients))
	for uid := range h.clients {
		ids = append(ids, uid)
	}
	return ids
}

func (h *Hub) SendTyping(toUserID uint, payload *TypingPayload) {
	clients := h.snapshotUserClients(toUserID)
	if len(clients) == 0 {
		return
	}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMTyping,
		Payload: mustMarshal(payload),
	})
	sendToClients(clients, data)
}

func (h *Hub) SendTypingToUsers(userIDs []uint, payload *TypingPayload) {
	clients := h.snapshotUsersClients(userIDs, payload.FromUserID)
	if len(clients) == 0 {
		return
	}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMTyping,
		Payload: mustMarshal(payload),
	})
	sendToClients(clients, data)
}

func (h *Hub) SendReadReceiptToUsers(userIDs []uint, payload *ReadReceiptPayload) {
	clients := h.snapshotUsersClients(userIDs, payload.FromUserID)
	if len(clients) == 0 {
		return
	}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMReadReceipt,
		Payload: mustMarshal(payload),
	})
	sendToClients(clients, data)
}

func (h *Hub) SendReadReceipt(toUserID uint, payload *ReadReceiptPayload) {
	clients := h.snapshotUserClients(toUserID)
	if len(clients) == 0 {
		return
	}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMReadReceipt,
		Payload: mustMarshal(payload),
	})
	sendToClients(clients, data)
}

func (h *Hub) broadcastStatus(userID uint, online bool) {
	clients := h.snapshotAllClients()
	if len(clients) == 0 {
		return
	}
	payload := StatusPayload{UserID: userID, IsOnline: online}
	data, _ := json.Marshal(WSMessage{
		Type:    WSMStatus,
		Payload: mustMarshal(&payload),
	})
	sendToClients(clients, data)
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
