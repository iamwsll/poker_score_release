package services

import (
	"fmt"
	"testing"
	"time"

	"poker_score_backend/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupSettlementTestDB(t *testing.T) {
	t.Helper()

	dbPath := fmt.Sprintf("file:settlement-test-%s.db?mode=memory&cache=shared&_fk=1", uuid.NewString())
	require.NoError(t, models.InitDatabase(dbPath, 1, 1, time.Minute))

	t.Cleanup(func() {
		require.NoError(t, models.CloseDatabase())
	})
}

func seedUsers(t *testing.T, nicknames []string) []models.User {
	t.Helper()

	users := make([]models.User, len(nicknames))
	for i, nickname := range nicknames {
		user := models.User{
			Phone:        fmt.Sprintf("139%08d", i+1),
			Nickname:     nickname,
			PasswordHash: "hash",
			Role:         "user",
		}
		require.NoError(t, models.DB.Create(&user).Error)
		users[i] = user
	}
	return users
}

func TestCalculateRmbAmount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		chipAmount int
		chipRate   string
		want       float64
	}{
		{
			name:       "standard rate",
			chipAmount: 200,
			chipRate:   "20:1",
			want:       10,
		},
		{
			name:       "decimal rate",
			chipAmount: 150,
			chipRate:   "30:2",
			want:       10,
		},
		{
			name:       "negative chip amount preserves sign",
			chipAmount: -200,
			chipRate:   "20:1",
			want:       -10,
		},
		{
			name:       "invalid format",
			chipAmount: 100,
			chipRate:   "201",
			want:       0,
		},
		{
			name:       "non numeric",
			chipAmount: 100,
			chipRate:   "a:b",
			want:       0,
		},
		{
			name:       "zero chip part",
			chipAmount: 100,
			chipRate:   "0:1",
			want:       0,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := calculateRmbAmount(tc.chipAmount, tc.chipRate)
			require.InDelta(t, tc.want, got, 1e-9)
		})
	}
}

func TestSettlementServiceGenerateSettlementPlan(t *testing.T) {
	setupSettlementTestDB(t)

	users := seedUsers(t, []string{"Alice", "Bob", "Carol", "Dave"})

	balances := []models.UserBalance{
		{RoomID: 1, UserID: users[0].ID, Balance: 500},
		{RoomID: 1, UserID: users[1].ID, Balance: 200},
		{RoomID: 1, UserID: users[2].ID, Balance: -300},
		{RoomID: 1, UserID: users[3].ID, Balance: -200},
	}

	svc := &SettlementService{}
	plan := svc.generateSettlementPlan(balances, "20:1")

	require.Len(t, plan, 3)

	require.Equal(t, users[2].ID, plan[0].FromUserID)
	require.Equal(t, users[0].ID, plan[0].ToUserID)
	require.Equal(t, 300, plan[0].ChipAmount)
	require.InDelta(t, 15, plan[0].RmbAmount, 1e-9)
	require.Contains(t, plan[0].Description, "Alice")
	require.Contains(t, plan[0].Description, "Carol")

	require.Equal(t, users[3].ID, plan[1].FromUserID)
	require.Equal(t, users[0].ID, plan[1].ToUserID)
	require.Equal(t, 200, plan[1].ChipAmount)
	require.InDelta(t, 10, plan[1].RmbAmount, 1e-9)

	require.Equal(t, users[0].ID, plan[2].FromUserID)
	require.Equal(t, users[1].ID, plan[2].ToUserID)
	require.Equal(t, 200, plan[2].ChipAmount)
	require.InDelta(t, 10, plan[2].RmbAmount, 1e-9)
	require.Contains(t, plan[2].Description, "Alice")
	require.Contains(t, plan[2].Description, "Bob")
}

func TestSettlementServiceGenerateSettlementPlan_NoTransfersWhenSingleSide(t *testing.T) {
	setupSettlementTestDB(t)

	users := seedUsers(t, []string{"Positive", "Neutral"})
	svc := &SettlementService{}

	onlyPositive := []models.UserBalance{
		{RoomID: 2, UserID: users[0].ID, Balance: 150},
		{RoomID: 2, UserID: users[1].ID, Balance: 0},
	}
	require.Empty(t, svc.generateSettlementPlan(onlyPositive, "20:1"))

	onlyNegative := []models.UserBalance{
		{RoomID: 3, UserID: users[0].ID, Balance: 0},
		{RoomID: 3, UserID: users[1].ID, Balance: -50},
	}
	require.Empty(t, svc.generateSettlementPlan(onlyNegative, "20:1"))
}
