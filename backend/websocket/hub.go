package websocket

import (
	"log"
	"sync"
)

// Hub WebSocket连接管理中心
type Hub struct {
	// 房间ID -> 客户端集合的映射
	rooms map[uint]map[*Client]bool

	// 注册请求
	register chan *Client

	// 注销请求
	unregister chan *Client

	// 广播消息
	broadcast chan *BroadcastMessage

	// 互斥锁
	mu sync.RWMutex
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	RoomID  uint
	Message []byte
}

// NewHub 创建Hub
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[uint]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.RoomID] == nil {
				h.rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.rooms[client.RoomID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket客户端注册: RoomID=%d, UserID=%d", client.RoomID, client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.RoomID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					log.Printf("WebSocket客户端注销: RoomID=%d, UserID=%d", client.RoomID, client.UserID)

					// 如果房间没有客户端了，删除房间
					if len(clients) == 0 {
						delete(h.rooms, client.RoomID)
						log.Printf("WebSocket房间清空: RoomID=%d", client.RoomID)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.rooms[message.RoomID]; ok {
				for client := range clients {
					select {
					case client.send <- message.Message:
					default:
						// 发送失败，关闭客户端
						close(client.send)
						delete(clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToRoom 向房间广播消息
func (h *Hub) BroadcastToRoom(roomID uint, message []byte) {
	h.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: message,
	}
}

// GetRoomClientCount 获取房间在线客户端数量
func (h *Hub) GetRoomClientCount(roomID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.rooms[roomID]; ok {
		return len(clients)
	}
	return 0
}

