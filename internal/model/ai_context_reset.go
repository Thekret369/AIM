package model

import "time"

const (
	AIContextConversationUser  = "user"
	AIContextConversationGroup = "group"
)

// AIContextReset 记录某个用户与某个 AI 在某个会话中的上下文截断点。
// 聊天历史仍然保留，但 AI 组装提示词时只读取截断点之后的消息。
type AIContextReset struct {
	ID uint `gorm:"primaryKey" json:"id"`

	BotUserID uint `gorm:"not null;index;uniqueIndex:idx_ai_context_resets_scope,priority:1" json:"bot_user_id"`
	UserID    uint `gorm:"not null;index;uniqueIndex:idx_ai_context_resets_scope,priority:2" json:"user_id"`

	ConversationType string `gorm:"size:16;not null;uniqueIndex:idx_ai_context_resets_scope,priority:3" json:"conversation_type"`
	ConversationID   uint   `gorm:"not null;uniqueIndex:idx_ai_context_resets_scope,priority:4" json:"conversation_id"`

	ResetAfterMessageID uint      `gorm:"not null;default:0" json:"reset_after_message_id"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
