package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"poker_score_backend/models"

	"gorm.io/gorm"
)

// OperationService 房间操作服务
type OperationService struct {
	roomService *RoomService
}

// NiuniuBetItem 牛牛下注明细（请求）
type NiuniuBetItem struct {
	ToUserID uint `json:"to_user_id"`
	Amount   int  `json:"amount"`
}

// NiuniuBetDetail 牛牛下注明细（带昵称）
type NiuniuBetDetail struct {
	ToUserID   uint   `json:"to_user_id"`
	ToNickname string `json:"to_nickname,omitempty"`
	Amount     int    `json:"amount"`
}

// NewOperationService 创建房间操作服务
func NewOperationService(roomService *RoomService) *OperationService {
	return &OperationService{
		roomService: roomService,
	}
}

// Bet 下注/支出（德扑）
func (s *OperationService) Bet(roomID, userID uint, amount int) (int, int, error) {
	if amount <= 0 {
		return 0, 0, errors.New("下注金额必须大于0")
	}

	var myBalance, tableBalance int
	var operation *models.RoomOperation

	// 使用事务确保原子性
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		// 更新用户积分
		err := s.roomService.UpdateUserBalanceWithDB(tx, roomID, userID, -amount)
		if err != nil {
			return err
		}

		// 获取更新后的积分
		myBalance, err = s.roomService.GetUserBalanceWithDB(tx, roomID, userID)
		if err != nil {
			return err
		}

		// 记录操作
		amountCopy := amount
		op, err := s.roomService.recordOperationWithDB(tx, roomID, userID, models.OpTypeBet, &amountCopy, nil, fmt.Sprintf("下注了%d积分", amount))
		if err != nil {
			return err
		}
		operation = op

		// 记录完成后重新计算桌面积分
		tableBalance = s.roomService.CalculateTableBalanceWithDB(tx, roomID)

		return nil
	})

	if err != nil {
		log.Printf("下注失败: RoomID=%d, UserID=%d, Amount=%d, %v", roomID, userID, amount, err)
		return 0, 0, err
	}

	log.Printf("下注成功: RoomID=%d, UserID=%d, Amount=%d, Balance=%d", roomID, userID, amount, myBalance)

	if operation != nil {
		s.roomService.broadcastBet(roomID, userID, amount, myBalance, tableBalance, operation.CreatedAt)
	}

	return myBalance, tableBalance, nil
}

// Withdraw 收回
func (s *OperationService) Withdraw(roomID, userID uint, amount int) (int, int, int, error) {
	var myBalance, tableBalance, actualAmount int
	var operation *models.RoomOperation

	// 使用事务确保原子性
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		// 查询当前桌面可收回积分
		available := s.roomService.CalculateTableBalanceWithDB(tx, roomID)
		if available <= 0 {
			return errors.New("桌面没有可收回的积分")
		}

		// 计算实际收回金额
		if amount <= 0 {
			actualAmount = available
		} else {
			if amount > available {
				return errors.New("收回数量超过桌面积分")
			}
			actualAmount = amount
		}

		// 更新用户积分
		if err := s.roomService.UpdateUserBalanceWithDB(tx, roomID, userID, actualAmount); err != nil {
			return err
		}

		// 获取更新后的积分
		var err error
		myBalance, err = s.roomService.GetUserBalanceWithDB(tx, roomID, userID)
		if err != nil {
			return err
		}

		// 记录操作
		amountCopy := actualAmount
		op, err := s.roomService.recordOperationWithDB(tx, roomID, userID, models.OpTypeWithdraw, &amountCopy, nil, fmt.Sprintf("收回了%d积分", actualAmount))
		if err != nil {
			return err
		}
		operation = op

		// 重新计算桌面积分，包含此次收回
		tableBalance = s.roomService.CalculateTableBalanceWithDB(tx, roomID)

		return nil
	})

	if err != nil {
		log.Printf("收回失败: RoomID=%d, UserID=%d, Amount=%d, %v", roomID, userID, amount, err)
		return 0, 0, 0, err
	}

	log.Printf("收回成功: RoomID=%d, UserID=%d, ActualAmount=%d, Balance=%d", roomID, userID, actualAmount, myBalance)

	if operation != nil {
		s.roomService.broadcastWithdraw(roomID, userID, actualAmount, myBalance, tableBalance, operation.CreatedAt)
	}

	return myBalance, tableBalance, actualAmount, nil
}

