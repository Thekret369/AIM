package tui

import "AIM/internal/model"

const (
	conversationUser      = model.ConversationUser
	conversationGroup     = model.ConversationGroup
	conversationBroadcast = "broadcast"
)

// Options 保存 TUI 客户端启动参数。
type Options struct {
	Server   string
	Username string
	Password string
	PageSize int
}

// Conversation 是终端界面中展示的一条会话入口。
type Conversation struct {
	Key      string
	Type     string
	ID       uint
	Name     string
	IsAI     bool
	ReadOnly bool
}

type friendInfo struct {
	ID       uint   `json:"id"`
	UserID   uint   `json:"user_id"`
	FriendID uint   `json:"friend_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	Note     string `json:"note"`
}

type aiBotInfo struct {
	ID       uint   `json:"id"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Status   string `json:"status"`
	IsSystem bool   `json:"is_system"`
}
