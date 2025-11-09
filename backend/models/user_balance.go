package models

import (
	"time"
)

// UserBalance 用户积分余额模型
type UserBalance struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RoomID    uint      `gorm:"not null;uniqueIndex:idx_room_user_balance" json:"room_id"` // 房间ID
	UserID    uint      `gorm:"not null;uniqueIndex:idx_room_user_balance" json:"user_id"` // 用户ID
	Balance   int       `gorm:"not null;default:0" json:"balance"`                          // 当前积分余额（可为负）
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (UserBalance) TableName() string {
	return "user_balances"
}

