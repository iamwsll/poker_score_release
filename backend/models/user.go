package models

import (
	"time"
)

// User 用户模型
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Phone        string    `gorm:"uniqueIndex;size:11;not null" json:"phone"`         // 手机号
	Nickname     string    `gorm:"size:50;not null" json:"nickname"`                  // 昵称
	PasswordHash string    `gorm:"size:255;not null" json:"-"`                        // 密码哈希（不返回给前端）
	Role         string    `gorm:"size:20;not null;default:'user'" json:"role"`       // 用户角色：admin/user
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

