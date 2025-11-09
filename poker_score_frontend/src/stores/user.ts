import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as authApi from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  // 状态
  const user = ref<authApi.User | null>(null)
  const isLoggedIn = ref(false)

  // 获取用户信息
  async function fetchUserInfo() {
    try {
      const res = await authApi.getMe()
      user.value = res.data
      isLoggedIn.value = true
      return res.data
    } catch (error) {
      user.value = null
      isLoggedIn.value = false
      throw error
    }
  }

  // 登录
  async function login(phone: string, password: string) {
    const res = await authApi.login({ phone, password })
    user.value = res.data.user
    isLoggedIn.value = true

    // 保存session_id到localStorage（用于移动端浏览器）
    if (res.data.session_id) {
      localStorage.setItem('session_id', res.data.session_id)
    }

    return res
  }

  // 注册
  async function register(phone: string, nickname: string, password: string) {
    const res = await authApi.register({ phone, nickname, password })
    user.value = res.data.user
    isLoggedIn.value = true

    // 保存session_id到localStorage（用于移动端浏览器）
    if (res.data.session_id) {
      localStorage.setItem('session_id', res.data.session_id)
    }

    return res
  }

  // 登出
  async function logout() {
    await authApi.logout()
    user.value = null
    isLoggedIn.value = false
    // 清除本地存储的session_id
    localStorage.removeItem('session_id')
  }

  // 修改昵称
  async function updateNickname(nickname: string) {
    await authApi.updateNickname(nickname)
    if (user.value) {
      user.value.nickname = nickname
    }
  }

  // 修改密码
  async function updatePassword(oldPassword: string, newPassword: string) {
    await authApi.updatePassword(oldPassword, newPassword)
  }

  return {
    user,
    isLoggedIn,
    fetchUserInfo,
    login,
    register,
    logout,
    updateNickname,
    updatePassword,
  }
})

