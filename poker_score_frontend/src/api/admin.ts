import { get, put } from './request'

// 获取用户列表
export function getUsers(page = 1, pageSize = 20) {
  return get('/admin/users', { params: { page, page_size: pageSize } })
}

// 更新用户信息
export function updateUser(userId: number, payload: { phone: string; nickname: string; role: string; password?: string }) {
  return put(`/admin/users/${userId}`, payload)
}

// 获取房间列表
export function getRooms(status = 'all', page = 1, pageSize = 20) {
  return get('/admin/rooms', { params: { status, page, page_size: pageSize } })
}

// 获取房间详情
export function getRoomDetails(roomId: number) {
  return get(`/admin/rooms/${roomId}`)
}

// 获取用户历史盈亏
export function getUserSettlements(userId: number, startTime?: string, endTime?: string) {
  const params: any = {}
  if (startTime) params.start_time = startTime
  if (endTime) params.end_time = endTime
  return get(`/admin/users/${userId}/settlements`, { params })
}

// 获取用户进出房间历史
export function getRoomMemberHistory(userId?: number, roomId?: number, page = 1, pageSize = 50) {
  const params: any = { page, page_size: pageSize }
  if (userId) params.user_id = userId
  if (roomId) params.room_id = roomId
  return get('/admin/room-member-history', { params })
}

