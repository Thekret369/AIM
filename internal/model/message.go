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

// Message 消息模型，单表存储所有类型消息
// 单聊：ToUserID 非空，GroupID 为空
// 群聊：GroupID 非空，ToUserID 为空
// 广播：ToUserID 和 GroupID 均为空
// 文件的真实存储路径在 Content 中，由 pkg/storage 包管理
type Message struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	Type      MessageType `gorm:"size:16;not null;default:'text'" json:"type"`
	FromUserID uint       `gorm:"index;not null" json:"from_user_id"`
	FromUser   User       `gorm:"foreignKey:FromUserID" json:"from_user,omitempty"`
	ToUserID   *uint      `gorm:"index" json:"to_user_id,omitempty"` // 单聊接收者
	GroupID    *uint      `gorm:"index" json:"group_id,omitempty"`    // 群聊 ID
	Content    string     `gorm:"type:text" json:"content"`            // 文本内容或文件 URL
	FileName   string     `gorm:"size:256" json:"file_name,omitempty"` // 文件/图片/音频的原文件名
	FileSize   int64      `json:"file_size,omitempty"`                 // 文件大小(字节)
	CreatedAt  time.Time  `json:"created_at"`
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
