package models

import (
	"time"
)

// RoomMember 房间成员模型
type RoomMember struct {
	ID       uint       `gorm:"primaryKey" json:"id"`
	RoomID   uint       `gorm:"not null;index:idx_room_user" json:"room_id"` // 房间ID
	UserID   uint       `gorm:"not null;index:idx_room_user" json:"user_id"` // 用户ID
	JoinedAt time.Time  `gorm:"not null;index" json:"joined_at"`             // 加入时间
	LeftAt   *time.Time `gorm:"index" json:"left_at,omitempty"`              // 离开时间（NULL表示在线）
	Status   string     `gorm:"size:20;not null;default:'online';index" json:"status"` // 状态：online/offline/kicked
}

// TableName 指定表名
func (RoomMember) TableName() string {
	return "room_members"
}

