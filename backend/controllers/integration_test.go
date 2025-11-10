package controllers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"poker_score_backend/models"
	"poker_score_backend/testutil"

	"github.com/stretchr/testify/require"
)

const testUserPassword = "123456"

type testUser struct {
	Client   *testutil.APIClient
	UserID   uint
	Phone    string
	Nickname string
}

type roomMember struct {
	UserID   uint   `json:"user_id"`
	Nickname string `json:"nickname"`
	Balance  int    `json:"balance"`
	Status   string `json:"status"`
}

type roomDetails struct {
	RoomID       uint         `json:"room_id"`
	RoomCode     string       `json:"room_code"`
	RoomType     string       `json:"room_type"`
	ChipRate     string       `json:"chip_rate"`
	Status       string       `json:"status"`
	CreatedBy    uint         `json:"created_by"`
	TableBalance int          `json:"table_balance"`
	MyBalance    int          `json:"my_balance"`
	Members      []roomMember `json:"members"`
}

type operationView struct {
	ID             uint      `json:"id"`
	UserID         uint      `json:"user_id"`
	Nickname       string    `json:"nickname"`
	OperationType  string    `json:"operation_type"`
	Amount         *int      `json:"amount,omitempty"`
	TargetUserID   *uint     `json:"target_user_id,omitempty"`
	TargetNickname string    `json:"target_nickname,omitempty"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
}

type settlementPlanView struct {
	FromUserID   uint    `json:"from_user_id"`
	FromNickname string  `json:"from_nickname"`
	ToUserID     uint    `json:"to_user_id"`
	ToNickname   string  `json:"to_nickname"`
	ChipAmount   int     `json:"chip_amount"`
	RmbAmount    float64 `json:"rmb_amount"`
	Description  string  `json:"description"`
}

type tonightRecord struct {
	UserID      uint   `json:"user_id"`
	Nickname    string `json:"nickname"`
	IsMe        bool   `json:"is_me"`
	SettledChip int    `json:"settled_chip"`
	TotalChip   int    `json:"total_chip"`
}

func registerUser(t *testing.T, client *testutil.APIClient, nickname string) testUser {
	t.Helper()

	phone := uniquePhone()
	resp, err := client.Do(http.MethodPost, "/api/auth/register", map[string]string{
		"phone":    phone,
		"nickname": nickname,
		"password": testUserPassword,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			User struct {
				ID       uint   `json:"id"`
				Phone    string `json:"phone"`
				Nickname string `json:"nickname"`
				Role     string `json:"role"`
			} `json:"user"`
			SessionID string `json:"session_id"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &body)
	require.Equal(t, 0, body.Code)
	require.Equal(t, "注册成功", body.Message)
	require.Equal(t, phone, body.Data.User.Phone)
	require.NotEmpty(t, body.Data.SessionID)
	require.NotNil(t, client.Cookie(testSessionCookieName))

	return testUser{
		Client:   client,
		UserID:   body.Data.User.ID,
		Phone:    phone,
		Nickname: nickname,
	}
}

