package model

import "time"

// FriendRelation 好友关系
// Status: pending=待同意 accepted=已接受 blocked=已拉黑
type FriendRelation struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index:idx_friend_user;not null" json:"user_id"`
	FriendID   uint      `gorm:"index:idx_friend_user;not null" json:"friend_id"`
	Status     string    `gorm:"size:16;default:'pending'" json:"status"` // pending | accepted | blocked
	Remark     string    `gorm:"size:128;default:''" json:"remark"`        // 好友备注名
	Note       string    `gorm:"size:512;default:''" json:"note"`          // 备注信息
	GroupID    *uint     `gorm:"index" json:"group_id,omitempty"`          // 所属联系人分组
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ContactGroup 联系人分组（如：家人、同事、朋友）
type ContactGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`   // 分组所属用户
	Name      string    `gorm:"size:64;not null" json:"name"`    // 分组名称
	SortOrder int       `gorm:"default:0" json:"sort_order"`      // 排序
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
