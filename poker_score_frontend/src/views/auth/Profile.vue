<template>
  <div class="profile-container">
    <div class="header">
      <a-button @click="router.push('/')" type="text">
        <template #icon><LeftOutlined /></template>
        返回
      </a-button>
      <h2>个人信息</h2>
      <div style="width: 60px"></div>
    </div>

    <div class="profile-content">
      <div class="user-info">
        <div class="info-item">
          <span class="label">手机号</span>
          <span class="value">{{ userStore.user?.phone }}</span>
        </div>
        <div class="info-item">
          <span class="label">昵称</span>
          <span class="value">{{ userStore.user?.nickname }}</span>
        </div>
        <div class="info-item">
          <span class="label">角色</span>
          <span class="value">{{ userStore.user?.role === 'admin' ? '管理员' : '普通用户' }}</span>
        </div>
      </div>

      <div class="action-buttons">
        <a-button block size="large" @click="showNicknameModal = true">
          修改昵称
        </a-button>
        <a-button block size="large" @click="showPasswordModal = true" style="margin-top: 16px">
          修改密码
        </a-button>
        <a-button block size="large" danger @click="handleLogout" style="margin-top: 16px">
          登出账户
        </a-button>
      </div>
    </div>

    <!-- 修改昵称弹窗 -->
    <a-modal
      v-model:open="showNicknameModal"
      title="修改昵称"
      @ok="handleUpdateNickname"
      :confirmLoading="nicknameLoading"
    >
      <a-form layout="vertical">
        <a-form-item label="新昵称">
          <a-input
            v-model:value="newNickname"
            placeholder="请输入新昵称"
            :maxlength="50"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 修改密码弹窗 -->
    <a-modal
      v-model:open="showPasswordModal"
      title="修改密码"
      @ok="handleUpdatePassword"
      :confirmLoading="passwordLoading"
    >
      <a-form layout="vertical">
        <a-form-item label="旧密码">
          <a-input-password
            v-model:value="passwordForm.oldPassword"
            placeholder="请输入旧密码"
          />
        </a-form-item>
        <a-form-item label="新密码">
          <a-input-password
            v-model:value="passwordForm.newPassword"
            placeholder="请输入新密码（至少6位）"
          />
        </a-form-item>
        <a-form-item label="确认新密码">
          <a-input-password
            v-model:value="passwordForm.confirmPassword"
            placeholder="请再次输入新密码"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { LeftOutlined } from '@ant-design/icons-vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const showNicknameModal = ref(false)
const showPasswordModal = ref(false)
const nicknameLoading = ref(false)
const passwordLoading = ref(false)

const newNickname = ref(userStore.user?.nickname || '')

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

// 修改昵称
const handleUpdateNickname = async () => {
  if (!newNickname.value.trim()) {
    message.warning('请输入新昵称')
    return
  }

  nicknameLoading.value = true
  try {
    await userStore.updateNickname(newNickname.value)
    message.success('昵称修改成功')
    showNicknameModal.value = false
  } catch (error) {
    // 错误已在API层处理
  } finally {
    nicknameLoading.value = false
  }
}

// 修改密码
const handleUpdatePassword = async () => {
  if (!passwordForm.oldPassword) {
    message.warning('请输入旧密码')
    return
  }
  if (!passwordForm.newPassword || passwordForm.newPassword.length < 6) {
    message.warning('新密码至少6位')
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    message.warning('两次输入的新密码不一致')
    return
  }

  passwordLoading.value = true
  try {
    await userStore.updatePassword(passwordForm.oldPassword, passwordForm.newPassword)
    message.success('密码修改成功')
    showPasswordModal.value = false
    // 清空表单
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
  } catch (error) {
    // 错误已在API层处理
  } finally {
    passwordLoading.value = false
  }
}

// 登出
const handleLogout = () => {
  Modal.confirm({
    title: '确认登出',
    content: '您确定要登出账户吗？',
    onOk: async () => {
      await userStore.logout()
      message.success('已登出')
      router.push('/login')
    }
  })
}
</script>

<style scoped>
.profile-container {
  min-height: 100vh;
  background: #f5f5f5;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.profile-content {
  padding: 20px;
}

.user-info {
  background: white;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.info-item:last-child {
  border-bottom: none;
}

.label {
  color: #666;
  font-size: 14px;
}

.value {
  color: #333;
  font-size: 15px;
  font-weight: 500;
}

.action-buttons {
  background: white;
  border-radius: 12px;
  padding: 20px;
}
</style>

