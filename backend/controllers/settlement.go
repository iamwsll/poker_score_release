package controllers

import (
	"poker_score_backend/services"
	"poker_score_backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SettlementController 结算控制器
type SettlementController struct {
	settlementService *services.SettlementService
}

// NewSettlementController 创建结算控制器
func NewSettlementController(settlementService *services.SettlementService) *SettlementController {
	return &SettlementController{
		settlementService: settlementService,
	}
}

// InitiateSettlement 发起结算
func (ctrl *SettlementController) InitiateSettlement(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 发起结算
	canSettle, tableBalance, plan, err := ctrl.settlementService.InitiateSettlement(uint(roomID), userID.(uint))
	if err != nil {
		utils.ErrorWithData(c, 400, 400, err.Error(), gin.H{
			"table_balance": tableBalance,
		})
		return
	}

	utils.SuccessWithMessage(c, "结算方案已生成", gin.H{
		"can_settle":      canSettle,
		"table_balance":   tableBalance,
		"settlement_plan": plan,
	})
}

// ConfirmSettlement 确认结算
func (ctrl *SettlementController) ConfirmSettlement(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 确认结算
	settlementBatch, settledAt, err := ctrl.settlementService.ConfirmSettlement(uint(roomID), userID.(uint))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "结算完成", gin.H{
		"settlement_batch": settlementBatch,
		"settled_at":       settledAt,
	})
}

