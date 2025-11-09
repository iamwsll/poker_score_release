package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"poker_score_backend/models"
	"poker_score_backend/utils"
	ws "poker_score_backend/websocket"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	roomInactivityDuration    = 12 * time.Hour
	roomInactivityCheckPeriod = time.Hour
)

// RoomService 房间服务
type RoomService struct {
	hub *ws.Hub
}

// NewRoomService 创建房间服务
func NewRoomService(hub *ws.Hub) *RoomService {
	service := &RoomService{
		hub: hub,
	}

	go service.runInactivityWatcher()

	return service
}

func (s *RoomService) runInactivityWatcher() {
	ticker := time.NewTicker(roomInactivityCheckPeriod)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupInactiveRooms()
	}
}

func (s *RoomService) cleanupInactiveRooms() {
	var roomIDs []uint
	if err := models.DB.Model(&models.Room{}).
		Where("status = ?", "active").
		Pluck("id", &roomIDs).Error; err != nil {
		log.Printf("清理超时房间失败: %v", err)
		return
	}

	for _, roomID := range roomIDs {
		s.checkAndDissolveRoom(roomID)
	}
}

// CreateRoom 创建房间
func (s *RoomService) CreateRoom(userID uint, roomType, chipRate string) (*models.Room, error) {
	// 生成唯一的房间号（最多尝试10次）
	var roomCode string
	for i := 0; i < 10; i++ {
		roomCode = utils.GenerateRoomCode()

		// 检查房间号是否已被使用（仅检查活跃房间）
		var count int64
		err := models.DB.Model(&models.Room{}).
			Where("room_code = ? AND status = ?", roomCode, "active").
			Count(&count).Error

		if err != nil {
			log.Printf("查询房间号失败: %v", err)
			return nil, err
		}

		if count == 0 {
			break
		}
	}

	// 创建房间
	room := models.Room{
		RoomCode:  roomCode,
		RoomType:  roomType,
		ChipRate:  chipRate,
		Status:    "active",
		CreatedBy: userID,
	}

	err := models.DB.Create(&room).Error
	if err != nil {
		log.Printf("创建房间失败: %v", err)
		return nil, err
	}

	log.Printf("房间创建成功: ID=%d, RoomCode=%s, Type=%s, CreatedBy=%d", room.ID, room.RoomCode, room.RoomType, userID)

	// 记录房间创建操作
	s.recordOperation(room.ID, userID, models.OpTypeCreate, nil, nil, "创建了房间")

	// 创建者自动加入房间
	_, err = s.JoinRoom(userID, room.ID)
	if err != nil {
		log.Printf("创建者加入房间失败: %v", err)
		// 回滚房间创建
		models.DB.Delete(&room)
		return nil, err
	}

	return &room, nil
}

// JoinRoom 加入房间
func (s *RoomService) JoinRoom(userID, roomID uint) (*models.RoomMember, error) {
	// 检查房间是否存在且活跃
	var room models.Room
	err := models.DB.First(&room, roomID).Error
	if err != nil {
		return nil, errors.New("房间不存在")
	}

	if room.Status != "active" {
		return nil, errors.New("房间已解散")
	}

	// 检查用户是否已在房间中（left_at为NULL）
	var existingMember models.RoomMember
	err = models.DB.Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).First(&existingMember).Error
	if err == nil {
		// 用户已在房间中
		return &existingMember, nil
	}

	// 创建房间成员记录
	member := models.RoomMember{
		RoomID:   roomID,
		UserID:   userID,
		JoinedAt: time.Now(),
		Status:   "online",
	}

	err = models.DB.Create(&member).Error
	if err != nil {
		log.Printf("加入房间失败: %v", err)
		return nil, err
	}

	// 初始化用户积分余额
	err = s.initUserBalance(roomID, userID)
	if err != nil {
		log.Printf("初始化用户积分失败: %v", err)
	}

	// 记录操作
	s.recordOperation(roomID, userID, models.OpTypeJoin, nil, nil, "加入了房间")

	log.Printf("用户加入房间成功: RoomID=%d, UserID=%d", roomID, userID)

	s.broadcastUserJoined(roomID, userID, member.JoinedAt)

	return &member, nil
}