// ForceTransfer 将桌面积分强制转移给指定用户
func (s *OperationService) ForceTransfer(roomID, userID, targetUserID uint) (int, int, int, int, error) {
	var actorBalance, targetBalance, tableBalance, transferredAmount int
	var operation *models.RoomOperation

	err := models.DB.Transaction(func(tx *gorm.DB) error {
		// 检查操作者是否在房间中
		var actorMember models.RoomMember
		if err := tx.Where("room_id = ? AND user_id = ?", roomID, userID).First(&actorMember).Error; err != nil {
			return errors.New("您不在该房间中")
		}

		// 检查目标用户是否在房间中
		var targetMember models.RoomMember
		if err := tx.Where("room_id = ? AND user_id = ?", roomID, targetUserID).First(&targetMember).Error; err != nil {
			return errors.New("目标用户不在房间中")
		}

		available := s.roomService.CalculateTableBalanceWithDB(tx, roomID)
		if available <= 0 {
			return errors.New("桌面没有可转移的积分")
		}

		transferredAmount = available

		if err := s.roomService.UpdateUserBalanceWithDB(tx, roomID, targetUserID, transferredAmount); err != nil {
			return err
		}

		var err error
		targetBalance, err = s.roomService.GetUserBalanceWithDB(tx, roomID, targetUserID)
		if err != nil {
			return err
		}

		actorBalance, err = s.roomService.GetUserBalanceWithDB(tx, roomID, userID)
		if err != nil {
			return err
		}

		var targetUser models.User
		desc := fmt.Sprintf("将桌面%d积分转移给用户%d", transferredAmount, targetUserID)
		if err := tx.First(&targetUser, targetUserID).Error; err == nil {
			desc = fmt.Sprintf("将桌面%d积分转移给%s", transferredAmount, targetUser.Nickname)
		}

		amountCopy := transferredAmount
		targetCopy := targetUserID
		op, err := s.roomService.recordOperationWithDB(tx, roomID, userID, models.OpTypeForceTransfer, &amountCopy, &targetCopy, desc)
		if err != nil {
			return err
		}
		operation = op

		tableBalance = s.roomService.CalculateTableBalanceWithDB(tx, roomID)

		return nil
	})

	if err != nil {
		log.Printf("积分强制转移失败: RoomID=%d, UserID=%d, TargetUserID=%d, %v", roomID, userID, targetUserID, err)
		return 0, 0, 0, 0, err
	}

	log.Printf("积分强制转移成功: RoomID=%d, UserID=%d, TargetUserID=%d, Amount=%d", roomID, userID, targetUserID, transferredAmount)

	if operation != nil {
		s.roomService.broadcastForceTransfer(roomID, userID, targetUserID, transferredAmount, actorBalance, targetBalance, tableBalance, operation.CreatedAt)
	}

	return actorBalance, targetBalance, tableBalance, transferredAmount, nil
}

// NiuniuBet 牛牛下注（给某人下注）
func (s *OperationService) NiuniuBet(roomID, userID uint, bets []NiuniuBetItem) (int, int, error) {
	totalAmount := 0
	var myBalance, tableBalance int
	var operation *models.RoomOperation
	betDetails := make([]NiuniuBetDetail, 0, len(bets))

	// 使用事务确保原子性
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		// 处理每个下注
		for _, bet := range bets {
			if bet.Amount <= 0 {
				return errors.New("下注金额必须大于0")
			}

			totalAmount += bet.Amount

			// 创建下注记录
			betRecord := models.BetRecord{
				RoomID:     roomID,
				FromUserID: userID,
				ToUserID:   bet.ToUserID,
				Amount:     bet.Amount,
			}

			err := tx.Create(&betRecord).Error
			if err != nil {
				return err
			}

			var targetUser models.User
			if err := tx.First(&targetUser, bet.ToUserID).Error; err != nil {
				betDetails = append(betDetails, NiuniuBetDetail{
					ToUserID: bet.ToUserID,
					Amount:   bet.Amount,
				})
			} else {
				betDetails = append(betDetails, NiuniuBetDetail{
					ToUserID:   bet.ToUserID,
					ToNickname: targetUser.Nickname,
					Amount:     bet.Amount,
				})
			}
		}

		// 更新下注者的积分
		err := s.roomService.UpdateUserBalanceWithDB(tx, roomID, userID, -totalAmount)
		if err != nil {
			return err
		}

		// 获取更新后的积分
		myBalance, err = s.roomService.GetUserBalanceWithDB(tx, roomID, userID)
		if err != nil {
			return err
		}

		// 记录操作
		descData, _ := json.Marshal(betDetails)
		amountCopy := totalAmount
		op, err := s.roomService.recordOperationWithDB(tx, roomID, userID, models.OpTypeNiuniuBet, &amountCopy, nil, string(descData))
		if err != nil {
			return err
		}
		operation = op

		// 记录完成后重新计算桌面积分
		tableBalance = s.roomService.CalculateTableBalanceWithDB(tx, roomID)

		return nil
	})

	if err != nil {
		log.Printf("牛牛下注失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
		return 0, 0, err
	}

	log.Printf("牛牛下注成功: RoomID=%d, UserID=%d, TotalAmount=%d", roomID, userID, totalAmount)

	if operation != nil {
		s.roomService.broadcastNiuniuBet(roomID, userID, betDetails, totalAmount, myBalance, tableBalance, operation.CreatedAt)
	}

	return myBalance, totalAmount, nil
}

