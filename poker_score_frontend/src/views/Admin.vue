<template>
  <div class="admin-container">
    <div class="header">
      <a-button @click="router.push('/')" type="text">
        <template #icon><LeftOutlined /></template>
        返回
      </a-button>
      <h2>后台管理</h2>
      <div style="width: 60px"></div>
    </div>

    <div class="content">
      <a-tabs v-model:activeKey="activeTab">
        <!-- 用户管理 -->
        <a-tab-pane key="users" tab="用户管理">
          <div class="tab-content">
            <a-button @click="loadUsers" :loading="usersLoading" style="margin-bottom: 16px">
              刷新
            </a-button>
            <div class="table-container">
              <div class="table-row header-row">
                <div class="cell">ID</div>
                <div class="cell">昵称</div>
                <div class="cell">手机号</div>
                <div class="cell">角色</div>
              </div>
              <div
                v-for="user in users"
                :key="user.id"
                class="table-row"
              >
                <div class="cell">{{ user.id }}</div>
                <div class="cell">{{ user.nickname }}</div>
                <div class="cell">{{ user.phone }}</div>
                <div class="cell">
                  <a-tag :color="user.role === 'admin' ? 'red' : 'blue'">
                    {{ user.role === 'admin' ? '管理员' : '普通用户' }}
                  </a-tag>
                </div>
              </div>
            </div>
            <a-pagination
              v-model:current="usersPage"
              v-model:page-size="usersPageSize"
              :total="usersTotal"
              @change="loadUsers"
              style="margin-top: 16px; text-align: center"
            />
          </div>
        </a-tab-pane>

        <!-- 房间管理 -->
        <a-tab-pane key="rooms" tab="房间管理">
          <div class="tab-content">
            <div style="display: flex; gap: 12px; margin-bottom: 16px">
              <a-radio-group v-model:value="roomsStatus" @change="loadRooms">
                <a-radio-button value="all">全部</a-radio-button>
                <a-radio-button value="active">活跃</a-radio-button>
                <a-radio-button value="dissolved">已解散</a-radio-button>
              </a-radio-group>
              <a-button @click="loadRooms" :loading="roomsLoading">
                刷新
              </a-button>
            </div>
            <div class="room-cards">
              <div
                v-for="room in rooms"
                :key="room.id"
                class="room-card"
                @click="viewRoomDetail(room.id)"
              >
                <div class="room-header">
                  <span class="room-code">{{ room.room_code }}</span>
                  <a-tag :color="room.status === 'active' ? 'green' : 'default'">
                    {{ room.status === 'active' ? '活跃' : '已解散' }}
                  </a-tag>
                </div>
                <div class="room-info">
                  <div>类型: {{ room.room_type === 'texas' ? '德扑' : '牛牛' }}</div>
                  <div>比例: {{ room.chip_rate }}</div>
                  <div>创建者: {{ room.creator_nickname }}</div>
                  <div>成员数: {{ room.member_count }} (在线: {{ room.online_count }})</div>
                </div>
              </div>
            </div>
            <a-pagination
              v-model:current="roomsPage"
              v-model:page-size="roomsPageSize"
              :total="roomsTotal"
              @change="loadRooms"
              style="margin-top: 16px; text-align: center"
            />
          </div>
        </a-tab-pane>

        <!-- 操作记录 -->
        <a-tab-pane key="history" tab="操作记录">
          <div class="tab-content">
            <a-button @click="loadHistory" :loading="historyLoading" style="margin-bottom: 16px">
              刷新
            </a-button>
            <div class="table-container">
              <div class="table-row header-row">
                <div class="cell">用户</div>
                <div class="cell">房间号</div>
                <div class="cell">操作</div>
                <div class="cell">目标用户</div>
                <div class="cell">描述</div>
                <div class="cell">时间</div>
              </div>
              <div
                v-for="record in history"
                :key="record.id"
                class="table-row"
              >
                <div class="cell">{{ record.user_nickname || record.nickname || '-' }}</div>
                <div class="cell">{{ record.room_code || '-' }}</div>
                <div class="cell">{{ renderOperationType(record.operation_type) }}</div>
                <div class="cell">{{ renderTargetUser(record) }}</div>
                <div class="cell">{{ renderDescription(record) }}</div>
                <div class="cell">{{ formatDateTime(record.created_at) }}</div>
              </div>
            </div>
            <a-pagination
              v-model:current="historyPage"
              v-model:page-size="historyPageSize"
              :total="historyTotal"
              @change="loadHistory"
              style="margin-top: 16px; text-align: center"
            />
          </div>
        </a-tab-pane>
      </a-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { LeftOutlined } from '@ant-design/icons-vue'
import * as adminApi from '@/api/admin'
import dayjs from 'dayjs'

const router = useRouter()

const activeTab = ref('users')

// 用户管理
const users = ref<any[]>([])
const usersPage = ref(1)
const usersPageSize = ref(20)
const usersTotal = ref(0)
const usersLoading = ref(false)