func loginUser(t *testing.T, client *testutil.APIClient, phone, password string) {
	t.Helper()

	resp, err := client.Do(http.MethodPost, "/api/auth/login", map[string]string{
		"phone":    phone,
		"password": password,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SessionID string `json:"session_id"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &body)
	require.Equal(t, 0, body.Code)
	require.Equal(t, "登录成功", body.Message)
	require.NotEmpty(t, body.Data.SessionID)
	require.NotNil(t, client.Cookie(testSessionCookieName))
}

func TestFullIntegrationFlow(t *testing.T) {
	engine, _ := newTestEnv(t)

	owner := registerUser(t, testutil.NewAPIClient(engine), "房主甲")
	member := registerUser(t, testutil.NewAPIClient(engine), "房间成员乙")
	spectator := registerUser(t, testutil.NewAPIClient(engine), "旁观者丙")

	var createRoomResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			RoomID   uint   `json:"room_id"`
			RoomCode string `json:"room_code"`
			RoomType string `json:"room_type"`
			ChipRate string `json:"chip_rate"`
		} `json:"data"`
	}
	resp, err := owner.Client.Do(http.MethodPost, "/api/rooms", map[string]string{
		"room_type": "texas",
		"chip_rate": "20:1",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)
	decodeResponse(t, resp, &createRoomResp)
	require.Equal(t, 0, createRoomResp.Code)
	require.Equal(t, "房间创建成功", createRoomResp.Message)

	roomID := createRoomResp.Data.RoomID
	roomCode := createRoomResp.Data.RoomCode
	require.NotZero(t, roomID)
	require.Len(t, roomCode, 6)

	var joinResp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    roomDetails `json:"data"`
	}
	resp, err = member.Client.Do(http.MethodPost, "/api/rooms/join", map[string]string{"room_code": roomCode})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)
	decodeResponse(t, resp, &joinResp)
	require.Equal(t, 0, joinResp.Code)
	require.Equal(t, "加入房间成功", joinResp.Message)
	require.Equal(t, roomID, joinResp.Data.RoomID)
	require.GreaterOrEqual(t, len(joinResp.Data.Members), 2)

	resp, err = spectator.Client.Do(http.MethodPost, "/api/rooms/join", map[string]string{"room_code": roomCode})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var spectatorJoinResp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    roomDetails `json:"data"`
	}
	decodeResponse(t, resp, &spectatorJoinResp)
	require.Equal(t, 0, spectatorJoinResp.Code)
	require.Equal(t, roomID, spectatorJoinResp.Data.RoomID)
	require.GreaterOrEqual(t, len(spectatorJoinResp.Data.Members), 3)

	var roomDetailsResp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    roomDetails `json:"data"`
	}
	resp, err = owner.Client.Do(http.MethodGet, fmt.Sprintf("/api/rooms/%d", roomID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)
	decodeResponse(t, resp, &roomDetailsResp)
	require.Equal(t, 0, roomDetailsResp.Code)
	require.Equal(t, roomID, roomDetailsResp.Data.RoomID)
	require.Equal(t, owner.UserID, roomDetailsResp.Data.CreatedBy)

	resp, err = owner.Client.Do(http.MethodGet, "/api/rooms/last", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var lastRoomResp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    roomDetails `json:"data"`
	}
	decodeResponse(t, resp, &lastRoomResp)
	require.Equal(t, 0, lastRoomResp.Code)
	require.Equal(t, roomID, lastRoomResp.Data.RoomID)

	resp, err = owner.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/kick", roomID), map[string]uint{"user_id": spectator.UserID})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var kickResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	decodeResponse(t, resp, &kickResp)
	require.Equal(t, 0, kickResp.Code)
	require.Equal(t, "踢出成功", kickResp.Message)

	resp, err = owner.Client.Do(http.MethodGet, fmt.Sprintf("/api/rooms/%d", roomID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var afterKickResp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    roomDetails `json:"data"`
	}
	decodeResponse(t, resp, &afterKickResp)
	require.Equal(t, 0, afterKickResp.Code)
	foundSpectator := false
	for _, memberInfo := range afterKickResp.Data.Members {
		if memberInfo.UserID == spectator.UserID {
			foundSpectator = true
			require.Equal(t, "offline", memberInfo.Status)
		}
	}
	require.True(t, foundSpectator)

	var betResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MyBalance    int `json:"my_balance"`
			TableBalance int `json:"table_balance"`
		} `json:"data"`
	}
	resp, err = owner.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/bet", roomID), map[string]int{"amount": 500})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)
	decodeResponse(t, resp, &betResp)
	require.Equal(t, 0, betResp.Code)
	require.Equal(t, "下注成功", betResp.Message)
	require.Equal(t, -500, betResp.Data.MyBalance)
	require.Equal(t, 500, betResp.Data.TableBalance)

	resp, err = member.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/bet", roomID), map[string]int{"amount": 300})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var memberBetResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MyBalance    int `json:"my_balance"`
			TableBalance int `json:"table_balance"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &memberBetResp)
	require.Equal(t, 0, memberBetResp.Code)
	require.Equal(t, -300, memberBetResp.Data.MyBalance)
	require.Equal(t, 800, memberBetResp.Data.TableBalance)

	resp, err = owner.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/niuniu-bet", roomID), map[string]interface{}{
		"bets": []map[string]interface{}{
			{
				"to_user_id": member.UserID,
				"amount":     50,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var niuniuResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MyBalance   int `json:"my_balance"`
			TotalAmount int `json:"total_amount"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &niuniuResp)
	require.Equal(t, 0, niuniuResp.Code)
	require.Equal(t, "下注成功", niuniuResp.Message)
	require.Equal(t, -550, niuniuResp.Data.MyBalance)
	require.Equal(t, 50, niuniuResp.Data.TotalAmount)

	resp, err = owner.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/withdraw", roomID), map[string]int{"amount": 250})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var withdrawResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MyBalance    int `json:"my_balance"`
			TableBalance int `json:"table_balance"`
			ActualAmount int `json:"actual_amount"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &withdrawResp)
	require.Equal(t, 0, withdrawResp.Code)
	require.Equal(t, "收回成功", withdrawResp.Message)
	require.Equal(t, -300, withdrawResp.Data.MyBalance)
	require.Equal(t, 600, withdrawResp.Data.TableBalance)
	require.Equal(t, 250, withdrawResp.Data.ActualAmount)

	resp, err = owner.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/force-transfer", roomID), map[string]uint{"target_user_id": member.UserID})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var forceResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			TableBalance      int  `json:"table_balance"`
			TransferredAmount int  `json:"transferred_amount"`
			TargetUserID      uint `json:"target_user_id"`
			TargetBalance     int  `json:"target_balance"`
			ActorUserID       uint `json:"actor_user_id"`
			ActorBalance      int  `json:"actor_balance"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &forceResp)
	require.Equal(t, 0, forceResp.Code)
	require.Equal(t, "积分已转移", forceResp.Message)
	require.Equal(t, 0, forceResp.Data.TableBalance)
	require.Equal(t, 600, forceResp.Data.TransferredAmount)
	require.Equal(t, member.UserID, forceResp.Data.TargetUserID)
	require.Equal(t, 300, forceResp.Data.TargetBalance)
	require.Equal(t, owner.UserID, forceResp.Data.ActorUserID)
	require.Equal(t, -300, forceResp.Data.ActorBalance)

	resp, err = owner.Client.Do(http.MethodGet, fmt.Sprintf("/api/rooms/%d/operations?limit=50&offset=0", roomID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var operationsResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Operations []operationView `json:"operations"`
			Total      int             `json:"total"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &operationsResp)
	require.Equal(t, 0, operationsResp.Code)
	require.GreaterOrEqual(t, len(operationsResp.Data.Operations), 6)

	hasBet := false
	hasWithdraw := false
	hasForceTransfer := false
	hasNiuniu := false
	for _, op := range operationsResp.Data.Operations {
		switch op.OperationType {
		case models.OpTypeBet:
			hasBet = true
		case models.OpTypeWithdraw:
			hasWithdraw = true
		case models.OpTypeForceTransfer:
			hasForceTransfer = true
		case models.OpTypeNiuniuBet:
			hasNiuniu = true
		}
	}
	require.True(t, hasBet)
	require.True(t, hasWithdraw)
	require.True(t, hasForceTransfer)
	require.True(t, hasNiuniu)

	resp, err = owner.Client.Do(http.MethodGet, fmt.Sprintf("/api/rooms/%d/history-amounts", roomID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var historyResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			BetAmounts      []int `json:"bet_amounts"`
			WithdrawAmounts []int `json:"withdraw_amounts"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &historyResp)
	require.Equal(t, 0, historyResp.Code)
	require.NotEmpty(t, historyResp.Data.BetAmounts)
	require.NotEmpty(t, historyResp.Data.WithdrawAmounts)

	resp, err = owner.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/settlement/initiate", roomID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var initiateResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			CanSettle      bool                 `json:"can_settle"`
			TableBalance   int                  `json:"table_balance"`
			SettlementPlan []settlementPlanView `json:"settlement_plan"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &initiateResp)
	require.Equal(t, 0, initiateResp.Code)
	require.True(t, initiateResp.Data.CanSettle)
	require.Equal(t, 0, initiateResp.Data.TableBalance)
	require.NotEmpty(t, initiateResp.Data.SettlementPlan)
	require.Equal(t, owner.UserID, initiateResp.Data.SettlementPlan[0].FromUserID)
	require.Equal(t, member.UserID, initiateResp.Data.SettlementPlan[0].ToUserID)

	resp, err = owner.Client.Do(http.MethodPost, fmt.Sprintf("/api/rooms/%d/settlement/confirm", roomID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var confirmResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SettlementBatch string `json:"settlement_batch"`
			SettledAt       string `json:"settled_at"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &confirmResp)
	require.Equal(t, 0, confirmResp.Code)
	require.Equal(t, "结算完成", confirmResp.Message)
	require.NotEmpty(t, confirmResp.Data.SettlementBatch)
	_, err = time.Parse(time.RFC3339, confirmResp.Data.SettledAt)
	require.NoError(t, err)

	resp, err = owner.Client.Do(http.MethodGet, "/api/records/tonight", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var tonightResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			FriendsRecords []tonightRecord `json:"friends_records"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &tonightResp)
	require.Equal(t, 0, tonightResp.Code)
	require.NotEmpty(t, tonightResp.Data.FriendsRecords)

	findRecord := func(userID uint) (tonightRecord, bool) {
		for _, record := range tonightResp.Data.FriendsRecords {
			if record.UserID == userID {
				return record, true
			}
		}
		return tonightRecord{}, false
	}

	ownerRecord, ok := findRecord(owner.UserID)
	require.True(t, ok)
	require.NotZero(t, ownerRecord.TotalChip)

	memberRecord, ok := findRecord(member.UserID)
	require.True(t, ok)
	require.NotZero(t, memberRecord.TotalChip)

	start := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	end := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	resp, err = owner.Client.Do(http.MethodGet, fmt.Sprintf("/api/records/tonight?start_time=%s&end_time=%s", url.QueryEscape(start), url.QueryEscape(end)), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var tonightRangeResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			FriendsRecords []tonightRecord `json:"friends_records"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &tonightRangeResp)
	require.Equal(t, 0, tonightRangeResp.Code)
	require.NotEmpty(t, tonightRangeResp.Data.FriendsRecords)

	resp, err = owner.Client.Do(http.MethodGet, "/api/admin/users", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.Code)

	var forbiddenResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	decodeResponse(t, resp, &forbiddenResp)
	require.Equal(t, 403, forbiddenResp.Code)

	admin := registerUser(t, testutil.NewAPIClient(engine), "管理员丁")
	require.NoError(t, models.DB.Model(&models.User{}).Where("id = ?", admin.UserID).Update("role", "admin").Error)
	loginUser(t, admin.Client, admin.Phone, testUserPassword)

	resp, err = admin.Client.Do(http.MethodGet, "/api/admin/users?page=1&page_size=20", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var adminUsersResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Users    []models.User `json:"users"`
			Total    int           `json:"total"`
			Page     int           `json:"page"`
			PageSize int           `json:"page_size"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &adminUsersResp)
	require.Equal(t, 0, adminUsersResp.Code)
	require.NotEmpty(t, adminUsersResp.Data.Users)

	updatedNickname := member.Nickname + "-更新"
	resp, err = admin.Client.Do(http.MethodPut, fmt.Sprintf("/api/admin/users/%d", member.UserID), map[string]string{
		"phone":    member.Phone,
		"nickname": updatedNickname,
		"role":     "user",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var updateUserResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			User models.User `json:"user"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &updateUserResp)
	require.Equal(t, 0, updateUserResp.Code)
	require.Equal(t, "用户信息更新成功", updateUserResp.Message)
	require.Equal(t, updatedNickname, updateUserResp.Data.User.Nickname)

	resp, err = admin.Client.Do(http.MethodGet, "/api/admin/rooms?status=all&page=1&page_size=20", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var adminRoomsResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Rooms []struct {
				ID          uint   `json:"id"`
				RoomCode    string `json:"room_code"`
				RoomType    string `json:"room_type"`
				Status      string `json:"status"`
				MemberCount int    `json:"member_count"`
			} `json:"rooms"`
			Total int `json:"total"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &adminRoomsResp)
	require.Equal(t, 0, adminRoomsResp.Code)
	require.NotEmpty(t, adminRoomsResp.Data.Rooms)
	require.Equal(t, roomID, adminRoomsResp.Data.Rooms[0].ID)

	resp, err = admin.Client.Do(http.MethodGet, fmt.Sprintf("/api/admin/rooms/%d?op_page=1&op_page_size=10", roomID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var adminRoomDetailResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Room struct {
				ID           uint `json:"id"`
				TableBalance int  `json:"table_balance"`
			} `json:"room"`
			TableBalance int `json:"table_balance"`
			Members      []struct {
				UserID uint   `json:"user_id"`
				Status string `json:"status"`
			} `json:"members"`
			Operations struct {
				List  []operationView `json:"list"`
				Total int64           `json:"total"`
			} `json:"operations"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &adminRoomDetailResp)
	require.Equal(t, 0, adminRoomDetailResp.Code)
	require.Equal(t, roomID, adminRoomDetailResp.Data.Room.ID)
	require.Equal(t, 0, adminRoomDetailResp.Data.TableBalance)
	require.NotEmpty(t, adminRoomDetailResp.Data.Members)
	require.NotEmpty(t, adminRoomDetailResp.Data.Operations.List)

	resp, err = admin.Client.Do(http.MethodGet, fmt.Sprintf("/api/admin/users/%d/settlements", member.UserID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var adminSettlementsResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			User struct {
				ID uint `json:"id"`
			} `json:"user"`
			Settlements []struct {
				RoomID     uint    `json:"room_id"`
				ChipAmount int     `json:"chip_amount"`
				RmbAmount  float64 `json:"rmb_amount"`
			} `json:"settlements"`
			TotalChip int     `json:"total_chip"`
			TotalRmb  float64 `json:"total_rmb"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &adminSettlementsResp)
	require.Equal(t, 0, adminSettlementsResp.Code)
	require.Equal(t, member.UserID, adminSettlementsResp.Data.User.ID)
	require.NotEmpty(t, adminSettlementsResp.Data.Settlements)

	resp, err = admin.Client.Do(http.MethodGet, "/api/admin/room-member-history?page=1&page_size=20", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code)

	var adminHistoryResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Records []struct {
				ID            uint   `json:"id"`
				RoomID        uint   `json:"room_id"`
				OperationType string `json:"operation_type"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &adminHistoryResp)
	require.Equal(t, 0, adminHistoryResp.Code)
	require.NotEmpty(t, adminHistoryResp.Data.Records)
}
