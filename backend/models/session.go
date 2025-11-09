package models

import (
	"time"
)

// Session Session模型
type Session struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID string    `gorm:"uniqueIndex;size:255;not null" json:"session_id"` // Session标识符（UUID）
	UserID    uint      `gorm:"not null;index" json:"user_id"`                    // 用户ID
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"` // 过期时间
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}

// IsExpired 判断Session是否过期
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