// 房间管理
const rooms = ref<any[]>([])
const roomsPage = ref(1)
const roomsPageSize = ref(20)
const roomsTotal = ref(0)
const roomsStatus = ref('all')
const roomsLoading = ref(false)

// 操作记录
const history = ref<any[]>([])
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
const historyLoading = ref(false)

// 格式化日期时间
const formatDateTime = (dateStr?: string) => {
  if (!dateStr) {
    return '-'
  }
  return dayjs(dateStr).format('MM-DD HH:mm')
}

const operationTypeLabels: Record<string, string> = {
  create: '创建房间',
  join: '加入房间',
  leave: '离开房间',
  return: '返回房间',
  kick: '踢出房间',
  settlement_confirmed: '确认结算',
}

const renderOperationType = (type?: string) => {
  if (!type) {
    return '-'
  }
  return operationTypeLabels[type] || type
}

const renderTargetUser = (record: any) => {
  if (record?.target_nickname) {
    return record.target_nickname
  }
  if (record?.target_user_id) {
    return `ID ${record.target_user_id}`
  }
  return '-'
}

const renderDescription = (record: any) => {
  if (record?.operation_type === 'settlement_confirmed' && record?.metadata) {
    try {
      const meta = record.metadata as any
      const batch =
        typeof meta?.batch === 'string' && meta.batch
          ? `批次 ${meta.batch.slice(0, 8)}`
          : ''
      const chipRate =
        typeof meta?.chip_rate === 'string' && meta.chip_rate
          ? `比例 ${meta.chip_rate}`
          : ''
      const details = Array.isArray(meta?.details) ? meta.details : []
      const items = (details as any[])
        .map((item) => {
          const name =
            typeof item?.nickname === 'string' && item.nickname
              ? item.nickname
              : item?.user_id
              ? `ID ${item.user_id}`
              : ''
          if (!name) {
            return ''
          }
          const chip =
            typeof item?.chip_amount === 'number' ? item.chip_amount : null
          const rmb =
            typeof item?.rmb_amount === 'number'
              ? Number(item.rmb_amount).toFixed(2)
              : null
          const chipText = chip !== null ? `${chip}筹码` : ''
          const rmbText = rmb !== null ? `¥${rmb}` : ''
          const amountText =
            chipText && rmbText
              ? `${chipText}（${rmbText}）`
              : chipText || rmbText
          return amountText ? `${name}: ${amountText}` : name
        })
        .filter((text) => !!text)
      const summary = items.join('，')
      return [batch, chipRate, summary].filter((text) => !!text).join(' | ') || record.description || '-'
    } catch (error) {
      return record.description || '-'
    }
  }
  return record?.description || '-'
}

// 加载用户列表
const loadUsers = async () => {
  usersLoading.value = true
  try {
    const res = await adminApi.getUsers(usersPage.value, usersPageSize.value)
    users.value = res.data.users
    usersTotal.value = res.data.total
  } catch (error) {
    message.error('加载用户列表失败')
  } finally {
    usersLoading.value = false
  }
}

// 加载房间列表
const loadRooms = async () => {
  roomsLoading.value = true
  try {
    const res = await adminApi.getRooms(roomsStatus.value, roomsPage.value, roomsPageSize.value)
    rooms.value = res.data.rooms
    roomsTotal.value = res.data.total
  } catch (error) {
    message.error('加载房间列表失败')
  } finally {
    roomsLoading.value = false
  }
}

// 加载操作记录
const loadHistory = async () => {
  historyLoading.value = true
  try {
    const res = await adminApi.getRoomMemberHistory(undefined, undefined, historyPage.value, historyPageSize.value)
    history.value = res.data.records
    historyTotal.value = res.data.total
  } catch (error) {
    message.error('加载操作记录失败')
  } finally {
    historyLoading.value = false
  }
}

// 查看房间详情
const viewRoomDetail = (roomId: number) => {
  router.push(`/room/${roomId}`)
}

onMounted(() => {
  loadUsers()
  loadRooms()
  loadHistory()
})
</script>

<style scoped>
.admin-container {
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

.content {
  padding: 20px;
}

.tab-content {
  background: white;
  border-radius: 12px;
  padding: 20px;
}

.table-container {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}

.table-row {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr 1.2fr 2fr 1.2fr;
  border-bottom: 1px solid #f0f0f0;
}

.table-row:last-child {
  border-bottom: none;
}

.header-row {
  background: #fafafa;
  font-weight: 600;
}

.header-row .cell {
  color: #333;
}

.cell {
  padding: 12px;
  font-size: 14px;
  color: #666;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.room-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.room-card {
  background: #fafafa;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s;
}

.room-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.room-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.room-code {
  font-size: 18px;
  font-weight: bold;
  color: #667eea;
}

.room-info {
  font-size: 14px;
  color: #666;
  line-height: 1.8;
}

@media (max-width: 768px) {
  .table-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .cell {
    white-space: normal;
  }
}
</style>

