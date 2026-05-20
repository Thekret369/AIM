package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	AIBotStatusEnabled  = "enabled"
	AIBotStatusDisabled = "disabled"
	AIBotStatusDeleted  = "deleted"
)

const (
	AIBotAPISourceSystem     = "system"
	AIBotAPISourceThirdParty = "third_party"
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

	// APISource 区分系统内置 API 和用户自带第三方 API，用于计费和用量统计。
	APISource string `gorm:"size:24;default:'third_party';index" json:"api_source"`

	BaseURL      string         `gorm:"size:512;not null" json:"base_url"`
	APIKey       string         `gorm:"type:text" json:"-"`
	Model        string         `gorm:"size:128;not null" json:"model"`
	SystemPrompt string         `gorm:"type:text" json:"system_prompt"`
	ContextLimit int            `gorm:"default:12" json:"context_limit"`
	Temperature  float64        `gorm:"default:0.7" json:"temperature"`
	MaxTokens    int            `gorm:"default:1024" json:"max_tokens"`
	Status       string         `gorm:"size:16;default:'enabled';index" json:"status"`
	PluginConfig datatypes.JSON `gorm:"type:json" json:"plugin_config,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (b *AIBot) IsEnabled() bool {
	return b != nil && b.Status == AIBotStatusEnabled
}

func (b *AIBot) NormalizedAPISource() string {
	if b == nil {
		return AIBotAPISourceThirdParty
	}
	if b.APISource == AIBotAPISourceSystem || b.IsSystem {
		return AIBotAPISourceSystem
	}
	return AIBotAPISourceThirdParty
}
