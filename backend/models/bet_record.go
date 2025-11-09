package models

import (
	"time"
)

// BetRecord 牛牛下注记录模型（专门记录给某人下注）
type BetRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RoomID     uint      `gorm:"not null;index:idx_bet_room_created" json:"room_id"` // 房间ID
	FromUserID uint      `gorm:"not null;index" json:"from_user_id"`                 // 下注者用户ID
	ToUserID   uint      `gorm:"not null;index" json:"to_user_id"`                   // 被下注者用户ID
	Amount     int       `gorm:"not null" json:"amount"`                             // 下注积分数量
	CreatedAt  time.Time `gorm:"index:idx_bet_room_created" json:"created_at"`       // 下注时间
}

// TableName 指定表名
func (BetRecord) TableName() string {
	return "bet_records"
}
