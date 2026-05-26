package ws

import (
	"encoding/json"
	"log"
	"time"

	"LanLine/internal/model"

	"github.com/gorilla/websocket"
)

// 读写超时和缓冲区大小
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 54 * time.Second
	maxMessageSize = 65536
	sendBufSize    = 256
)

// Client 表示单个 WebSocket 连接
type Client struct {
	UserID   uint
	Username string
	Hub      *Hub
	Conn     *websocket.Conn
	send     chan []byte
}

// NewClient 创建客户端实例并注册到 Hub
func NewClient(userID uint, username string, hub *Hub, conn *websocket.Conn) *Client {
	c := &Client{
		UserID:   userID,
		Username: username,
		Hub:      hub,
		Conn:     conn,
		send:     make(chan []byte, sendBufSize),
	}
	hub.Register(c)
	return c
}

// Start 启动客户端的读写协程
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// readPump 从 WebSocket 读取消息，按 type 分发到对应通道
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] 连接异常关闭: uid=%d, err=%v", c.UserID, err)
			}
			break
		}

		// 先尝试解析外层 WSMessage 包装
		var wrapper WSMessage
		if err := json.Unmarshal(data, &wrapper); err != nil {
			log.Printf("[ws] 消息解析失败: %v", err)
			c.SendError("", "bad_message", "消息格式错误")
			continue
		}
		if wrapper.Type == "" {
			// 兼容旧协议：无 type 字段则视为 chat 消息
			var msg model.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("[ws] 消息解析失败: %v", err)
				c.SendError("", "bad_message", "消息格式错误")
				continue
			}
			c.dispatchChat("", &msg)
			continue
		}
		if !IsClientMessageTypeAllowed(wrapper.Type) {
			log.Printf("[ws] 未允许的消息类型: %s", wrapper.Type)
			c.SendError(wrapper.RequestID, "unsupported_type", "不支持的消息类型")
			continue
		}

		switch wrapper.Type {
		case WSMChat:
			var msg model.Message
			if err := json.Unmarshal(wrapper.Payload, &msg); err != nil {
				log.Printf("[ws] chat 解析失败: %v", err)
				c.SendError(wrapper.RequestID, "bad_payload", "聊天消息格式错误")
				continue
			}
			c.dispatchChat(wrapper.RequestID, &msg)

		case WSMTyping:
			var p TypingPayload
			if err := json.Unmarshal(wrapper.Payload, &p); err != nil {
				log.Printf("[ws] typing 解析失败: %v", err)
				c.SendError(wrapper.RequestID, "bad_payload", "输入状态格式错误")
				continue
			}
			p.FromUserID = c.UserID
			c.Hub.OnTyping <- &p

		case WSMReadReceipt:
			var p ReadReceiptPayload
			if err := json.Unmarshal(wrapper.Payload, &p); err != nil {
				log.Printf("[ws] read_receipt 解析失败: %v", err)
				c.SendError(wrapper.RequestID, "bad_payload", "已读回执格式错误")
				continue
			}
			p.FromUserID = c.UserID
			c.Hub.OnReadReceipt <- &p
		}
	}
}

func (c *Client) dispatchChat(requestID string, msg *model.Message) {
	msg.FromUserID = c.UserID
	if c.Hub.UseClientMessages {
		c.Hub.OnClientMessage <- &ClientMessage{
			Client:    c,
			RequestID: requestID,
			Message:   msg,
		}
		return
	}
	c.Hub.OnMessage <- msg
}

// writePump 将发送缓冲区的消息写入 WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendAck(requestID string, messageID uint) {
	c.sendWrapped(WSMAck, requestID, &AckPayload{
		RequestID: requestID,
		MessageID: messageID,
		Status:    "ok",
	})
}

func (c *Client) SendError(requestID, code, message string) {
	if code == "" {
		code = "error"
	}
	if message == "" {
		message = "请求处理失败"
	}
	c.sendWrapped(WSMError, requestID, &ErrorPayload{
		RequestID: requestID,
		Code:      code,
		Message:   message,
	})
}

func (c *Client) sendWrapped(messageType WSMessageType, requestID string, payload interface{}) {
	data, err := json.Marshal(WSMessage{
		Type:      messageType,
		RequestID: requestID,
		Payload:   mustMarshal(payload),
	})
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}
