package services

import (
	"errors"
	"log"
	"poker_score_backend/models"
	"poker_score_backend/utils"
	"time"

	"github.com/google/uuid"
)

// AuthService 认证服务
type AuthService struct {
	sessionMaxAge time.Duration
}

// NewAuthService 创建认证服务
func NewAuthService(sessionMaxAge time.Duration) *AuthService {
	return &AuthService{
		sessionMaxAge: sessionMaxAge,
	}
}

// Register 用户注册
func (s *AuthService) Register(phone, nickname, password string) (*models.User, *models.Session, error) {
	// 检查手机号是否已注册
	var count int64
	err := models.DB.Model(&models.User{}).Where("phone = ?", phone).Count(&count).Error
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return nil, nil, err
	}

	if count > 0 {
		return nil, nil, errors.New("手机号已注册")
	}

	// 对密码进行哈希加密
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		return nil, nil, err
	}

	// 创建用户
	user := models.User{
		Phone:        phone,
		Nickname:     nickname,
		PasswordHash: passwordHash,
		Role:         "user",
	}

	err = models.DB.Create(&user).Error
	if err != nil {
		log.Printf("创建用户失败: %v", err)
		return nil, nil, err
	}

	log.Printf("用户注册成功: ID=%d, Phone=%s, Nickname=%s", user.ID, user.Phone, user.Nickname)

	// 自动登录，创建Session
	session, err := s.CreateSession(user.ID)
	if err != nil {
		return &user, nil, err
	}

	return &user, session, nil
}

// Login 用户登录
func (s *AuthService) Login(phone, password string) (*models.User, *models.Session, error) {
	// 查询用户
	var user models.User
	err := models.DB.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		log.Printf("用户不存在: Phone=%s", phone)
		return nil, nil, errors.New("手机号或密码错误")
	}

	// 验证密码
	if !utils.CheckPassword(password, user.PasswordHash) {
		log.Printf("密码错误: UserID=%d", user.ID)
		return nil, nil, errors.New("手机号或密码错误")
	}

	// 创建Session
	session, err := s.CreateSession(user.ID)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("用户登录成功: ID=%d, Phone=%s, Nickname=%s", user.ID, user.Phone, user.Nickname)

	return &user, session, nil
}

// CreateSession 创建Session
func (s *AuthService) CreateSession(userID uint) (*models.Session, error) {
	// 生成Session ID
	sessionID := uuid.New().String()

	// 创建Session
	session := models.Session{
		SessionID: sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.sessionMaxAge),
	}

	err := models.DB.Create(&session).Error
	if err != nil {
		log.Printf("创建Session失败: %v", err)
		return nil, err
	}

	log.Printf("Session创建成功: UserID=%d, SessionID=%s", userID, sessionID)

	return &session, nil
}

// Logout 用户登出
func (s *AuthService) Logout(sessionID string) error {
	// 删除Session
	err := models.DB.Where("session_id = ?", sessionID).Delete(&models.Session{}).Error
	if err != nil {
		log.Printf("删除Session失败: %v", err)
		return err
	}

	log.Printf("用户登出成功: SessionID=%s", sessionID)
	return nil
}

// GetUserByID 根据ID获取用户信息
func (s *AuthService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	err := models.DB.First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateNickname 修改昵称
func (s *AuthService) UpdateNickname(userID uint, nickname string) error {
	err := models.DB.Model(&models.User{}).Where("id = ?", userID).Update("nickname", nickname).Error
	if err != nil {
		log.Printf("修改昵称失败: UserID=%d, %v", userID, err)
		return err
	}

	log.Printf("昵称修改成功: UserID=%d, NewNickname=%s", userID, nickname)
	return nil
}

// UpdatePassword 修改密码
func (s *AuthService) UpdatePassword(userID uint, oldPassword, newPassword string) error {
	// 查询用户
	var user models.User
	err := models.DB.First(&user, userID).Error
	if err != nil {
		return err
	}

	// 验证旧密码
	if !utils.CheckPassword(oldPassword, user.PasswordHash) {
		return errors.New("旧密码错误")
	}

	// 对新密码进行哈希加密
	newPasswordHash, err := utils.HashPassword(newPassword)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		return err
	}

	// 更新密码
	err = models.DB.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", newPasswordHash).Error
	if err != nil {
		log.Printf("修改密码失败: UserID=%d, %v", userID, err)
		return err
	}

	log.Printf("密码修改成功: UserID=%d", userID)
	return nil
}

// CleanExpiredSessions 清理过期的Session
func (s *AuthService) CleanExpiredSessions() error {
	result := models.DB.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("清理过期Session: 删除%d条记录", result.RowsAffected)
	}

	return nil
}

