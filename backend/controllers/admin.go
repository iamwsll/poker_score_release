package controllers

import (
	"poker_score_backend/services"
	"poker_score_backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminController 后台管理控制器
type AdminController struct {
	adminService *services.AdminService
}

// NewAdminController 创建后台管理控制器
func NewAdminController(adminService *services.AdminService) *AdminController {
	return &AdminController{
		adminService: adminService,
	}
}

// GetUsers 获取用户列表
func (ctrl *AdminController) GetUsers(c *gin.Context) {
	// 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	// 查询用户列表
	users, total, err := ctrl.adminService.GetUsers(page, pageSize)
	if err != nil {
		utils.InternalServerError(c, "查询用户列表失败")
		return
	}

	utils.Success(c, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetRooms 获取房间列表
func (ctrl *AdminController) GetRooms(c *gin.Context) {
	// 获取参数
	status := c.DefaultQuery("status", "all")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	// 查询房间列表
	rooms, total, err := ctrl.adminService.GetRooms(status, page, pageSize)
	if err != nil {
		utils.InternalServerError(c, "查询房间列表失败")
		return
	}

	utils.Success(c, gin.H{
		"rooms":     rooms,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetRoomDetails 获取房间详细信息
func (ctrl *AdminController) GetRoomDetails(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 查询房间详情
	details, err := ctrl.adminService.GetRoomDetails(uint(roomID))
	if err != nil {
		utils.NotFound(c, "房间不存在")
		return
	}

	utils.Success(c, details)
}

// GetUserSettlements 获取用户历史盈亏
func (ctrl *AdminController) GetUserSettlements(c *gin.Context) {
	// 获取用户ID
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "用户ID格式错误")
		return
	}

	// 获取时间参数
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime *time.Time
	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err == nil {
			startTime = &t
		}
	}
	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err == nil {
			endTime = &t
		}
	}

	// 查询用户盈亏
	settlements, err := ctrl.adminService.GetUserSettlements(uint(userID), startTime, endTime)
	if err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	utils.Success(c, settlements)
}

// GetRoomMemberHistory 获取用户进出房间历史
func (ctrl *AdminController) GetRoomMemberHistory(c *gin.Context) {
	// 获取参数
	userIDStr := c.Query("user_id")
	roomIDStr := c.Query("room_id")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "50")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	var userID, roomID *uint
	if userIDStr != "" {
		id, err := strconv.ParseUint(userIDStr, 10, 32)
		if err == nil {
			uid := uint(id)
			userID = &uid
		}
	}
	if roomIDStr != "" {
		id, err := strconv.ParseUint(roomIDStr, 10, 32)
		if err == nil {
			rid := uint(id)
			roomID = &rid
		}
	}

	// 查询历史记录
	records, total, err := ctrl.adminService.GetRoomMemberHistory(userID, roomID, page, pageSize)
	if err != nil {
		utils.InternalServerError(c, "查询历史记录失败")
		return
	}

	utils.Success(c, gin.H{
		"records": records,
		"total":   total,
	})
}