// JoinRoomByCode 通过房间号加入房间
func (s *RoomService) JoinRoomByCode(userID uint, roomCode string) (*models.Room, *models.RoomMember, error) {
	// 查询活跃的房间
	var room models.Room
	err := models.DB.Where("room_code = ? AND status = ?", roomCode, "active").First(&room).Error
	if err != nil {
		return nil, nil, errors.New("房间不存在或已解散")
	}

	// 加入房间
	member, err := s.JoinRoom(userID, room.ID)
	if err != nil {
		return nil, nil, err
	}

	return &room, member, nil
}

// LeaveRoom 离开房间
func (s *RoomService) LeaveRoom(userID, roomID uint) error {
	// 查找用户的在线成员记录
	var member models.RoomMember
	err := models.DB.Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).First(&member).Error
	if err != nil {
		return errors.New("您不在该房间中")
	}

	now := time.Now()
	if err := models.DB.Model(&member).Update("status", "offline").Error; err != nil {
		log.Printf("离开房间失败: %v", err)
		return err
	}

	// 记录操作
	s.recordOperation(roomID, userID, models.OpTypeLeave, nil, nil, "离开了房间")

	log.Printf("用户离开房间成功: RoomID=%d, UserID=%d", roomID, userID)

	s.broadcastUserLeft(roomID, userID, "offline", now)

	// 检查房间是否应该解散（所有人都离开了）
	s.checkAndDissolveRoom(roomID)

	return nil
}

// KickUser 踢出用户
func (s *RoomService) KickUser(roomID, userID, targetUserID uint) error {
	// 检查目标用户是否在房间中
	var member models.RoomMember
	err := models.DB.Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, targetUserID).First(&member).Error
	if err != nil {
		return errors.New("目标用户不在房间中")
	}

	now := time.Now()
	if err := models.DB.Model(&member).Update("status", "offline").Error; err != nil {
		log.Printf("踢出用户失败: %v", err)
		return err
	}

	// 记录操作
	targetUserIDCopy := targetUserID
	desc := "踢出了用户"
	var targetUser models.User
	if err := models.DB.First(&targetUser, targetUserID).Error; err == nil {
		desc = fmt.Sprintf("踢出了用户%s", targetUser.Nickname)
	}
	s.recordOperation(roomID, userID, models.OpTypeKick, nil, &targetUserIDCopy, desc)

	log.Printf("踢出用户成功: RoomID=%d, KickedBy=%d, KickedUser=%d", roomID, userID, targetUserID)

	s.broadcastUserKicked(roomID, targetUserID, userID, now)

	// 检查房间是否应该解散
	s.checkAndDissolveRoom(roomID)

	return nil
}

// GetRoomDetails 获取房间详情
func (s *RoomService) GetRoomDetails(roomID, userID uint) (map[string]interface{}, error) {
	// 查询房间信息
	var room models.Room
	err := models.DB.First(&room, roomID).Error
	if err != nil {
		return nil, errors.New("房间不存在")
	}

	// 检查用户是否在房间中
	var member models.RoomMember
	err = models.DB.Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).First(&member).Error
	if err != nil {
		return nil, errors.New("您不在该房间中")
	}

	// 获取所有在线成员及其积分
	members, err := s.GetRoomMembers(roomID)
	if err != nil {
		return nil, err
	}

	// 获取用户的积分余额
	myBalance, err := s.GetUserBalance(roomID, userID)
	if err != nil {
		myBalance = 0
	}

	// 计算桌面积分
	tableBalance := s.CalculateTableBalance(roomID)

	return map[string]interface{}{
		"room_id":       room.ID,
		"room_code":     room.RoomCode,
		"room_type":     room.RoomType,
		"chip_rate":     room.ChipRate,
		"status":        room.Status,
		"table_balance": tableBalance,
		"my_balance":    myBalance,
		"members":       members,
	}, nil
}

