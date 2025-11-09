package services

import (
	"poker_score_backend/models"
	"strings"
	"time"
)

// RecordService 战绩统计服务
type RecordService struct{}

type tonightRecordSummary struct {
	UserID     uint
	ManualChip int
	ManualRmb  float64
	AutoChip   int
	AutoRmb    float64
	ActiveChip int
	ActiveRmb  float64
	Nickname   string
	IsMe       bool
}

func ensureRecordSummary(cache map[uint]*tonightRecordSummary, userID uint) *tonightRecordSummary {
	summary, exists := cache[userID]
	if !exists {
		summary = &tonightRecordSummary{
			UserID: userID,
		}
		cache[userID] = summary
	}

	return summary
}

// NewRecordService 创建战绩统计服务
func NewRecordService() *RecordService {
	return &RecordService{}
}

// GetTonightRecords 获取今晚战绩
func (s *RecordService) GetTonightRecords(userID uint, startTime, endTime *time.Time) (map[string]interface{}, error) {
	// 如果没有提供时间，自动计算
	var start, end time.Time
	if startTime == nil || endTime == nil {
		start, end = s.calculateDefaultTimeRange()
	} else {
		start = *startTime
		end = *endTime
	}

	// 查找用户在时间段内加入过的房间
	var roomIDs []uint
	err := models.DB.Model(&models.RoomMember{}).
		Where("user_id = ? AND joined_at BETWEEN ? AND ?", userID, start, end).
		Distinct("room_id").
		Pluck("room_id", &roomIDs).Error
	if err != nil {
		return nil, err
	}

	// 使用BFS查找"今晚一起玩过的好友"
	friendIDs := s.findFriendsWithBFS(roomIDs, start, end)
	friendSet := make(map[uint]struct{})
	friendSet[userID] = struct{}{}
	for _, fid := range friendIDs {
		friendSet[fid] = struct{}{}
	}

	buildIDList := func(set map[uint]struct{}) []uint {
		ids := make([]uint, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}
		return ids
	}

	summaries := make(map[uint]*tonightRecordSummary)

	settlementQueryIDs := buildIDList(friendSet)
	if len(settlementQueryIDs) > 0 {
		var settlements []models.Settlement
		if err := models.DB.Where("user_id IN ? AND settled_at BETWEEN ? AND ?", settlementQueryIDs, start, end).
			Find(&settlements).Error; err != nil {
			return nil, err
		}

		for _, settlement := range settlements {
			friendSet[settlement.UserID] = struct{}{}
			summary := ensureRecordSummary(summaries, settlement.UserID)
			if strings.HasPrefix(settlement.SettlementBatch, "auto-") {
				summary.AutoChip += settlement.ChipAmount
				summary.AutoRmb += settlement.RmbAmount
			} else {
				summary.ManualChip += settlement.ChipAmount
				summary.ManualRmb += settlement.RmbAmount
			}
		}
	}

	activeQueryIDs := buildIDList(friendSet)
	if len(activeQueryIDs) > 0 {
		var activeMembers []models.RoomMember
		if err := models.DB.Where("user_id IN ?", activeQueryIDs).
			Find(&activeMembers).Error; err != nil {
			return nil, err
		}

		roomIDSet := make(map[uint]struct{})
		for _, member := range activeMembers {
			roomIDSet[member.RoomID] = struct{}{}
		}

		if len(roomIDSet) > 0 {
			activeRoomIDs := make([]uint, 0, len(roomIDSet))
			for rid := range roomIDSet {
				activeRoomIDs = append(activeRoomIDs, rid)
			}

			var activeRooms []models.Room
			if err := models.DB.Where("id IN ? AND status = ?", activeRoomIDs, "active").
				Find(&activeRooms).Error; err != nil {
				return nil, err
			}

			roomMap := make(map[uint]models.Room, len(activeRooms))
			for _, room := range activeRooms {
				roomMap[room.ID] = room
			}

			type roomUserKey struct {
				RoomID uint
				UserID uint
			}

			var balances []models.UserBalance
			if err := models.DB.Where("room_id IN ? AND user_id IN ?", activeRoomIDs, activeQueryIDs).
				Find(&balances).Error; err != nil {
				return nil, err
			}

			balanceMap := make(map[roomUserKey]int, len(balances))
			for _, balance := range balances {
				balanceMap[roomUserKey{RoomID: balance.RoomID, UserID: balance.UserID}] = balance.Balance
			}

			for _, member := range activeMembers {
				room, ok := roomMap[member.RoomID]
				if !ok {
					continue
				}

				balance := balanceMap[roomUserKey{RoomID: member.RoomID, UserID: member.UserID}]
				if balance == 0 {
					continue
				}

				summary := ensureRecordSummary(summaries, member.UserID)
				summary.ActiveChip += balance
				summary.ActiveRmb += calculateRmbAmount(balance, room.ChipRate)
				friendSet[member.UserID] = struct{}{}
			}
		}
	}

	for id := range friendSet {
		ensureRecordSummary(summaries, id)
	}

	ensureRecordSummary(summaries, userID).IsMe = true

	userIDs := make([]uint, 0, len(summaries))
	for id := range summaries {
		userIDs = append(userIDs, id)
	}

	if len(userIDs) > 0 {
		var users []models.User
		if err := models.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, err
		}

		for _, user := range users {
			if summary, exists := summaries[user.ID]; exists {
				summary.Nickname = user.Nickname
				if user.ID == userID {
					summary.IsMe = true
				}
			}
		}
	}

	friendsRecords := make([]map[string]interface{}, 0, len(summaries))
	totalCheck := 0.0
	for _, summary := range summaries {
		settledChip := summary.ManualChip + summary.AutoChip
		settledRmb := summary.ManualRmb + summary.AutoRmb
		totalChip := settledChip + summary.ActiveChip
		totalRmb := settledRmb + summary.ActiveRmb

		record := map[string]interface{}{
			"user_id":                summary.UserID,
			"nickname":               summary.Nickname,
			"is_me":                  summary.IsMe,
			"manual_settlement_chip": summary.ManualChip,
			"manual_settlement_rmb":  summary.ManualRmb,
			"auto_settlement_chip":   summary.AutoChip,
			"auto_settlement_rmb":    summary.AutoRmb,
			"active_balance_chip":    summary.ActiveChip,
			"active_balance_rmb":     summary.ActiveRmb,
			"settled_chip":           settledChip,
			"settled_rmb":            settledRmb,
			"total_chip":             totalChip,
			"total_rmb":              totalRmb,
		}

		friendsRecords = append(friendsRecords, record)
		totalCheck += totalRmb
	}

	// 查询用户当前在的房间
	var currentRoomMembers []models.RoomMember
	models.DB.Where("user_id = ?", userID).Find(&currentRoomMembers)

	currentRooms := make([]map[string]interface{}, 0, len(currentRoomMembers))
	for _, member := range currentRoomMembers {
		var room models.Room
		if err := models.DB.First(&room, member.RoomID).Error; err == nil && room.Status == "active" {
			currentRooms = append(currentRooms, map[string]interface{}{
				"room_id":   room.ID,
				"room_code": room.RoomCode,
				"room_type": room.RoomType,
			})
		}
	}

	return map[string]interface{}{
		"time_range": map[string]interface{}{
			"start": start,
			"end":   end,
		},
		"current_rooms":   currentRooms,
		"friends_records": friendsRecords,
		"total_check":     totalCheck,
	}, nil
}

