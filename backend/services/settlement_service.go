package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"poker_score_backend/models"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SettlementService 结算服务
type SettlementService struct {
	roomService *RoomService
}

// NewSettlementService 创建结算服务
func NewSettlementService(roomService *RoomService) *SettlementService {
	return &SettlementService{
		roomService: roomService,
	}
}

// SettlementPlan 结算方案
type SettlementPlan struct {
	FromUserID   uint    `json:"from_user_id"`
	FromNickname string  `json:"from_nickname"`
	ToUserID     uint    `json:"to_user_id"`
	ToNickname   string  `json:"to_nickname"`
	ChipAmount   int     `json:"chip_amount"`
	RmbAmount    float64 `json:"rmb_amount"`
	Description  string  `json:"description"`
}

// InitiateSettlement 发起结算
func (s *SettlementService) InitiateSettlement(roomID, userID uint) (bool, int, []SettlementPlan, error) {
	// 检查桌面积分是否为0
	tableBalance := s.roomService.CalculateTableBalance(roomID)
	if tableBalance != 0 {
		return false, tableBalance, nil, fmt.Errorf("桌面积分不为0，当前桌面积分：%d，无法结算", tableBalance)
	}

	// 获取房间信息
	var room models.Room
	err := models.DB.First(&room, roomID).Error
	if err != nil {
		return false, 0, nil, errors.New("房间不存在")
	}

	// 获取所有用户的积分
	var balances []models.UserBalance
	err = models.DB.Where("room_id = ?", roomID).Find(&balances).Error
	if err != nil {
		return false, 0, nil, err
	}

	// 生成结算方案
	plan := s.generateSettlementPlan(balances, room.ChipRate)

	initiatedAt := time.Now()

	// 记录操作
	s.roomService.recordOperation(roomID, userID, models.OpTypeSettlementInitiated, nil, nil, "发起了结算")

	log.Printf("发起结算: RoomID=%d, UserID=%d", roomID, userID)
	s.roomService.broadcastSettlementInitiated(roomID, userID, initiatedAt, plan, tableBalance)

	return true, 0, plan, nil
}