// GetRoomMembers 获取房间成员列表
func (s *RoomService) GetRoomMembers(roomID uint) ([]map[string]interface{}, error) {
	// 查询所有在线成员
	var members []models.RoomMember
	err := models.DB.Where("room_id = ? AND left_at IS NULL", roomID).Find(&members).Error
	if err != nil {
		return nil, err
	}

	// 获取用户信息和积分
	result := make([]map[string]interface{}, 0, len(members))
	for _, member := range members {
		var user models.User
		err := models.DB.First(&user, member.UserID).Error
		if err != nil {
			continue
		}

		balance, _ := s.GetUserBalance(roomID, member.UserID)

		result = append(result, map[string]interface{}{
			"user_id":  user.ID,
			"nickname": user.Nickname,
			"balance":  balance,
			"status":   member.Status,
		})
	}

	return result, nil
}

// GetLastRoom 获取用户最后加入的房间
func (s *RoomService) GetLastRoom(userID uint) (*models.Room, error) {
	// 查询用户最后加入的房间成员记录
	var member models.RoomMember
	err := models.DB.Where("user_id = ? AND left_at IS NULL", userID).
		Order("joined_at DESC").
		First(&member).Error

	if err != nil {
		return nil, errors.New("没有找到上次加入的房间")
	}

	// 查询房间信息
	var room models.Room
	err = models.DB.First(&room, member.RoomID).Error
	if err != nil {
		return nil, errors.New("房间不存在")
	}

	// 检查房间是否已解散
	if room.Status != "active" {
		return nil, errors.New("房间已解散")
	}

	operation, err := s.markUserReturned(&member)
	if err != nil {
		log.Printf("记录用户返回房间失败: RoomID=%d, UserID=%d, %v", member.RoomID, userID, err)
		return nil, err
	}

	if operation != nil {
		s.broadcastUserReturned(member.RoomID, userID, operation.CreatedAt)
	}

	return &room, nil
}

func (s *RoomService) markUserReturned(member *models.RoomMember) (*models.RoomOperation, error) {
	var operation *models.RoomOperation

	err := models.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.RoomMember{}).
			Where("id = ?", member.ID).
			Update("status", "online")
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("您不在该房间中")
		}

		op, err := s.recordOperationWithDB(tx, member.RoomID, member.UserID, models.OpTypeReturn, nil, nil, "返回了房间")
		if err != nil {
			return err
		}
		operation = op
		return nil
	})

	if err != nil {
		return nil, err
	}

	return operation, nil
}