// calculateDefaultTimeRange 计算默认时间段
func (s *RecordService) calculateDefaultTimeRange() (time.Time, time.Time) {
	now := time.Now()
	today7am := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())

	var start, end time.Time
	if now.Before(today7am) {
		// 当前时间在7:00am之前，统计昨天7:00am到今天7:00am
		start = today7am.Add(-24 * time.Hour)
		end = today7am
	} else {
		// 当前时间在7:00am之后，统计今天7:00am到明天7:00am
		start = today7am
		end = today7am.Add(24 * time.Hour)
	}

	return start, end
}

// findFriendsWithBFS 使用BFS查找"今晚一起玩过的好友"
func (s *RecordService) findFriendsWithBFS(initialRoomIDs []uint, start, end time.Time) []uint {
	visited := make(map[uint]bool)
	visitedRooms := make(map[uint]bool)
	queue := initialRoomIDs

	for _, roomID := range initialRoomIDs {
		visitedRooms[roomID] = true
	}

	for len(queue) > 0 {
		roomID := queue[0]
		queue = queue[1:]

		// 查找该房间在时间段内的所有成员
		var members []models.RoomMember
		models.DB.Where("room_id = ? AND joined_at BETWEEN ? AND ?", roomID, start, end).
			Find(&members)

		for _, member := range members {
			if !visited[member.UserID] {
				visited[member.UserID] = true

				// 查找这个用户在时间段内加入的其他房间
				var otherRoomIDs []uint
				models.DB.Model(&models.RoomMember{}).
					Where("user_id = ? AND joined_at BETWEEN ? AND ?", member.UserID, start, end).
					Distinct("room_id").
					Pluck("room_id", &otherRoomIDs)

				for _, otherRoomID := range otherRoomIDs {
					if !visitedRooms[otherRoomID] {
						visitedRooms[otherRoomID] = true
						queue = append(queue, otherRoomID)
					}
				}
			}
		}
	}

	// 转换为slice
	friendIDs := make([]uint, 0, len(visited))
	for userID := range visited {
		friendIDs = append(friendIDs, userID)
	}

	return friendIDs
}
