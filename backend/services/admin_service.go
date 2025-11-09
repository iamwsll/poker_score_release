package services

import (
	"poker_score_backend/models"
	"time"
)

// AdminService 后台管理服务
type AdminService struct{}

// NewAdminService 创建后台管理服务
func NewAdminService() *AdminService {
	return &AdminService{}
}

// GetUsers 获取用户列表
func (s *AdminService) GetUsers(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// 查询总数
	err := models.DB.Model(&models.User{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = models.DB.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetRooms 获取房间列表
func (s *AdminService) GetRooms(status string, page, pageSize int) ([]map[string]interface{}, int64, error) {
	var rooms []models.Room
	var total int64

	query := models.DB.Model(&models.Room{})

	// 按状态过滤
	if status != "all" && status != "" {
		query = query.Where("status = ?", status)
	}

	// 查询总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&rooms).Error
	if err != nil {
		return nil, 0, err
	}

	// 组装结果
	result := make([]map[string]interface{}, 0, len(rooms))
	for _, room := range rooms {
		// 获取创建者昵称
		var creator models.User
		models.DB.First(&creator, room.CreatedBy)

		// 统计成员数量
		var memberCount, onlineCount int64
		models.DB.Model(&models.RoomMember{}).Where("room_id = ?", room.ID).Count(&memberCount)
		models.DB.Model(&models.RoomMember{}).Where("room_id = ? AND status = ?", room.ID, "online").Count(&onlineCount)

		result = append(result, map[string]interface{}{
			"id":               room.ID,
			"room_code":        room.RoomCode,
			"room_type":        room.RoomType,
			"chip_rate":        room.ChipRate,
			"status":           room.Status,
			"created_by":       room.CreatedBy,
			"creator_nickname": creator.Nickname,
			"member_count":     memberCount,
			"online_count":     onlineCount,
			"created_at":       room.CreatedAt,
		})
	}

	return result, total, nil
}

// GetRoomDetails 获取房间详细信息
func (s *AdminService) GetRoomDetails(roomID uint) (map[string]interface{}, error) {
	// 查询房间信息
	var room models.Room
	err := models.DB.First(&room, roomID).Error
	if err != nil {
		return nil, err
	}

	// 查询所有成员
	var members []models.RoomMember
	err = models.DB.Where("room_id = ?", roomID).Order("joined_at DESC").Find(&members).Error
	if err != nil {
		return nil, err
	}

	// 组装成员信息
	memberList := make([]map[string]interface{}, 0, len(members))
	for _, member := range members {
		var user models.User
		models.DB.First(&user, member.UserID)

		// 获取积分余额
		var balance models.UserBalance
		balanceAmount := 0
		if err := models.DB.Where("room_id = ? AND user_id = ?", roomID, member.UserID).First(&balance).Error; err == nil {
			balanceAmount = balance.Balance
		}

		memberList = append(memberList, map[string]interface{}{
			"user_id":   user.ID,
			"nickname":  user.Nickname,
			"balance":   balanceAmount,
			"status":    member.Status,
			"joined_at": member.JoinedAt,
			"left_at":   member.LeftAt,
		})
	}

	// 查询操作记录（最近100条）
	var operations []models.RoomOperation
	models.DB.Where("room_id = ?", roomID).Order("created_at DESC").Limit(100).Find(&operations)

	operationList := make([]map[string]interface{}, 0, len(operations))
	for _, op := range operations {
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

		operationList = append(operationList, opMap)
	}

	return map[string]interface{}{
		"room": map[string]interface{}{
			"id":        room.ID,
			"room_code": room.RoomCode,
			"room_type": room.RoomType,
			"chip_rate": room.ChipRate,
			"status":    room.Status,
		},
		"members":    memberList,
		"operations": operationList,
	}, nil
}

// GetUserSettlements 获取用户历史盈亏
func (s *AdminService) GetUserSettlements(userID uint, startTime, endTime *time.Time) (map[string]interface{}, error) {
	// 查询用户信息
	var user models.User
	err := models.DB.First(&user, userID).Error
	if err != nil {
		return nil, err
	}

	// 查询结算记录
	query := models.DB.Where("user_id = ?", userID)

	if startTime != nil {
		query = query.Where("settled_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("settled_at <= ?", *endTime)
	}

	var settlements []models.Settlement
	err = query.Order("settled_at DESC").Find(&settlements).Error
	if err != nil {
		return nil, err
	}

	// 组装结果
	settlementList := make([]map[string]interface{}, 0, len(settlements))
	totalChip := 0
	totalRmb := 0.0

	for _, settlement := range settlements {
		// 获取房间信息
		var room models.Room
		models.DB.First(&room, settlement.RoomID)

		settlementList = append(settlementList, map[string]interface{}{
			"room_id":     room.ID,
			"room_code":   room.RoomCode,
			"room_type":   room.RoomType,
			"chip_amount": settlement.ChipAmount,
			"rmb_amount":  settlement.RmbAmount,
			"settled_at":  settlement.SettledAt,
		})

		totalChip += settlement.ChipAmount
		totalRmb += settlement.RmbAmount
	}

	return map[string]interface{}{
		"user": map[string]interface{}{
			"id":       user.ID,
			"nickname": user.Nickname,
		},
		"settlements": settlementList,
		"total_chip":  totalChip,
		"total_rmb":   totalRmb,
	}, nil
}

// GetRoomMemberHistory 获取用户进出房间历史
func (s *AdminService) GetRoomMemberHistory(userID, roomID *uint, page, pageSize int) ([]map[string]interface{}, int64, error) {
	query := models.DB.Model(&models.RoomMember{})

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if roomID != nil {
		query = query.Where("room_id = ?", *roomID)
	}

	// 查询总数
	var total int64
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	var members []models.RoomMember
	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("joined_at DESC").Find(&members).Error
	if err != nil {
		return nil, 0, err
	}

	// 组装结果
	result := make([]map[string]interface{}, 0, len(members))
	for _, member := range members {
		var user models.User
		var room models.Room
		models.DB.First(&user, member.UserID)
		models.DB.First(&room, member.RoomID)

		// 计算在线时长（分钟）
		var durationMinutes int64
		if member.LeftAt != nil {
			durationMinutes = int64(member.LeftAt.Sub(member.JoinedAt).Minutes())
		}

		result = append(result, map[string]interface{}{
			"id":               member.ID,
			"room_id":          room.ID,
			"room_code":        room.RoomCode,
			"user_id":          user.ID,
			"nickname":         user.Nickname,
			"joined_at":        member.JoinedAt,
			"left_at":          member.LeftAt,
			"status":           member.Status,
			"duration_minutes": durationMinutes,
		})
	}

	return result, total, nil
}