// initUserBalance 初始化用户积分余额
func (s *RoomService) initUserBalance(roomID, userID uint) error {
	// 检查是否已存在余额记录
	var count int64
	err := models.DB.Model(&models.UserBalance{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error

	if err != nil {
		return err
	}

	if count > 0 {
		// 已存在，不需要初始化
		return nil
	}

	// 创建余额记录
	balance := models.UserBalance{
		RoomID:  roomID,
		UserID:  userID,
		Balance: 0,
	}

	return models.DB.Create(&balance).Error
}

// GetUserBalance 获取用户积分余额
func (s *RoomService) GetUserBalance(roomID, userID uint) (int, error) {
	return s.GetUserBalanceWithDB(nil, roomID, userID)
}

// GetUserBalanceWithDB 使用指定的DB实例获取用户积分余额
func (s *RoomService) GetUserBalanceWithDB(db *gorm.DB, roomID, userID uint) (int, error) {
	if db == nil {
		db = models.DB
	}

	var balance models.UserBalance
	err := db.Where("room_id = ? AND user_id = ?", roomID, userID).First(&balance).Error
	if err != nil {
		return 0, err
	}
	return balance.Balance, nil
}

// UpdateUserBalance 更新用户积分余额
func (s *RoomService) UpdateUserBalance(roomID, userID uint, amount int) error {
	return s.UpdateUserBalanceWithDB(nil, roomID, userID, amount)
}

// UpdateUserBalanceWithDB 使用指定的DB实例更新用户积分余额
func (s *RoomService) UpdateUserBalanceWithDB(db *gorm.DB, roomID, userID uint, amount int) error {
	if db == nil {
		db = models.DB
	}

	res := db.Model(&models.UserBalance{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount))

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// CalculateTableBalance 计算桌面积分
func (s *RoomService) CalculateTableBalance(roomID uint) int {
	return s.CalculateTableBalanceWithDB(nil, roomID)
}

// CalculateTableBalanceWithDB 使用指定的DB实例计算桌面积分
func (s *RoomService) CalculateTableBalanceWithDB(db *gorm.DB, roomID uint) int {
	if db == nil {
		db = models.DB
	}

	var totalBet int64
	if err := db.Model(&models.RoomOperation{}).
		Where("room_id = ? AND operation_type IN ?", roomID, []string{models.OpTypeBet, models.OpTypeNiuniuBet}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalBet).Error; err != nil {
		return 0
	}

	var totalWithdraw int64
	if err := db.Model(&models.RoomOperation{}).
		Where("room_id = ? AND operation_type = ?", roomID, models.OpTypeWithdraw).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalWithdraw).Error; err != nil {
		return 0
	}

	tableBalance := int(totalBet - totalWithdraw)
	if tableBalance < 0 {
		tableBalance = 0
	}

	return tableBalance
}

// checkAndDissolveRoom 检查并解散房间
func (s *RoomService) checkAndDissolveRoom(roomID uint) {
	var room models.Room
	if err := models.DB.First(&room, roomID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("查询房间失败: RoomID=%d, %v", roomID, err)
		}
		return
	}

	if room.Status != "active" {
		return
	}

	cutoff := time.Now().Add(-roomInactivityDuration)

	var lastOp models.RoomOperation
	err := models.DB.Where("room_id = ?", roomID).
		Order("created_at DESC").
		First(&lastOp).Error

	var settledAt time.Time
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("查询房间最近操作失败: RoomID=%d, %v", roomID, err)
			return
		}

		if room.CreatedAt.After(cutoff) {
			return
		}
		settledAt = room.CreatedAt
	} else if lastOp.CreatedAt.After(cutoff) {
		return
	} else {
		settledAt = lastOp.CreatedAt
	}
	//我们让"自动结算"的时间设定在"房间关闭前的最后一次"金融"操作",避免在计算历史战绩时搞错了时间
	financialOpTypes := []string{
		models.OpTypeBet,
		models.OpTypeWithdraw,
		models.OpTypeNiuniuBet,
	}

	var lastFinancialOp models.RoomOperation
	finErr := models.DB.Where("room_id = ? AND operation_type IN ?", roomID, financialOpTypes).
		Order("created_at DESC").
		First(&lastFinancialOp).Error

	if finErr != nil {
		if !errors.Is(finErr, gorm.ErrRecordNotFound) {
			log.Printf("查询房间最近金融操作失败: RoomID=%d, %v", roomID, finErr)
			return
		}
	} else {
		settledAt = lastFinancialOp.CreatedAt
	}

	now := time.Now()
	if settledAt.IsZero() || settledAt.After(now) {
		settledAt = now
	}

	err = models.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Room{}).
			Where("id = ? AND status = ?", roomID, "active").
			Updates(map[string]interface{}{
				"status":       "dissolved",
				"dissolved_at": now,
			})

		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if err := s.autoSettleRoomWithDB(tx, &room, settledAt); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}

		log.Printf("解散房间失败: RoomID=%d, %v", roomID, err)
		return
	}

	log.Printf("房间因12小时无操作已解散: RoomID=%d", roomID)
	s.broadcastRoomDissolved(roomID, now)
}

func (s *RoomService) autoSettleRoomWithDB(tx *gorm.DB, room *models.Room, settledAt time.Time) error {
	var balances []models.UserBalance
	if err := tx.Where("room_id = ?", room.ID).Find(&balances).Error; err != nil {
		return err
	}

	if len(balances) == 0 {
		return nil
	}

	tableBalance := s.CalculateTableBalanceWithDB(tx, room.ID)
	if tableBalance > 0 {
		bestIdx := -1
		for i := range balances {
			if bestIdx == -1 ||
				balances[i].Balance > balances[bestIdx].Balance ||
				(balances[i].Balance == balances[bestIdx].Balance && balances[i].UserID < balances[bestIdx].UserID) {
				bestIdx = i
			}
		}

		if bestIdx >= 0 {
			balances[bestIdx].Balance += tableBalance
		}
	}

	batchID := fmt.Sprintf("auto-%s", uuid.New().String())

	for _, balance := range balances {
		if balance.Balance == 0 {
			continue
		}

		rmbAmount := calculateRmbAmount(balance.Balance, room.ChipRate)
		settlement := models.Settlement{
			RoomID:          room.ID,
			UserID:          balance.UserID,
			ChipAmount:      balance.Balance,
			RmbAmount:       rmbAmount,
			SettledAt:       settledAt,
			SettlementBatch: batchID,
		}

		if err := tx.Create(&settlement).Error; err != nil {
			return err
		}
	}

	if err := tx.Model(&models.UserBalance{}).
		Where("room_id = ?", room.ID).
		Update("balance", 0).Error; err != nil {
		return err
	}

	return nil
}

