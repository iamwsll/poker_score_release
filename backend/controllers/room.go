package controllers

import (
	"poker_score_backend/services"
	"poker_score_backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RoomController 房间控制器
type RoomController struct {
	roomService *services.RoomService
}

// NewRoomController 创建房间控制器
func NewRoomController(roomService *services.RoomService) *RoomController {
	return &RoomController{
		roomService: roomService,
	}
}

// CreateRoomRequest 创建房间请求
type CreateRoomRequest struct {
	RoomType string `json:"room_type" binding:"required,oneof=texas niuniu"`
	ChipRate string `json:"chip_rate" binding:"required"`
}

// CreateRoom 创建房间
func (ctrl *RoomController) CreateRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 调用服务层创建房间
	room, err := ctrl.roomService.CreateRoom(userID.(uint), req.RoomType, req.ChipRate)
	if err != nil {
		utils.InternalServerError(c, "创建房间失败")
		return
	}

	utils.SuccessWithMessage(c, "房间创建成功", gin.H{
		"room_id":    room.ID,
		"room_code":  room.RoomCode,
		"room_type":  room.RoomType,
		"chip_rate":  room.ChipRate,
		"created_at": room.CreatedAt,
	})
}

// JoinRoomRequest 加入房间请求
type JoinRoomRequest struct {
	RoomCode string `json:"room_code" binding:"required,len=6"`
}

// JoinRoom 加入房间
func (ctrl *RoomController) JoinRoom(c *gin.Context) {
	var req JoinRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 调用服务层加入房间
	room, _, err := ctrl.roomService.JoinRoomByCode(userID.(uint), req.RoomCode)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 获取房间详情
	details, err := ctrl.roomService.GetRoomDetails(room.ID, userID.(uint))
	if err != nil {
		utils.InternalServerError(c, "获取房间详情失败")
		return
	}

	utils.SuccessWithMessage(c, "加入房间成功", details)
}

// GetLastRoom 返回上次房间
func (ctrl *RoomController) GetLastRoom(c *gin.Context) {
	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 获取最后的房间
	room, err := ctrl.roomService.GetLastRoom(userID.(uint))
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	// 获取房间详情
	details, err := ctrl.roomService.GetRoomDetails(room.ID, userID.(uint))
	if err != nil {
		utils.InternalServerError(c, "获取房间详情失败")
		return
	}

	utils.Success(c, details)
}

// GetRoomDetails 获取房间详情
func (ctrl *RoomController) GetRoomDetails(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 获取房间详情
	details, err := ctrl.roomService.GetRoomDetails(uint(roomID), userID.(uint))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, details)
}

// ReturnToRoom 返回房间
func (ctrl *RoomController) ReturnToRoom(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	userID, _ := c.Get("user_id")

	operation, err := ctrl.roomService.ReturnToRoom(uint(roomID), userID.(uint))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var data interface{}
	if operation != nil {
		data = gin.H{
			"operation": operation,
		}
	}

	utils.SuccessWithMessage(c, "返回房间成功", data)
}

// LeaveRoom 离开房间
func (ctrl *RoomController) LeaveRoom(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 离开房间
	err = ctrl.roomService.LeaveRoom(userID.(uint), uint(roomID))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "离开房间成功", nil)
}

// KickUserRequest 踢出用户请求
type KickUserRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// KickUser 踢出用户
func (ctrl *RoomController) KickUser(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	var req KickUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 踢出用户
	err = ctrl.roomService.KickUser(uint(roomID), userID.(uint), req.UserID)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "踢出成功", nil)
}