// GetOperations 获取操作历史
func (s *OperationService) GetOperations(roomID, userID uint, limit, offset int, includeAll bool) ([]map[string]interface{}, int64, error) {
	// 获取用户加入房间的时间
	var member models.RoomMember
	err := models.DB.Where("room_id = ? AND user_id = ?", roomID, userID).
		Order("joined_at DESC").
		First(&member).Error

	if err != nil {
		return nil, 0, errors.New("您不在该房间中")
	}

	// 查询操作记录（默认只返回用户加入后的记录）
	var operations []models.RoomOperation
	query := models.DB.Where("room_id = ?", roomID).
		Order("created_at DESC")

	if !includeAll {
		query = query.Where("created_at >= ?", member.JoinedAt)
	}

	// 获取总数
	var total int64
	query.Model(&models.RoomOperation{}).Count(&total)

	// 分页查询
	if limit <= 0 {
		err = query.Limit(-1).Offset(-1).Find(&operations).Error
	} else {
		err = query.Limit(limit).Offset(offset).Find(&operations).Error
	}
	if err != nil {
		return nil, 0, err
	}

	// 组装结果
	result := make([]map[string]interface{}, 0, len(operations))
	for _, op := range operations {
		// 获取用户昵称
		var user models.User
		models.DB.First(&user, op.UserID)

		opMap := map[string]interface{}{
			"id":             op.ID,
			"user_id":        op.UserID,
			"nickname":       user.Nickname,
			"operation_type": op.OperationType,
			"description":    op.Description,
			"created_at":     op.CreatedAt,
		}

		if op.Amount != nil {
			opMap["amount"] = *op.Amount
		}

		if op.TargetUserID != nil {
			opMap["target_user_id"] = *op.TargetUserID
			var targetUser models.User
			if err := models.DB.First(&targetUser, *op.TargetUserID).Error; err == nil {
				opMap["target_nickname"] = targetUser.Nickname
			}
		}

		result = append(result, opMap)
	}

	return result, total, nil
}

// GetHistoryAmounts 获取用户历史操作金额
func (s *OperationService) GetHistoryAmounts(roomID, userID uint) ([]int, []int, error) {
	// 查询下注历史（最近6条）
	var betOps []models.RoomOperation
	models.DB.Where("room_id = ? AND user_id = ? AND operation_type IN (?, ?)",
		roomID, userID, models.OpTypeBet, models.OpTypeNiuniuBet).
		Order("created_at DESC").
		Limit(6).
		Find(&betOps)

	betAmounts := make([]int, 0, len(betOps))
	for _, op := range betOps {
		if op.Amount != nil {
			betAmounts = append(betAmounts, *op.Amount)
		}
	}

	// 查询收回历史（最近6条）
	var withdrawOps []models.RoomOperation
	models.DB.Where("room_id = ? AND user_id = ? AND operation_type = ?",
		roomID, userID, models.OpTypeWithdraw).
		Order("created_at DESC").
		Limit(6).
		Find(&withdrawOps)

	withdrawAmounts := make([]int, 0, len(withdrawOps))
	for _, op := range withdrawOps {
		if op.Amount != nil {
			withdrawAmounts = append(withdrawAmounts, *op.Amount)
		}
	}

	return betAmounts, withdrawAmounts, nil
}

// 结算相关方法将在settlement_service.go中实现
