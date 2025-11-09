package websocket

import (
	"encoding/json"
	"log"
	"poker_score_backend/models"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 写入等待时间
	writeWait = 10 * time.Second

	// 读取超时时间
	pongWait = 60 * time.Second

	// Ping周期（必须小于pongWait）
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512
)

// Client WebSocket客户端
type Client struct {
	hub *Hub

	// WebSocket连接
	conn *websocket.Conn

	// 发送消息的通道
	send chan []byte

	// 用户ID
	UserID uint

	// 房间ID
	RoomID uint
}

// Message WebSocket消息
type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// readPump 从WebSocket读取消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()

		// 将用户标记为离线状态，但保留房间成员身份
		res := models.DB.Model(&models.RoomMember{}).
			Where("room_id = ? AND user_id = ? AND left_at IS NULL", c.RoomID, c.UserID).
			Update("status", "offline")

		if res.Error != nil {
			log.Printf("用户离线标记失败: RoomID=%d, UserID=%d, %v", c.RoomID, c.UserID, res.Error)
		}

		log.Printf("用户WebSocket连接断开，状态已设为离线: RoomID=%d, UserID=%d", c.RoomID, c.UserID)
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		// 处理客户端消息（目前只处理ping）
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg.Type == "ping" {
				// 回复pong
				pongMsg := Message{Type: "pong"}
				pongBytes, _ := json.Marshal(pongMsg)
				c.send <- pongBytes
			}
		}
	}
}

// writePump 向WebSocket写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub关闭了通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 将排队的消息一起发送
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs 处理WebSocket请求
func ServeWs(hub *Hub, conn *websocket.Conn, userID, roomID uint) {
	// 确保成员状态为在线
	if err := models.DB.Model(&models.RoomMember{}).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		Update("status", "online").Error; err != nil {
		log.Printf("更新成员在线状态失败: RoomID=%d, UserID=%d, %v", roomID, userID, err)
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		UserID: userID,
		RoomID: roomID,
	}

	client.hub.register <- client

	// 在新的goroutine中启动读写
	go client.writePump()
	go client.readPump()
}
