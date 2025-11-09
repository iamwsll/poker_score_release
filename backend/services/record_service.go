package services

import (
	"poker_score_backend/models"
	"time"
)

// RecordService 战绩统计服务
type RecordService struct{}

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

	// 查询这些好友的结算记录
	var settlements []models.Settlement
	err = models.DB.Where("user_id IN ? AND settled_at BETWEEN ? AND ?", friendIDs, start, end).
		Find(&settlements).Error
	
	if err != nil {
		return nil, err
	}

	// 按用户聚合结算记录
	userRecords := make(map[uint]map[string]interface{})
	for _, settlement := range settlements {
		if _, exists := userRecords[settlement.UserID]; !exists {
			userRecords[settlement.UserID] = map[string]interface{}{
				"user_id":    settlement.UserID,
				"total_chip": 0,
				"total_rmb":  0.0,
			}
		}
		
		record := userRecords[settlement.UserID]
		record["total_chip"] = record["total_chip"].(int) + settlement.ChipAmount
		record["total_rmb"] = record["total_rmb"].(float64) + settlement.RmbAmount
	}

	// 获取用户昵称并构造结果
	friendsRecords := make([]map[string]interface{}, 0, len(userRecords))
	for uid, record := range userRecords {
		var user models.User
		models.DB.First(&user, uid)
		
		record["nickname"] = user.Nickname
		record["is_me"] = (uid == userID)
		friendsRecords = append(friendsRecords, record)
	}

	// 查询用户当前在的房间
	var currentRoomMembers []models.RoomMember
	models.DB.Where("user_id = ? AND left_at IS NULL", userID).Find(&currentRoomMembers)
	
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

	// 计算总和校验
	totalCheck := 0.0
	for _, record := range friendsRecords {
		totalCheck += record["total_rmb"].(float64)
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

