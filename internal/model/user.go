// Package model 定义用户模型
// 用户分为普通用户和 AI 用户，通过 IsAI 字段区分
// AI 用户由系统内部创建，不走正常注册流程
package model

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// UserSettings 用户偏好配置，以 JSON 形式存入 User.Settings 字段
type UserSettings struct {
	Theme        string `json:"theme"`         // "light" | "dark"
	FontSize     string `json:"font_size"`     // "small" | "medium" | "large"
	EnterToSend  bool   `json:"enter_to_send"` // Enter 发送消息/换行
	SoundEnabled bool   `json:"sound_enabled"` // 消息提示音
}

// DefaultSettings 返回新建用户/未配置时的默认设置
func DefaultSettings() *UserSettings {
	return &UserSettings{
		Theme:        "light",
		FontSize:     "medium",
		EnterToSend:  true,
		SoundEnabled: true,
	}
}

// ParseSettings 从 JSON 字段解析为 UserSettings，失败时返回默认值
func ParseSettings(raw datatypes.JSON) *UserSettings {
	s := DefaultSettings()
	if raw == nil {
		return s
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return DefaultSettings()
	}
	// 校验枚举值，非法值回退默认
	switch s.Theme {
	case "light", "dark":
	default:
		s.Theme = "light"
	}
	switch s.FontSize {
	case "small", "medium", "large":
	default:
		s.FontSize = "medium"
	}
	return s
}

// User 用户模型
type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password string `gorm:"size:256;not null" json:"-"` // bcrypt 哈希，json 不输出
	Nickname string `gorm:"size:128" json:"nickname"`
	Avatar   string `gorm:"size:512" json:"avatar"`
	Bio      string `gorm:"size:512" json:"bio"` // 个人简介

	// IsAI 标记是否为 AI 用户，true 时不可通过普通注册创建
	IsAI bool `gorm:"default:false" json:"is_ai"`
	// LanLineodel 若为 AI 用户，记录其背后的模型名称（如 gpt-4 / qwen）
	LanLineodel string `gorm:"size:128;default:''" json:"ai_model,omitempty"`
	// AISystemPrompt 若为 AI 用户，其行为约束的系统提示词
	AISystemPrompt string `gorm:"type:text" json:"ai_system_prompt,omitempty"`
	// AIEndpoint 若为 AI 用户，其对应的 API 端点
	AIEndpoint string `gorm:"size:512;default:''" json:"ai_endpoint,omitempty"`

	// Settings 用户配置（JSON）：主题、字体、聊天偏好、隐私等
	Settings datatypes.JSON `gorm:"type:json" json:"settings"`

	// TokenVersion 登录版本号，每次登录+1，用于踢出旧登录
	TokenVersion int `gorm:"default:0" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
