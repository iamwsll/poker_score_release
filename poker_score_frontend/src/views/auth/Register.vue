<template>
  <div class="register-container">
    <div class="register-card">
      <h1 class="title">注册账户</h1>
      <a-form
        :model="formState"
        :rules="rules"
        @finish="handleRegister"
        layout="vertical"
      >
        <a-form-item label="手机号" name="phone">
          <a-input
            v-model:value="formState.phone"
            placeholder="请输入手机号"
            size="large"
            :maxlength="11"
          />
        </a-form-item>

        <a-form-item label="昵称" name="nickname">
          <a-input
            v-model:value="formState.nickname"
            placeholder="请输入昵称"
            size="large"
            :maxlength="50"
          />
        </a-form-item>

        <a-form-item label="密码" name="password">
          <a-input-password
            v-model:value="formState.password"
            placeholder="请输入密码（至少6位）"
            size="large"
          />
        </a-form-item>

        <a-form-item label="确认密码" name="confirmPassword">
          <a-input-password
            v-model:value="formState.confirmPassword"
            placeholder="请再次输入密码"
            size="large"
          />
        </a-form-item>

        <a-form-item>
          <a-button
            type="primary"
            html-type="submit"
            block
            size="large"
            :loading="loading"
          >
            注册并登录
          </a-button>
        </a-form-item>
      </a-form>

      <div class="login-tip">
        <span>已有账户？</span>
        <router-link to="/login" class="login-link">去登录</router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const loading = ref(false)

const formState = reactive({
  phone: '',
  nickname: '',
  password: '',
  confirmPassword: ''
})

const rules = {
  phone: [
    { required: true, message: '请输入手机号' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号' }
  ],
  nickname: [
    { required: true, message: '请输入昵称' },
    { min: 1, max: 50, message: '昵称长度为1-50个字符' }
  ],
  password: [
    { required: true, message: '请输入密码' },
    { min: 6, message: '密码至少6位' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码' },
    {
      validator: (_rule: any, value: string) => {
        if (value !== formState.password) {
          return Promise.reject('两次输入的密码不一致')
        }
        return Promise.resolve()
      }
    }
  ]
}

const handleRegister = async () => {
  loading.value = true
  try {
    await userStore.register(formState.phone, formState.nickname, formState.password)
    message.success('注册成功，已自动登录')
    router.push('/')
  } catch (error) {
    // 错误已在API层处理
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.register-card {
  width: 100%;
  max-width: 400px;
  background: white;
  border-radius: 16px;
  padding: 40px 30px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
}

.title {
  text-align: center;
  font-size: 28px;
  font-weight: bold;
  color: #333;
  margin-bottom: 40px;
}

.login-tip {
  text-align: center;
  margin-top: 20px;
  color: #666;
  font-size: 14px;
}

.login-link {
  color: #667eea;
  margin-left: 8px;
  font-weight: 500;
}

.login-link:hover {
  color: #764ba2;
}
</style>

