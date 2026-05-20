package model

import "time"

// AITokenUsage 记录每次 AI 调用的 token 消耗和人民币费用。
type AITokenUsage struct {
	ID uint `gorm:"primaryKey" json:"id"`

	UserID    uint  `gorm:"index;not null" json:"user_id"`
	User      User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CallerID  uint  `gorm:"index;not null" json:"caller_id"`
	Caller    User  `gorm:"foreignKey:CallerID" json:"caller,omitempty"`
	BotID     uint  `gorm:"index;not null" json:"bot_id"`
	Bot       AIBot `gorm:"foreignKey:BotID" json:"bot,omitempty"`
	BotUserID uint  `gorm:"index;not null" json:"bot_user_id"`

	MessageID      *uint `gorm:"index" json:"message_id,omitempty"`
	ReplyMessageID *uint `gorm:"index" json:"reply_message_id,omitempty"`

	APISource       string `gorm:"size:24;index" json:"api_source"`
	ProviderBaseURL string `gorm:"size:512" json:"provider_base_url"`
	Model           string `gorm:"size:128" json:"model"`

	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	Billable      bool    `gorm:"index" json:"billable"`
	InputCostCNY  float64 `json:"input_cost_cny"`
	OutputCostCNY float64 `json:"output_cost_cny"`
	TotalCostCNY  float64 `json:"total_cost_cny"`

	CreatedAt time.Time `json:"created_at"`
}
