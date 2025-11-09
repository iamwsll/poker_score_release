package services

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"poker_score_backend/models"
	"poker_score_backend/utils"

	"gorm.io/gorm"
)

// AdminService 后台管理服务
type AdminService struct{}

// NewAdminService 创建后台管理服务
func NewAdminService() *AdminService {
	return &AdminService{}
}

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrPhoneAlreadyExists = errors.New("phone already exists")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidPhone       = errors.New("invalid phone")
	ErrInvalidNickname    = errors.New("invalid nickname")
	ErrInvalidPassword    = errors.New("invalid password")
)

// UpdateUserInput 更新用户请求体
type UpdateUserInput struct {
	Phone    string
	Nickname string
	Role     string
	Password *string
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

// UpdateUser 更新用户信息
func (s *AdminService) UpdateUser(userID uint, input UpdateUserInput) (*models.User, error) {
	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	phone := strings.TrimSpace(input.Phone)
	nickname := strings.TrimSpace(input.Nickname)
	role := strings.TrimSpace(input.Role)

	if phone == "" || len(phone) != 11 || !isAllDigits(phone) {
		return nil, ErrInvalidPhone
	}
	if nickname == "" || len([]rune(nickname)) > 50 {
		return nil, ErrInvalidNickname
	}
	if role != "admin" && role != "user" {
		return nil, ErrInvalidRole
	}

	if phone != user.Phone {
		var count int64
		if err := models.DB.Model(&models.User{}).
			Where("phone = ? AND id <> ?", phone, userID).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrPhoneAlreadyExists
		}
	}

	updates := map[string]interface{}{
		"phone":    phone,
		"nickname": nickname,
		"role":     role,
	}

	if input.Password != nil {
		password := strings.TrimSpace(*input.Password)
		if password == "" || len(password) < 6 {
			return nil, ErrInvalidPassword
		}
		passwordHash, err := utils.HashPassword(password)
		if err != nil {
			return nil, err
		}
		updates["password_hash"] = passwordHash
	}

	if err := models.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := models.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	return &user, nil
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
func (s *AdminService) GetRoomDetails(roomID uint, opPage, opPageSize int) (map[string]interface{}, error) {
	if opPage <= 0 {
		opPage = 1
	}
	if opPageSize <= 0 {
		opPageSize = 20
	} else if opPageSize > 200 {
		opPageSize = 200
	}
	// 查询房间信息
	var room models.Room
	err := models.DB.First(&room, roomID).Error
	if err != nil {
		return nil, err
	}

	// 计算桌面积分（无需依赖管理员是否在房间中）
	tableBalance := (&RoomService{}).CalculateTableBalance(room.ID)

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

	// 查询操作记录（分页）
	var (
		operations       []models.RoomOperation
		totalOperations  int64
		operationRecords = models.DB.Model(&models.RoomOperation{}).Where("room_id = ?", roomID)
	)

	if err := operationRecords.Count(&totalOperations).Error; err != nil {
		return nil, err
	}

	offset := (opPage - 1) * opPageSize
	if err := operationRecords.Order("created_at DESC").Offset(offset).Limit(opPageSize).Find(&operations).Error; err != nil {
		return nil, err
	}

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

	result := map[string]interface{}{
		"room": map[string]interface{}{
			"id":            room.ID,
			"room_code":     room.RoomCode,
			"room_type":     room.RoomType,
			"chip_rate":     room.ChipRate,
			"status":        room.Status,
			"dissolved_at":  room.DissolvedAt,
			"table_balance": tableBalance,
		},
		"members": memberList,
		"operations": map[string]interface{}{
			"list":      operationList,
			"total":     totalOperations,
			"page":      opPage,
			"page_size": opPageSize,
		},
		// 为兼容历史调用保留独立字段，便于前端直接取用
		"table_balance": tableBalance,
	}

	return result, nil
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
	eventTypes := []string{
		models.OpTypeCreate,
		models.OpTypeJoin,
		models.OpTypeLeave,
		models.OpTypeReturn,
		models.OpTypeKick,
		models.OpTypeSettlementConfirmed,
	}

	query := models.DB.Model(&models.RoomOperation{}).Where("operation_type IN ?", eventTypes)

	if userID != nil {
		query = query.Where("(user_id = ?) OR (target_user_id = ?)", *userID, *userID)
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
	var operations []models.RoomOperation
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&operations).Error
	if err != nil {
		return nil, 0, err
	}

	if len(operations) == 0 {
		return []map[string]interface{}{}, total, nil
	}

	roomIDSet := make(map[uint]struct{})
	userIDSet := make(map[uint]struct{})
	targetIDSet := make(map[uint]struct{})

	for _, op := range operations {
		roomIDSet[op.RoomID] = struct{}{}
		userIDSet[op.UserID] = struct{}{}
		if op.TargetUserID != nil {
			targetIDSet[*op.TargetUserID] = struct{}{}
		}
	}

	uniqueUserIDs := make([]uint, 0, len(userIDSet)+len(targetIDSet))
	for id := range userIDSet {
		uniqueUserIDs = append(uniqueUserIDs, id)
	}
	for id := range targetIDSet {
		if _, exists := userIDSet[id]; !exists {
			uniqueUserIDs = append(uniqueUserIDs, id)
		}
	}

	userMap := make(map[uint]models.User)
	if len(uniqueUserIDs) > 0 {
		var users []models.User
		if err := models.DB.Where("id IN ?", uniqueUserIDs).Find(&users).Error; err != nil {
			return nil, 0, err
		}
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	roomIDs := make([]uint, 0, len(roomIDSet))
	for id := range roomIDSet {
		roomIDs = append(roomIDs, id)
	}

	roomMap := make(map[uint]models.Room)
	if len(roomIDs) > 0 {
		var rooms []models.Room
		if err := models.DB.Where("id IN ?", roomIDs).Find(&rooms).Error; err != nil {
			return nil, 0, err
		}
		for _, room := range rooms {
			roomMap[room.ID] = room
		}
	}

	result := make([]map[string]interface{}, 0, len(operations))
	for _, op := range operations {
		record := map[string]interface{}{
			"id":             op.ID,
			"room_id":        op.RoomID,
			"room_code":      "",
			"room_type":      "",
			"user_id":        op.UserID,
			"user_nickname":  "",
			"operation_type": op.OperationType,
			"description":    op.Description,
			"created_at":     op.CreatedAt,
		}

		if room, ok := roomMap[op.RoomID]; ok {
			record["room_code"] = room.RoomCode
			record["room_type"] = room.RoomType
		}

		if user, ok := userMap[op.UserID]; ok {
			record["user_nickname"] = user.Nickname
		}

		if op.Amount != nil {
			record["amount"] = *op.Amount
		}

		if op.TargetUserID != nil {
			record["target_user_id"] = *op.TargetUserID
			if targetUser, ok := userMap[*op.TargetUserID]; ok {
				record["target_nickname"] = targetUser.Nickname
			}
		}

		if op.Description != "" {
			var metadata interface{}
			if err := json.Unmarshal([]byte(op.Description), &metadata); err == nil {
				record["metadata"] = metadata
			}
		}

		result = append(result, record)
	}

	return result, total, nil
}

func isAllDigits(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
