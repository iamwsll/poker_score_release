import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { Capacitor } from '@capacitor/core'
import { message } from 'ant-design-vue'
import router from '@/router'

const DEFAULT_API_PATH = '/api'
const rawApiBaseUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim()
const rawNativeApiBaseUrl = (import.meta.env.VITE_NATIVE_API_BASE_URL as string | undefined)?.trim()

function isNativeRuntime(): boolean {
  if (typeof Capacitor === 'undefined') {
    return false
  }
  if (typeof Capacitor.isNativePlatform === 'function') {
    return Capacitor.isNativePlatform()
  }
  return Capacitor.getPlatform() !== 'web'
}

function resolveApiBaseUrl(): string {
  const nativeRuntime = isNativeRuntime()

  if (nativeRuntime) {
    if (rawNativeApiBaseUrl && rawNativeApiBaseUrl.length > 0) {
      return rawNativeApiBaseUrl
    }

    if (rawApiBaseUrl && isAbsoluteHttpUrl(rawApiBaseUrl)) {
      return rawApiBaseUrl
    }

    console.warn(
      '[request] 检测到原生运行环境，但未设置 `VITE_NATIVE_API_BASE_URL`，已回退到默认值 `/api`，请在构建前设置完整的后端地址。'
    )
  }

  if (rawApiBaseUrl && rawApiBaseUrl.length > 0) {
    return rawApiBaseUrl
  }

  return DEFAULT_API_PATH
}

function isAbsoluteHttpUrl(url: string): boolean {
  return url.startsWith('http://') || url.startsWith('https://')
}

const apiBaseUrl = resolveApiBaseUrl()

function normalizeBaseUrl(url: string): string {
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }

  return url.startsWith('/') ? url : `/${url}`
}

// 创建axios实例
const service: AxiosInstance = axios.create({
  baseURL: normalizeBaseUrl(apiBaseUrl),
  timeout: 15000,
  withCredentials: true // 携带cookie
})

// 请求拦截器
service.interceptors.request.use(
  (config) => {
    // 从localStorage获取session_id，添加到Authorization header
    // 这对移动端浏览器更可靠，因为某些浏览器不支持Cookie
    const sessionId = localStorage.getItem('session_id')
    if (sessionId) {
      config.headers['Authorization'] = `Bearer ${sessionId}`
    }
    return config
  },
  (error) => {
    console.error('请求错误:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    const res = response.data

    // 如果返回的状态码不是0，说明有错误
    if (res.code !== 0) {
      message.error(res.message || '请求失败')

      // 401: 未登录或Session过期
      if (res.code === 401) {
        // 清除本地存储的session_id
        localStorage.removeItem('session_id')
        router.push('/login')
      }

      return Promise.reject(new Error(res.message || '请求失败'))
    } else {
      return res
    }
  },
  (error) => {
    console.error('响应错误:', error)

    if (error.response) {
      const status = error.response.status
      if (status === 401) {
        message.error('未登录或Session已过期，请重新登录')
        // 清除本地存储的session_id
        localStorage.removeItem('session_id')
        router.push('/login')
      } else if (status === 403) {
        message.error('权限不足')
      } else if (status === 404) {
        message.error('请求的资源不存在')
      } else if (status === 500) {
        message.error('服务器错误')
      } else {
        message.error(error.response.data?.message || '网络错误')
      }
    } else {
      message.error('网络连接失败，请检查网络')
    }

    return Promise.reject(error)
  }
)

// 封装请求方法
export function request<T = any>(config: AxiosRequestConfig): Promise<T> {
  return service.request(config)
}

export function get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
  return service.get(url, config)
}

export function post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
  return service.post(url, data, config)
}

export function put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
  return service.put(url, data, config)
}

export function del<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
  return service.delete(url, config)
}

export default service

