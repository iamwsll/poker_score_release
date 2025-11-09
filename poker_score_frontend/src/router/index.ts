import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { message } from 'ant-design-vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/Home.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/Login.vue'),
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/auth/Register.vue'),
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/auth/Profile.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/room/:id',
      name: 'room',
      component: () => import('@/views/room/Room.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/tonight-record',
      name: 'tonight-record',
      component: () => import('@/views/TonightRecord.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('@/views/Admin.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
  ],
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()

  // 如果需要认证
  if (to.meta.requiresAuth) {
    // 如果未登录，尝试获取用户信息
    if (!userStore.isLoggedIn) {
      try {
        await userStore.fetchUserInfo()
      } catch (error) {
        message.warning('请先登录')
        next({ name: 'login', query: { redirect: to.fullPath } })
        return
      }
    }

    // 如果需要管理员权限
    if (to.meta.requiresAdmin && userStore.user?.role !== 'admin') {
      message.error('权限不足')
      next({ name: 'home' })
      return
    }
  }

  next()
})

export default router
