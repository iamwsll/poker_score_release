package models

import (
	"time"
)

// RoomOperation 房间操作记录模型
type RoomOperation struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RoomID        uint      `gorm:"not null;index:idx_room_created" json:"room_id"` // 房间ID
	UserID        uint      `gorm:"not null;index" json:"user_id"`                  // 操作用户ID
	OperationType string    `gorm:"size:50;not null;index" json:"operation_type"`   // 操作类型
	Amount        *int      `json:"amount,omitempty"`                               // 涉及积分数量
	TargetUserID  *uint     `json:"target_user_id,omitempty"`                       // 目标用户ID（踢人、给某人下注）
	Description   string    `gorm:"type:text" json:"description"`                   // 操作描述（JSON格式）
	CreatedAt     time.Time `gorm:"index:idx_room_created" json:"created_at"`       // 操作时间
}

// TableName 指定表名
func (RoomOperation) TableName() string {
	return "room_operations"
}

// 操作类型常量
const (
	OpTypeJoin                = "join"                 // 加入房间
	OpTypeLeave               = "leave"                // 离开房间
	OpTypeReturn              = "return"               // 返回房间
	OpTypeBet                 = "bet"                  // 下注/支出
	OpTypeWithdraw            = "withdraw"             // 收回
	OpTypeKick                = "kick"                 // 被踢出
	OpTypeSettlementInitiated = "settlement_initiated" // 发起结算
	OpTypeSettlementConfirmed = "settlement_confirmed" // 确认结算
	OpTypeNiuniuBet           = "niuniu_bet"           // 牛牛下注
)
