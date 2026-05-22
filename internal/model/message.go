package model

import "time"

// MessageType 消息类型
// text=文本 image=图片 file=文件 audio=音频
type MessageType string

const (
	MsgText  MessageType = "text"
	MsgImage MessageType = "image"
	MsgFile  MessageType = "file"
	MsgAudio MessageType = "audio"
)

const (
	ConversationUser  = "user"
	ConversationGroup = "group"
)

// Message 消息模型，单表存储所有类型消息
// 单聊：ToUserID 非空，GroupID 为空
// 群聊：GroupID 非空，ToUserID 为空
// 广播：ToUserID 和 GroupID 均为空
// 文件的真实存储路径在 Content 中，由 pkg/storage 包管理
type Message struct {
	ID         uint        `gorm:"primaryKey" json:"id"`
	Type       MessageType `gorm:"size:16;not null;default:'text'" json:"type"`
	FromUserID uint        `gorm:"index;not null" json:"from_user_id"`
	FromUser   User        `gorm:"foreignKey:FromUserID" json:"from_user,omitempty"`
	ToUserID   *uint       `gorm:"index" json:"to_user_id,omitempty"`   // 单聊接收者
	GroupID    *uint       `gorm:"index" json:"group_id,omitempty"`     // 群聊 ID
	Content    string      `gorm:"type:text" json:"content"`            // 文本内容或文件 URL
	FileName   string      `gorm:"size:256" json:"file_name,omitempty"` // 文件/图片/音频的原文件名
	FileSize   int64       `json:"file_size,omitempty"`                 // 文件大小(字节)
	// QuoteMessageID 指向被引用的消息，非空时表示这是一条引用回复。
	QuoteMessageID *uint    `gorm:"index" json:"quote_message_id,omitempty"`
	QuoteMessage   *Message `gorm:"foreignKey:QuoteMessageID" json:"quote_message,omitempty"`
	// RecipientCount 记录群消息发送时除发送者外的可读成员数，用于稳定历史已读分母
	RecipientCount int `gorm:"default:0" json:"recipient_count,omitempty"`
	// ThumbnailURL 缩略图 URL（仅图片消息）
	ThumbnailURL string `gorm:"size:512" json:"thumbnail_url,omitempty"`
	// Mentions 被 @ 的用户 ID 列表，JSON 数组如 "[1,3,5]"
	Mentions string `gorm:"size:512" json:"mentions,omitempty"`
	// IsRecalled 标记消息是否已撤回；撤回后内容字段会被清空，历史只保留占位状态。
	IsRecalled bool       `gorm:"not null;default:false;index" json:"is_recalled"`
	RecalledAt *time.Time `json:"recalled_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// MessageRead 消息已读记录
// 用户在某会话（单聊/群聊）中已读的最后一条消息 ID
type MessageRead struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	UserID           uint   `gorm:"index;not null" json:"user_id"`
	ConversationType string `gorm:"size:16;not null;default:'';index" json:"conversation_type"`
	ConversationID   uint   `gorm:"not null;default:0;index" json:"conversation_id"`
	PeerUserID       *uint  `gorm:"index" json:"peer_user_id,omitempty"` // 单聊对方，保留用于兼容旧查询
	GroupID          *uint  `gorm:"index" json:"group_id,omitempty"`     // 群聊，保留用于兼容旧查询
	LastReadMsgID    uint   `gorm:"not null" json:"last_read_msg_id"`    // 已读到的最后一条消息 ID
}

// MessageDeletion 记录某个用户对某条消息的个人软删除状态。
// 删除后仅该用户查询历史、搜索和离线同步时不可见，不影响其他会话成员。
type MessageDeletion struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_message_deletions_user_message" json:"user_id"`
	MessageID uint      `gorm:"not null;index;uniqueIndex:idx_message_deletions_user_message" json:"message_id"`
	Message   Message   `gorm:"foreignKey:MessageID" json:"message,omitempty"`
	DeletedAt time.Time `gorm:"not null" json:"deleted_at"`
}

// IsToUser 是否为发给特定用户的单聊消息
func (m *Message) IsToUser() bool {
	return m.ToUserID != nil
}

// IsToGroup 是否为群聊消息
func (m *Message) IsToGroup() bool {
	return m.GroupID != nil
}

// IsBroadcast 是否为广播消息
func (m *Message) IsBroadcast() bool {
	return m.ToUserID == nil && m.GroupID == nil
}
