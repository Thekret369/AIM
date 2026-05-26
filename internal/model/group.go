package model

import "time"

// GroupRole 群成员角色
type GroupRole string

const (
	RoleOwner  GroupRole = "owner"  // 群主
	RoleAdmin  GroupRole = "admin"  // 管理员
	RoleMember GroupRole = "member" // 普通成员
)

// Group 群组模型
type Group struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Avatar      string    `gorm:"size:512" json:"avatar"`
	Description string    `gorm:"size:1024" json:"description"` // 群简介
	OwnerID     uint      `gorm:"index;not null" json:"owner_id"`
	Announce    string    `gorm:"type:text" json:"announce"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GroupMember 群成员关系
type GroupMember struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	GroupID    uint       `gorm:"index:idx_group_user;not null" json:"group_id"`
	UserID     uint       `gorm:"index:idx_group_user;not null" json:"user_id"`
	Role       GroupRole  `gorm:"size:16;default:'member'" json:"role"` // 角色
	MutedUntil *time.Time `json:"muted_until,omitempty"`
	DND        bool       `gorm:"default:false" json:"dnd"` // 消息免打扰
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Announcement 群公告，每次发布新增一条，保留历史记录
type Announcement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GroupID   uint      `gorm:"index;not null" json:"group_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	EditorID  uint      `json:"editor_id"` // 发布者
	Editor    User      `gorm:"foreignKey:EditorID" json:"editor,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// IsMuted 检查成员是否处于禁言状态（MutedUntil 不为空且时间未到）
func (gm *GroupMember) IsMuted() bool {
	if gm.MutedUntil == nil {
		return false
	}
	return time.Now().Before(*gm.MutedUntil)
}
