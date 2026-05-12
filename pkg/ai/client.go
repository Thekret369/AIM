// Package ai 为 AIM 提供 AI 助手集成能力
// AI 以"虚拟用户"形态接入系统，通过 WebSocket 收发消息，
// 对 Hub 和其他模块来说与普通用户无异。
//
// 当前为预留接口，定义了 AI 客户端的核心行为：
//   - 连接到 AIM WebSocket 服务
//   - 接收消息（单聊/群聊@唤起）
//   - 调用后端大模型生成回复
//   - 通过 WebSocket 发送回复消息
//
// 后续实现时，AI 作为独立进程运行，与主服务解耦，可独立升级。
package ai

import "context"

// Message AI 收发消息的抽象表示
type Message struct {
	Type      string `json:"type"`         // text | image | file | audio
	Content   string `json:"content"`      // 文本内容或文件 URL
	FromID    uint   `json:"from_user_id"` // 发送者 ID
	ToID      *uint  `json:"to_user_id,omitempty"`
	GroupID   *uint  `json:"group_id,omitempty"`
	AtMention bool   `json:"at_mention"` // 是否被 @ 唤起
}

// Agent AI 客户端接口，所有 AI 模型实现需满足此接口
type Agent interface {
	// ID 返回此 AI 对应的用户 ID（在 AIM 系统中注册的虚拟用户）
	ID() uint
	// Process 接收一条消息，生成回复文本
	// ctx 用于控制超时和取消
	Process(ctx context.Context, msg *Message) (reply string, err error)
	// Connect 建立 WebSocket 连接，开始收发
	Connect(wsURL, token string) error
}
