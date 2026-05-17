package model

import "time"

const (
	AIBotStatusEnabled  = "enabled"
	AIBotStatusDisabled = "disabled"
	AIBotStatusDeleted  = "deleted"
)

// AIBot 保存 AI 用户背后的模型配置和归属关系。
type AIBot struct {
	ID      uint  `gorm:"primaryKey" json:"id"`
	UserID  uint  `gorm:"uniqueIndex;not null" json:"user_id"`
	User    User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OwnerID *uint `gorm:"index" json:"owner_id,omitempty"`
	Owner   User  `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`

	// IsSystem 表示后端托管的内置 AI，普通用户不能修改。
	IsSystem bool `gorm:"default:false;index" json:"is_system"`

	BaseURL      string  `gorm:"size:512;not null" json:"base_url"`
	APIKey       string  `gorm:"type:text" json:"-"`
	Model        string  `gorm:"size:128;not null" json:"model"`
	SystemPrompt string  `gorm:"type:text" json:"system_prompt"`
	ContextLimit int     `gorm:"default:12" json:"context_limit"`
	Temperature  float64 `gorm:"default:0.7" json:"temperature"`
	MaxTokens    int     `gorm:"default:1024" json:"max_tokens"`
	Status       string  `gorm:"size:16;default:'enabled';index" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (b *AIBot) IsEnabled() bool {
	return b != nil && b.Status == AIBotStatusEnabled
}
