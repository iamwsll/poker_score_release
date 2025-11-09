package controllers

import (
	"poker_score_backend/services"
	"poker_score_backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// OperationController 房间操作控制器
type OperationController struct {
	operationService *services.OperationService
}

// NewOperationController 创建房间操作控制器
func NewOperationController(operationService *services.OperationService) *OperationController {
	return &OperationController{
		operationService: operationService,
	}
}

// BetRequest 下注请求
type BetRequest struct {
	Amount int `json:"amount" binding:"required,gt=0"`
}

// Bet 下注/支出
func (ctrl *OperationController) Bet(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	var req BetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 下注
	myBalance, tableBalance, err := ctrl.operationService.Bet(uint(roomID), userID.(uint), req.Amount)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "下注成功", gin.H{
		"my_balance":    myBalance,
		"table_balance": tableBalance,
	})
}

// WithdrawRequest 收回请求
type WithdrawRequest struct {
	Amount int `json:"amount"` // 0或负数表示全收
}

// ForceTransferRequest 积分强制转移请求
type ForceTransferRequest struct {
	TargetUserID uint `json:"target_user_id" binding:"required"`
}

// Withdraw 收回
func (ctrl *OperationController) Withdraw(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 收回
	myBalance, tableBalance, actualAmount, err := ctrl.operationService.Withdraw(uint(roomID), userID.(uint), req.Amount)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "收回成功", gin.H{
		"my_balance":    myBalance,
		"table_balance": tableBalance,
		"actual_amount": actualAmount,
	})
}

// ForceTransfer 积分强制转移
func (ctrl *OperationController) ForceTransfer(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	var req ForceTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID, _ := c.Get("user_id")

	actorBalance, targetBalance, tableBalance, amount, err := ctrl.operationService.ForceTransfer(uint(roomID), userID.(uint), req.TargetUserID)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	response := gin.H{
		"table_balance":      tableBalance,
		"transferred_amount": amount,
		"target_user_id":     req.TargetUserID,
		"target_balance":     targetBalance,
		"actor_user_id":      userID.(uint),
		"actor_balance":      actorBalance,
	}

	utils.SuccessWithMessage(c, "积分已转移", response)
}

// NiuniuBetRequest 牛牛下注请求
type NiuniuBetRequest struct {
	Bets []services.NiuniuBetItem `json:"bets" binding:"required"`
}

// NiuniuBet 牛牛下注
func (ctrl *OperationController) NiuniuBet(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	var req NiuniuBetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if len(req.Bets) == 0 {
		utils.BadRequest(c, "至少选择一个下注对象")
		return
	}

	for _, bet := range req.Bets {
		if bet.ToUserID == 0 {
			utils.BadRequest(c, "下注对象无效")
			return
		}
		if bet.Amount <= 0 {
			utils.BadRequest(c, "下注金额必须大于0")
			return
		}
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 牛牛下注
	myBalance, totalAmount, err := ctrl.operationService.NiuniuBet(uint(roomID), userID.(uint), req.Bets)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "下注成功", gin.H{
		"my_balance":   myBalance,
		"total_amount": totalAmount,
	})
}

// GetOperations 获取操作历史
func (ctrl *OperationController) GetOperations(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 获取分页参数
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 获取操作历史
	operations, total, err := ctrl.operationService.GetOperations(uint(roomID), userID.(uint), limit, offset)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"operations": operations,
		"total":      total,
	})
}

// GetHistoryAmounts 获取用户历史操作金额
func (ctrl *OperationController) GetHistoryAmounts(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 获取历史金额
	betAmounts, withdrawAmounts, err := ctrl.operationService.GetHistoryAmounts(uint(roomID), userID.(uint))
	if err != nil {
		utils.InternalServerError(c, "获取历史金额失败")
		return
	}

	utils.Success(c, gin.H{
		"bet_amounts":      betAmounts,
		"withdraw_amounts": withdrawAmounts,
	})
}
