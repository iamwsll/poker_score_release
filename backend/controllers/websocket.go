package controllers

import (
	"log"
	"net/http"
	"poker_score_backend/models"
	"poker_score_backend/utils"
	ws "poker_score_backend/websocket"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有来源（生产环境应该限制）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketController WebSocket控制器
type WebSocketController struct {
	hub *ws.Hub
}

// NewWebSocketController 创建WebSocket控制器
func NewWebSocketController(hub *ws.Hub) *WebSocketController {
	return &WebSocketController{
		hub: hub,
	}
}

// HandleWebSocket 处理WebSocket连接
func (ctrl *WebSocketController) HandleWebSocket(c *gin.Context) {
	// 获取房间ID
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "房间ID格式错误")
		return
	}

	// 获取用户ID（从认证中间件）
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "未登录")
		return
	}

	// 检查用户是否在房间中
	var member models.RoomMember
	err = models.DB.Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).First(&member).Error
	if err != nil {
		utils.BadRequest(c, "您不在该房间中")
		return
	}

	// 升级为WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	// 创建WebSocket客户端并启动
	ws.ServeWs(ctrl.hub, conn, userID.(uint), uint(roomID))

	log.Printf("WebSocket连接建立: RoomID=%d, UserID=%d", roomID, userID)
}

// GetHub 获取Hub实例（供其他控制器使用）
func (ctrl *WebSocketController) GetHub() *ws.Hub {
	return ctrl.hub
}

