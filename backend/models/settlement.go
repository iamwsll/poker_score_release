package models

import (
	"time"
)

// Settlement 结算记录模型
type Settlement struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	RoomID          uint      `gorm:"not null;index:idx_room_settled" json:"room_id"` // 房间ID
	UserID          uint      `gorm:"not null;index:idx_user_settled" json:"user_id"` // 用户ID
	ChipAmount      int       `gorm:"not null" json:"chip_amount"`                     // 积分盈亏（正为盈，负为亏）
	RmbAmount       float64   `gorm:"type:decimal(10,2);not null" json:"rmb_amount"`   // 人民币盈亏
	SettledAt       time.Time `gorm:"not null;index" json:"settled_at"`                // 结算时间
	SettlementBatch string    `gorm:"size:50;not null;index" json:"settlement_batch"`  // 结算批次号（UUID）
}

// TableName 指定表名
func (Settlement) TableName() string {
	return "settlements"
}

