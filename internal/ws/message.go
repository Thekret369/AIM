// Package ws WebSocket 消息协议定义
// WSMessage 作为外层包装，通过 Type 区分 chat/typing/read_receipt/status
// 客户端和服务端均使用此格式收发
package ws

import (
	"encoding/json"

	"AIM/internal/model"
)

// WSMessageType 区分 WS 消息类型
type WSMessageType string

const (
	WSMChat        WSMessageType = "chat"         // 聊天消息，Payload 为 model.Message JSON
	WSMTyping      WSMessageType = "typing"       // 输入状态
	WSMReadReceipt WSMessageType = "read_receipt" // 已读回执
	WSMStatus      WSMessageType = "status"       // 在线状态变更
	WSMAck         WSMessageType = "ack"          // 服务端确认
	WSMError       WSMessageType = "error"        // 服务端错误
)

// WSMessage 所有 WS 消息的外层包装
type WSMessage struct {
	Type      WSMessageType   `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

// ClientMessage 保留消息来源连接，便于服务端回传 ack/error。
type ClientMessage struct {
	Client    *Client
	RequestID string
	Message   *model.Message
}

// TypingPayload 输入状态
type TypingPayload struct {
	FromUserID uint `json:"from_user_id"`
	ToUserID   uint `json:"to_user_id,omitempty"` // 单聊接收方
	GroupID    uint `json:"group_id,omitempty"`   // 群聊 ID
	IsTyping   bool `json:"is_typing"`
}

// ReadReceiptPayload 已读回执
type ReadReceiptPayload struct {
	FromUserID    uint   `json:"from_user_id"`
	PeerUserID    uint   `json:"peer_user_id,omitempty"`     // 单聊时标记对方消息已读
	GroupID       uint   `json:"group_id,omitempty"`         // 群聊时标记群消息已读
	MessageIDs    []uint `json:"message_ids,omitempty"`      // 兼容旧协议：逐条消息 ID
	LastReadMsgID uint   `json:"last_read_msg_id,omitempty"` // 新协议：已读高水位
}

// StatusPayload 在线状态
type StatusPayload struct {
	UserID   uint `json:"user_id"`
	IsOnline bool `json:"is_online"`
}

// AckPayload 表示服务端已经成功处理某个客户端请求。
type AckPayload struct {
	RequestID string `json:"request_id,omitempty"`
	MessageID uint   `json:"message_id,omitempty"`
	Status    string `json:"status"`
}

// ErrorPayload 表示服务端拒绝或处理失败。
type ErrorPayload struct {
	RequestID string `json:"request_id,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

// IsClientMessageTypeAllowed 限定客户端可主动发送的 WS 类型。
func IsClientMessageTypeAllowed(t WSMessageType) bool {
	switch t {
	case WSMChat, WSMTyping, WSMReadReceipt:
		return true
	default:
		return false
	}
}
