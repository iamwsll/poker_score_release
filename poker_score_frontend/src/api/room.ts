import { post, get } from './request'

// 创建房间
export function createRoom(roomType: 'texas' | 'niuniu', chipRate: string) {
  return post('/rooms', { room_type: roomType, chip_rate: chipRate })
}

// 加入房间
export function joinRoom(roomCode: string) {
  return post('/rooms/join', { room_code: roomCode })
}

// 返回上次房间
export function getLastRoom() {
  return get('/rooms/last')
}

// 获取房间详情
export function getRoomDetails(roomId: number) {
  return get(`/rooms/${roomId}`)
}

// 返回房间
export function returnToRoom(roomId: number) {
  return post(`/rooms/${roomId}/return`)
}

// 离开房间
export function leaveRoom(roomId: number) {
  return post(`/rooms/${roomId}/leave`)
}

// 踢出用户
export function kickUser(roomId: number, userId: number) {
  return post(`/rooms/${roomId}/kick`, { user_id: userId })
}

// 下注
export function bet(roomId: number, amount: number) {
  return post(`/rooms/${roomId}/bet`, { amount })
}

// 收回
export function withdraw(roomId: number, amount: number) {
  return post(`/rooms/${roomId}/withdraw`, { amount })
}

// 牛牛下注
export function niuniuBet(roomId: number, bets: Array<{ to_user_id: number; amount: number }>) {
  return post(`/rooms/${roomId}/niuniu-bet`, { bets })
}

// 获取操作历史
export function getOperations(roomId: number, limit = 50, offset = 0) {
  return get(`/rooms/${roomId}/operations`, { params: { limit, offset } })
}

// 获取历史金额
export function getHistoryAmounts(roomId: number) {
  return get(`/rooms/${roomId}/history-amounts`)
}

// 发起结算
export function initiateSettlement(roomId: number) {
  return post(`/rooms/${roomId}/settlement/initiate`)
}

// 确认结算
export function confirmSettlement(roomId: number) {
  return post(`/rooms/${roomId}/settlement/confirm`)
}