// ConfirmSettlement 确认结算
func (s *SettlementService) ConfirmSettlement(roomID, userID uint) (string, time.Time, error) {
	// 再次检查桌面积分是否为0
	tableBalance := s.roomService.CalculateTableBalance(roomID)
	if tableBalance != 0 {
		return "", time.Time{}, fmt.Errorf("桌面积分不为0，当前桌面积分：%d，无法结算", tableBalance)
	}

	// 获取房间信息
	var room models.Room
	err := models.DB.First(&room, roomID).Error
	if err != nil {
		return "", time.Time{}, errors.New("房间不存在")
	}

	// 获取所有用户的积分
	var balances []models.UserBalance
	err = models.DB.Where("room_id = ?", roomID).Find(&balances).Error
	if err != nil {
		return "", time.Time{}, err
	}

	// 生成结算批次号
	settlementBatch := uuid.New().String()
	settledAt := time.Now()

	// 使用事务确保原子性
	err = models.DB.Transaction(func(tx *gorm.DB) error {
		// 保存结算记录
		for _, balance := range balances {
			if balance.Balance == 0 {
				continue // 跳过积分为0的用户
			}

			// 计算人民币金额
			rmbAmount := calculateRmbAmount(balance.Balance, room.ChipRate)

			settlement := models.Settlement{
				RoomID:          roomID,
				UserID:          balance.UserID,
				ChipAmount:      balance.Balance,
				RmbAmount:       rmbAmount,
				SettledAt:       settledAt,
				SettlementBatch: settlementBatch,
			}

			err := tx.Create(&settlement).Error
			if err != nil {
				return err
			}
		}

		// 清空所有用户的积分
		err := tx.Model(&models.UserBalance{}).
			Where("room_id = ?", roomID).
			Update("balance", 0).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("确认结算失败: RoomID=%d, %v", roomID, err)
		return "", time.Time{}, err
	}

	// 收集结算详情
	type settlementDetail struct {
		UserID     uint    `json:"user_id"`
		Nickname   string  `json:"nickname"`
		ChipAmount int     `json:"chip_amount"`
		RmbAmount  float64 `json:"rmb_amount"`
	}

	userIDSet := make(map[uint]struct{})
	for _, balance := range balances {
		if balance.Balance == 0 {
			continue
		}
		userIDSet[balance.UserID] = struct{}{}
	}

	userIDs := make([]uint, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	userMap := make(map[uint]models.User)
	if len(userIDs) > 0 {
		var users []models.User
		if err := models.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			log.Printf("获取结算用户信息失败: RoomID=%d, %v", roomID, err)
		} else {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	details := make([]settlementDetail, 0, len(userIDs))
	for _, balance := range balances {
		if balance.Balance == 0 {
			continue
		}

		nickname := ""
		if user, exists := userMap[balance.UserID]; exists {
			nickname = user.Nickname
		}

		details = append(details, settlementDetail{
			UserID:     balance.UserID,
			Nickname:   nickname,
			ChipAmount: balance.Balance,
			RmbAmount:  calculateRmbAmount(balance.Balance, room.ChipRate),
		})
	}

	descPayload := map[string]interface{}{
		"batch":      settlementBatch,
		"settled_at": settledAt.Format(time.RFC3339),
		"chip_rate":  room.ChipRate,
		"details":    details,
	}

	descBytes, err := json.Marshal(descPayload)
	description := "确认了结算"
	if err != nil {
		log.Printf("序列化结算详情失败: RoomID=%d, Batch=%s, %v", roomID, settlementBatch, err)
	} else {
		description = string(descBytes)
	}

	// 记录操作
	s.roomService.recordOperation(roomID, userID, models.OpTypeSettlementConfirmed, nil, nil, description)

	log.Printf("确认结算成功: RoomID=%d, UserID=%d, Batch=%s", roomID, userID, settlementBatch)
	s.roomService.broadcastSettlementConfirmed(roomID, userID, settlementBatch, settledAt, descPayload)

	return settlementBatch, settledAt, nil
}

// generateSettlementPlan 生成结算方案
// 规则：所有负积分的人向正积分最高的人转账，正积分最高的人给其他正积分的人转账
func (s *SettlementService) generateSettlementPlan(balances []models.UserBalance, chipRate string) []SettlementPlan {
	// 分类用户
	type UserBalance struct {
		UserID   uint
		Nickname string
		Balance  int
	}

	var positiveUsers []UserBalance // 正积分用户
	var negativeUsers []UserBalance // 负积分用户

	for _, balance := range balances {
		if balance.Balance == 0 {
			continue
		}

		// 获取用户昵称
		var user models.User
		models.DB.First(&user, balance.UserID)

		ub := UserBalance{
			UserID:   balance.UserID,
			Nickname: user.Nickname,
			Balance:  balance.Balance,
		}

		if balance.Balance > 0 {
			positiveUsers = append(positiveUsers, ub)
		} else {
			negativeUsers = append(negativeUsers, ub)
		}
	}

	// 按积分排序
	sort.Slice(positiveUsers, func(i, j int) bool {
		return positiveUsers[i].Balance > positiveUsers[j].Balance
	})
	sort.Slice(negativeUsers, func(i, j int) bool {
		return negativeUsers[i].Balance < negativeUsers[j].Balance // 负数从小到大
	})

	plan := make([]SettlementPlan, 0)

	if len(positiveUsers) == 0 || len(negativeUsers) == 0 {
		return plan
	}

	// 找出正积分最高的人
	maxPositiveUser := positiveUsers[0]

	// 1. 所有负积分的人向正积分最高的人转账
	for _, negUser := range negativeUsers {
		amount := -negUser.Balance
		rmbAmount := calculateRmbAmount(amount, chipRate)

		plan = append(plan, SettlementPlan{
			FromUserID:   negUser.UserID,
			FromNickname: negUser.Nickname,
			ToUserID:     maxPositiveUser.UserID,
			ToNickname:   maxPositiveUser.Nickname,
			ChipAmount:   amount,
			RmbAmount:    rmbAmount,
			Description:  fmt.Sprintf("%s → %s %d积分（¥%.2f）", negUser.Nickname, maxPositiveUser.Nickname, amount, rmbAmount),
		})
	}

	// 2. 正积分最高的人给其他正积分的人转账
	for i := 1; i < len(positiveUsers); i++ {
		posUser := positiveUsers[i]
		amount := posUser.Balance
		rmbAmount := calculateRmbAmount(amount, chipRate)

		plan = append(plan, SettlementPlan{
			FromUserID:   maxPositiveUser.UserID,
			FromNickname: maxPositiveUser.Nickname,
			ToUserID:     posUser.UserID,
			ToNickname:   posUser.Nickname,
			ChipAmount:   amount,
			RmbAmount:    rmbAmount,
			Description:  fmt.Sprintf("%s → %s %d积分（¥%.2f）", maxPositiveUser.Nickname, posUser.Nickname, amount, rmbAmount),
		})
	}

	return plan
}

// ... existing code ...
