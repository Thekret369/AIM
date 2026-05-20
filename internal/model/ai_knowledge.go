package model

import "time"

const (
	AIKnowledgeStatusEnabled  = "enabled"
	AIKnowledgeStatusDisabled = "disabled"
	AIKnowledgeStatusDeleted  = "deleted"
)

// AIKnowledgeBase 是蓝妹可引用的知识库；系统知识库只供后台调试使用。
type AIKnowledgeBase struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	OwnerID     *uint  `gorm:"index" json:"owner_id,omitempty"`
	Owner       User   `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"size:512" json:"description"`
	IsSystem    bool   `gorm:"default:false;index" json:"is_system"`
	Status      string `gorm:"size:16;default:'enabled';index" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AIKnowledgeDocument 存放知识库中的纯文本资料。
type AIKnowledgeDocument struct {
	ID              uint            `gorm:"primaryKey" json:"id"`
	KnowledgeBaseID uint            `gorm:"index;not null" json:"knowledge_base_id"`
	KnowledgeBase   AIKnowledgeBase `gorm:"foreignKey:KnowledgeBaseID" json:"knowledge_base,omitempty"`
	Title           string          `gorm:"size:128;not null" json:"title"`
	Content         string          `gorm:"type:text;not null" json:"content"`
	Status          string          `gorm:"size:16;default:'enabled';index" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AIBotKnowledgeBase 绑定蓝妹实例和知识库。
type AIBotKnowledgeBase struct {
	ID              uint            `gorm:"primaryKey" json:"id"`
	AIBotID         uint            `gorm:"uniqueIndex:idx_ai_bot_kb;not null" json:"ai_bot_id"`
	AIBot           AIBot           `gorm:"foreignKey:AIBotID" json:"ai_bot,omitempty"`
	KnowledgeBaseID uint            `gorm:"uniqueIndex:idx_ai_bot_kb;index;not null" json:"knowledge_base_id"`
	KnowledgeBase   AIKnowledgeBase `gorm:"foreignKey:KnowledgeBaseID" json:"knowledge_base,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}
