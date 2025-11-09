import { post, get, put } from './request'

export interface LoginParams {
  phone: string
  password: string
}

export interface RegisterParams {
  phone: string
  nickname: string
  password: string
}

export interface User {
  id: number
  phone: string
  nickname: string
  role: string
  created_at: string
}

// 注册
export function register(data: RegisterParams) {
  return post('/auth/register', data)
}

// 登录
export function login(data: LoginParams) {
  return post('/auth/login', data)
}

// 登出
export function logout() {
  return post('/auth/logout')
}

// 获取当前用户信息
export function getMe() {
  return get<{ data: User }>('/auth/me')
}

// 修改昵称
export function updateNickname(nickname: string) {
  return put('/auth/nickname', { nickname })
}

// 修改密码
export function updatePassword(oldPassword: string, newPassword: string) {
  return put('/auth/password', { old_password: oldPassword, new_password: newPassword })
}

