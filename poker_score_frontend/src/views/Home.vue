<template>
  <div class="home-container">
    <div class="header">
      <h1 class="title">比赛计分器</h1>
      <div class="user-info">
        <span>{{ userStore.user?.nickname }}</span>
      </div>
    </div>

    <div class="button-grid">
      <div class="button-item" @click="showCreateRoomModal = true">
        <PlusCircleOutlined class="icon" />
        <span>创建房间</span>
      </div>

      <div class="button-item" @click="showJoinRoomModal = true">
        <LoginOutlined class="icon" />
        <span>加入房间</span>
      </div>

      <div class="button-item" @click="handleReturnRoom">
        <RollbackOutlined class="icon" />
        <span>返回房间</span>
      </div>

      <div class="button-item" @click="router.push('/profile')">
        <UserOutlined class="icon" />
        <span>个人信息</span>
      </div>

      <div class="button-item" @click="router.push('/tonight-record')">
        <TrophyOutlined class="icon" />
        <span>今晚战绩</span>
      </div>

      <div v-if="userStore.user?.role === 'admin'" class="button-item" @click="router.push('/admin')">
        <SettingOutlined class="icon" />
        <span>后台管理</span>
      </div>
    </div>

    <!-- 创建房间弹窗 -->
    <a-modal
      v-model:open="showCreateRoomModal"
      title="创建房间"
      @ok="handleCreateRoom"
      :confirmLoading="createLoading"
    >
      <a-form layout="vertical">
        <a-form-item label="房间类型">
          <a-radio-group v-model:value="createForm.roomType" size="large">
            <a-radio-button value="texas">单计分房间</a-radio-button>
            <a-radio-button value="niuniu">多计分房间</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="积分比例">
          <a-input
            v-model:value="createForm.chipRate"
            placeholder="例如: 20:1"
            size="large"
          />
          <div style="margin-top: 8px; color: #999; font-size: 12px">
            {{ createForm.roomType === 'texas' ? '默认 20:1' : '默认 1:1' }}
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 加入房间弹窗 -->
    <a-modal
      v-model:open="showJoinRoomModal"
      title="加入房间"
      @ok="handleJoinRoom"
      :confirmLoading="joinLoading"
    >
      <a-form layout="vertical">
        <a-form-item label="房间号">
          <a-input
            v-model:value="roomCode"
            placeholder="请输入6位房间号"
            size="large"
            :maxlength="6"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  PlusCircleOutlined,
  LoginOutlined,
  RollbackOutlined,
  UserOutlined,
  TrophyOutlined,
  SettingOutlined
} from '@ant-design/icons-vue'
import { useUserStore } from '@/stores/user'
import * as roomApi from '@/api/room'

const router = useRouter()
const userStore = useUserStore()

const showCreateRoomModal = ref(false)
const showJoinRoomModal = ref(false)
const createLoading = ref(false)
const joinLoading = ref(false)
const roomCode = ref('')

const createForm = reactive({
  roomType: 'texas' as 'texas' | 'niuniu',
  chipRate: '20:1'
})

// 监听房间类型变化，自动设置默认比例
watch(() => createForm.roomType, (newType) => {
  createForm.chipRate = newType === 'texas' ? '20:1' : '1:1'
})

// 创建房间
const handleCreateRoom = async () => {
  if (!createForm.chipRate.match(/^\d+:\d+$/)) {
    message.warning('请输入正确的比例格式，例如: 20:1')
    return
  }

  createLoading.value = true
  try {
    const res = await roomApi.createRoom(createForm.roomType, createForm.chipRate)
    message.success(`房间创建成功，房间号: ${res.data.room_code}`)
    showCreateRoomModal.value = false
    // 跳转到房间页面
    router.push(`/room/${res.data.room_id}`)
  } catch (error) {
    // 错误已在API层处理
  } finally {
    createLoading.value = false
  }
}

// 加入房间
const handleJoinRoom = async () => {
  if (!roomCode.value || roomCode.value.length !== 6) {
    message.warning('请输入6位房间号')
    return
  }

  joinLoading.value = true
  try {
    const res = await roomApi.joinRoom(roomCode.value)
    message.success('加入房间成功')
    showJoinRoomModal.value = false
    roomCode.value = ''
    // 跳转到房间页面
    router.push(`/room/${res.data.room_id}`)
  } catch (error) {
    // 错误已在API层处理
  } finally {
    joinLoading.value = false
  }
}

// 返回上次房间
const handleReturnRoom = async () => {
  try {
    const res = await roomApi.getLastRoom()
    router.push(`/room/${res.data.room_id}`)
  } catch (error) {
    message.warning('没有可返回的房间')
  }
}
</script>

<style scoped>
.home-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.header {
  text-align: center;
  padding: 40px 0 60px;
}

.title {
  font-size: 36px;
  font-weight: bold;
  color: white;
  margin: 0 0 20px;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
}

.user-info {
  color: rgba(255, 255, 255, 0.9);
  font-size: 16px;
}

.button-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  max-width: 600px;
  margin: 0 auto;
}

.button-item {
  background: white;
  border-radius: 16px;
  padding: 40px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.button-item:active {
  transform: scale(0.95);
}

.button-item .icon {
  font-size: 48px;
  color: #667eea;
  margin-bottom: 12px;
}

.button-item span {
  font-size: 16px;
  font-weight: 500;
  color: #333;
}

@media (max-width: 480px) {
  .button-grid {
    gap: 12px;
  }

  .button-item {
    padding: 30px 15px;
  }

  .button-item .icon {
    font-size: 40px;
  }

  .button-item span {
    font-size: 14px;
  }
}
</style>

