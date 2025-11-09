package models

import (
	"time"
)

// Room 房间模型
type Room struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	RoomCode    string     `gorm:"size:6;not null;index:idx_room_code" json:"room_code"` // 6位房间号
	RoomType    string     `gorm:"size:20;not null" json:"room_type"`                     // 房间类型：texas/niuniu
	ChipRate    string     `gorm:"size:20;not null" json:"chip_rate"`                     // 积分与人民币比例（如"20:1"）
	Status      string     `gorm:"size:20;not null;default:'active';index" json:"status"` // 房间状态：active/dissolved
	CreatedBy   uint       `gorm:"not null;index" json:"created_by"`                      // 创建者用户ID
	CreatedAt   time.Time  `gorm:"index:idx_room_code" json:"created_at"`
	DissolvedAt *time.Time `json:"dissolved_at,omitempty"` // 解散时间
}

// TableName 指定表名
func (Room) TableName() string {
	return "rooms"
}