// recordOperation 记录操作
func (s *RoomService) recordOperation(roomID, userID uint, opType string, amount *int, targetUserID *uint, description string) {
	if _, err := s.recordOperationWithDB(nil, roomID, userID, opType, amount, targetUserID, description); err != nil {
		log.Printf("记录操作失败: %v", err)
	}
}

func (s *RoomService) recordOperationWithDB(db *gorm.DB, roomID, userID uint, opType string, amount *int, targetUserID *uint, description string) (*models.RoomOperation, error) {
	if db == nil {
		db = models.DB
	}

	operation := models.RoomOperation{
		RoomID:        roomID,
		UserID:        userID,
		OperationType: opType,
		Amount:        amount,
		TargetUserID:  targetUserID,
		Description:   description,
	}

	if err := db.Create(&operation).Error; err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *RoomService) broadcastUserJoined(roomID, userID uint, joinedAt time.Time) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		log.Printf("广播用户加入时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	balance, err := s.GetUserBalance(roomID, userID)
	if err != nil {
		log.Printf("广播用户加入时获取用户积分失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		balance = 0
	}

	message := ws.Message{
		Type: "user_joined",
		Data: map[string]interface{}{
			"user_id":   user.ID,
			"nickname":  user.Nickname,
			"balance":   balance,
			"status":    "online",
			"joined_at": joinedAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化用户加入消息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	log.Printf("广播用户加入: RoomID=%d, UserID=%d", roomID, userID)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastUserReturned(roomID, userID uint, returnedAt time.Time) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		log.Printf("广播用户返回时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	balance, err := s.GetUserBalance(roomID, userID)
	if err != nil {
		log.Printf("广播用户返回时获取用户积分失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		balance = 0
	}

	message := ws.Message{
		Type: "user_returned",
		Data: map[string]interface{}{
			"user_id":     user.ID,
			"nickname":    user.Nickname,
			"balance":     balance,
			"status":      "online",
			"returned_at": returnedAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化用户返回消息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	log.Printf("广播用户返回: RoomID=%d, UserID=%d", roomID, userID)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastBet(roomID, userID uint, amount, myBalance, tableBalance int, createdAt time.Time) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		log.Printf("广播下注时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	message := ws.Message{
		Type: "bet",
		Data: map[string]interface{}{
			"user_id":       user.ID,
			"nickname":      user.Nickname,
			"amount":        amount,
			"balance":       myBalance,
			"table_balance": tableBalance,
			"created_at":    createdAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化下注消息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	log.Printf("广播下注: RoomID=%d, UserID=%d, Amount=%d", roomID, userID, amount)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastNiuniuBet(roomID, userID uint, betDetails []NiuniuBetDetail, totalAmount, myBalance, tableBalance int, createdAt time.Time) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		log.Printf("广播牛牛下注时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	message := ws.Message{
		Type: "niuniu_bet",
		Data: map[string]interface{}{
			"user_id":       user.ID,
			"nickname":      user.Nickname,
			"total_amount":  totalAmount,
			"balance":       myBalance,
			"table_balance": tableBalance,
			"bets":          betDetails,
			"created_at":    createdAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化牛牛下注消息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	log.Printf("广播牛牛下注: RoomID=%d, UserID=%d, TotalAmount=%d", roomID, userID, totalAmount)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastWithdraw(roomID, userID uint, amount, myBalance, tableBalance int, createdAt time.Time) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		log.Printf("广播收回时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	message := ws.Message{
		Type: "withdraw",
		Data: map[string]interface{}{
			"user_id":       user.ID,
			"nickname":      user.Nickname,
			"amount":        amount,
			"balance":       myBalance,
			"table_balance": tableBalance,
			"created_at":    createdAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化收回消息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	log.Printf("广播收回: RoomID=%d, UserID=%d, Amount=%d", roomID, userID, amount)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastUserLeft(roomID, userID uint, status string, leftAt time.Time) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		log.Printf("广播用户离开时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	message := ws.Message{
		Type: "user_left",
		Data: map[string]interface{}{
			"user_id":  user.ID,
			"nickname": user.Nickname,
			"status":   status,
			"left_at":  leftAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化用户离开消息失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return
	}

	log.Printf("广播用户离开: RoomID=%d, UserID=%d", roomID, userID)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastUserKicked(roomID, kickedUserID, kickedBy uint, kickedAt time.Time) {
	if s.hub == nil {
		return
	}

	var targetUser models.User
	if err := models.DB.First(&targetUser, kickedUserID).Error; err != nil {
		log.Printf("广播踢出消息时获取目标用户失败: RoomID=%d, UserID=%d, %v", roomID, kickedUserID, err)
		return
	}

	var kicker models.User
	if err := models.DB.First(&kicker, kickedBy).Error; err != nil {
		log.Printf("广播踢出消息时获取操作者失败: RoomID=%d, UserID=%d, %v", roomID, kickedBy, err)
		return
	}

	message := ws.Message{
		Type: "user_kicked",
		Data: map[string]interface{}{
			"user_id":            targetUser.ID,
			"nickname":           targetUser.Nickname,
			"kicked_by":          kicker.ID,
			"kicked_by_nickname": kicker.Nickname,
			"status":             "offline",
			"kicked_at":          kickedAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化踢出消息失败: RoomID=%d, UserID=%d, %v", roomID, kickedUserID, err)
		return
	}

	log.Printf("广播踢出: RoomID=%d, UserID=%d, KickedBy=%d", roomID, kickedUserID, kickedBy)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastSettlementInitiated(roomID, initiatedBy uint, initiatedAt time.Time, plan []SettlementPlan, tableBalance int) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, initiatedBy).Error; err != nil {
		log.Printf("广播结算发起时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, initiatedBy, err)
		return
	}

	message := ws.Message{
		Type: "settlement_initiated",
		Data: map[string]interface{}{
			"initiated_by":          user.ID,
			"initiated_by_nickname": user.Nickname,
			"initiated_at":          initiatedAt.Format(time.RFC3339),
			"settlement_plan":       plan,
			"table_balance":         tableBalance,
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化结算发起消息失败: RoomID=%d, UserID=%d, %v", roomID, initiatedBy, err)
		return
	}

	log.Printf("广播结算发起: RoomID=%d, UserID=%d", roomID, initiatedBy)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastSettlementConfirmed(roomID, confirmedBy uint, settlementBatch string, settledAt time.Time) {
	if s.hub == nil {
		return
	}

	var user models.User
	if err := models.DB.First(&user, confirmedBy).Error; err != nil {
		log.Printf("广播结算确认时获取用户信息失败: RoomID=%d, UserID=%d, %v", roomID, confirmedBy, err)
		return
	}

	message := ws.Message{
		Type: "settlement_confirmed",
		Data: map[string]interface{}{
			"confirmed_by":          user.ID,
			"confirmed_by_nickname": user.Nickname,
			"settlement_batch":      settlementBatch,
			"settled_at":            settledAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化结算确认消息失败: RoomID=%d, UserID=%d, %v", roomID, confirmedBy, err)
		return
	}

	log.Printf("广播结算确认: RoomID=%d, UserID=%d, Batch=%s", roomID, confirmedBy, settlementBatch)
	s.hub.BroadcastToRoom(roomID, payload)
}

func (s *RoomService) broadcastRoomDissolved(roomID uint, dissolvedAt time.Time) {
	if s.hub == nil {
		return
	}

	message := ws.Message{
		Type: "room_dissolved",
		Data: map[string]interface{}{
			"room_id":      roomID,
			"dissolved_at": dissolvedAt.Format(time.RFC3339),
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化房间解散消息失败: RoomID=%d, %v", roomID, err)
		return
	}

	log.Printf("广播房间解散: RoomID=%d", roomID)
	s.hub.BroadcastToRoom(roomID, payload)
}
