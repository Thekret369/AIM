// Package model 定义用户模型
// 用户分为普通用户和 AI 用户，通过 IsAI 字段区分
// AI 用户由系统内部创建，不走正常注册流程
package model

import "time"

// User 用户模型
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password  string    `gorm:"size:256;not null" json:"-"` // bcrypt 哈希，json 不输出
	Nickname  string    `gorm:"size:128" json:"nickname"`
	Avatar    string    `gorm:"size:512" json:"avatar"`

	// IsAI 标记是否为 AI 用户，true 时不可通过普通注册创建
	IsAI bool `gorm:"default:false" json:"is_ai"`
	// AIModel 若为 AI 用户，记录其背后的模型名称（如 gpt-4 / qwen）
	AIModel     string `gorm:"size:128;default:''" json:"ai_model,omitempty"`
	// AISystemPrompt 若为 AI 用户，其行为约束的系统提示词
	AISystemPrompt string `gorm:"type:text;default:''" json:"ai_system_prompt,omitempty"`
	// AIEndpoint 若为 AI 用户，其对应的 API 端点
	AIEndpoint  string `gorm:"size:512;default:''" json:"ai_endpoint,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
