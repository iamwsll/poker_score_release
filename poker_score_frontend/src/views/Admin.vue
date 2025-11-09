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
                <div class="cell">创建时间</div>
                <div class="cell">操作</div>
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
                <div class="cell">{{ formatDateTime(user.created_at) }}</div>
                <div class="cell">
                  <a-button type="link" size="small" @click="openEditUserModal(user)">编辑</a-button>
                </div>
              </div>
            </div>
            <a-modal
              v-model:open="editUserVisible"
              title="编辑用户"
              :confirm-loading="editUserLoading"
              destroyOnClose
              @ok="handleEditUserSubmit"
              @cancel="handleEditUserCancel"
            >
              <a-form layout="vertical">
                <a-form-item label="昵称">
                  <a-input v-model:value="editUserForm.nickname" maxlength="50" />
                </a-form-item>
                <a-form-item label="手机号">
                  <a-input v-model:value="editUserForm.phone" maxlength="11" />
                </a-form-item>
                <a-form-item label="角色">
                  <a-select v-model:value="editUserForm.role">
                    <a-select-option value="user">普通用户</a-select-option>
                    <a-select-option value="admin">管理员</a-select-option>
                  </a-select>
                </a-form-item>
                <a-form-item label="密码">
                  <a-input-password v-model:value="editUserForm.password" placeholder="不修改请留空" />
                </a-form-item>
              </a-form>
              <div class="modal-tip">若不修改密码，请留空（密码至少 6 位）。</div>
            </a-modal>
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
                @click="viewRoomDetail(room)"
              >
                <div class="room-header">
                  <span class="room-code">{{ room.room_code }}</span>
                  <a-tag :color="room.status === 'active' ? 'green' : 'default'">
                    {{ room.status === 'active' ? '活跃' : '已解散' }}
                  </a-tag>
                </div>
                <div class="room-info">
                  <div>类型: {{ room.room_type === 'texas' ? '单记分' : '多计分' }}</div>
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

      <a-modal
        :open="dissolvedRoomDetailVisible"
        :title="dissolvedRoomModalTitle"
        width="720px"
        :footer="null"
        destroy-on-close
        @cancel="handleCloseDissolvedDetail"
      >
        <div v-if="dissolvedRoomDetailLoading" class="detail-loading">
          <a-spin />
        </div>
        <div v-else-if="dissolvedRoomDetail" class="detail-content">
          <div v-if="dissolvedRoomDetail.room" class="detail-section">
            <div class="detail-section-title">房间信息</div>
            <div class="detail-item">房间号：{{ dissolvedRoomDetail.room.room_code }}</div>
            <div class="detail-item">类型：{{ renderRoomType(dissolvedRoomDetail.room.room_type) }}</div>
            <div class="detail-item">比例：{{ dissolvedRoomDetail.room.chip_rate }}</div>
            <div class="detail-item">
              状态：
              {{ dissolvedRoomDetail.room.status === 'active' ? '活跃' : '已解散' }}
            </div>
            <div class="detail-item">桌面积分：{{ dissolvedRoomDetail.table_balance }}</div>
            <div class="detail-item">
              解散时间：
              {{ dissolvedRoomDetail.room.dissolved_at ? formatDateTime(dissolvedRoomDetail.room.dissolved_at) : '-' }}
            </div>
          </div>

          <div class="detail-section">
            <div class="detail-section-title">成员</div>
            <div v-if="!dissolvedRoomDetail.members || !dissolvedRoomDetail.members.length" class="detail-empty">
              暂无成员数据
            </div>
            <div v-else class="detail-table">
              <div class="detail-table-header">
                <span>成员</span>
                <span>积分</span>
                <span>状态</span>
                <span>加入时间</span>
                <span>离开时间</span>
              </div>
              <div
                v-for="member in dissolvedRoomDetail.members"
                :key="member.user_id"
                class="detail-table-row"
              >
                <span>{{ member.nickname || `用户${member.user_id}` }}</span>
                <span>{{ member.balance ?? 0 }}</span>
                <span>{{ member.status === 'online' ? '在线' : '离线' }}</span>
                <span>{{ member.joined_at ? formatDateTime(member.joined_at) : '-' }}</span>
                <span>{{ member.left_at ? formatDateTime(member.left_at) : '-' }}</span>
              </div>
            </div>
          </div>

          <div class="detail-section">
            <div class="detail-section-title">最近操作</div>
            <div v-if="dissolvedRoomOperationsLoading" class="detail-loading detail-loading-inline">
              <a-spin />
            </div>
            <template v-else>
              <div v-if="!dissolvedRoomOperations?.list?.length" class="detail-empty">
                暂无操作记录
              </div>
              <div v-else>
                <div class="detail-operations">
                  <div
                    v-for="op in dissolvedRoomOperations.list"
                    :key="op.id"
                    class="detail-operation-item"
                  >
                    <div class="detail-operation-header">
                      <span class="detail-operation-user">{{ op.nickname || `用户${op.user_id}` }}</span>
                      <span class="detail-operation-time">{{ formatDateTime(op.created_at) }}</span>
                    </div>
                    <div class="detail-operation-desc">
                      <span class="detail-operation-type">{{ renderOperationType(op.operation_type) }}</span>
                      <span class="detail-operation-description">{{ op.description }}</span>
                      <span v-if="typeof op.amount === 'number'" class="detail-operation-amount">金额：{{ op.amount }}</span>
                    </div>
                  </div>
                </div>
                <a-pagination
                  v-model:current="dissolvedRoomDetailOperationsPage"
                  v-model:page-size="dissolvedRoomDetailOperationsPageSize"
                  :total="dissolvedRoomDetailOperationsTotal"
                  :show-size-changer="true"
                  :page-size-options="['10', '20', '30', '50', '100', '200']"
                  :hide-on-single-page="true"
                  size="small"
                  show-less-items
                  style="margin-top: 12px; text-align: right"
                  @change="handleOperationsPageChange"
                />
              </div>
            </template>
          </div>
        </div>
        <div v-else class="detail-empty">暂无数据</div>
      </a-modal>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { LeftOutlined } from '@ant-design/icons-vue'
import * as adminApi from '@/api/admin'
import dayjs from 'dayjs'

const router = useRouter()

const activeTab = ref('users')

interface EditUserForm {
  id: number
  phone: string
  nickname: string
  role: string
  password: string
}

// 用户管理
const users = ref<any[]>([])
const usersPage = ref(1)
const usersPageSize = ref(20)
const usersTotal = ref(0)
const usersLoading = ref(false)
const editUserVisible = ref(false)
const editUserLoading = ref(false)
const editUserForm = reactive<EditUserForm>({
  id: 0,
  phone: '',
  nickname: '',
  role: 'user',
  password: ''
})

const resetEditUserForm = () => {
  editUserForm.id = 0
  editUserForm.phone = ''
  editUserForm.nickname = ''
  editUserForm.role = 'user'
  editUserForm.password = ''
}

// 房间管理
const DEFAULT_OPERATION_PAGE_SIZE = 20
const rooms = ref<any[]>([])
const roomsPage = ref(1)
const roomsPageSize = ref(20)
const roomsTotal = ref(0)
const roomsStatus = ref('all')
const roomsLoading = ref(false)
const dissolvedRoomDetailVisible = ref(false)
const dissolvedRoomDetailLoading = ref(false)
const dissolvedRoomDetail = ref<any | null>(null)
const dissolvedRoomDetailRoomId = ref<number | null>(null)
const dissolvedRoomDetailOperationsPage = ref(1)
const dissolvedRoomDetailOperationsPageSize = ref(DEFAULT_OPERATION_PAGE_SIZE)
const dissolvedRoomDetailOperationsTotal = ref(0)
const dissolvedRoomOperationsLoading = ref(false)

const dissolvedRoomModalTitle = computed(() => {
  if (dissolvedRoomDetail.value?.room?.room_code) {
    return `房间 ${dissolvedRoomDetail.value.room.room_code}`
  }
  return '房间详情'
})

const dissolvedRoomOperations = computed(() => {
  const operations = dissolvedRoomDetail.value?.operations
  if (Array.isArray(operations)) {
    return {
      list: operations,
      total: operations.length,
      page: dissolvedRoomDetailOperationsPage.value,
      page_size: dissolvedRoomDetailOperationsPageSize.value
    }
  }
  return operations || null
})

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

const renderRoomType = (type?: string) => {
  if (type === 'texas') {
    return '单计分'
  }
  if (type === 'niuniu') {
    return '多计分'
  }
  return type || '-'
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

const resetDissolvedRoomOperations = () => {
  dissolvedRoomDetailOperationsPage.value = 1
  dissolvedRoomDetailOperationsPageSize.value = DEFAULT_OPERATION_PAGE_SIZE
  dissolvedRoomDetailOperationsTotal.value = 0
  dissolvedRoomOperationsLoading.value = false
}

const resetDissolvedRoomDetailState = () => {
  dissolvedRoomDetail.value = null
  dissolvedRoomDetailRoomId.value = null
  resetDissolvedRoomOperations()
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

const openEditUserModal = (user: any) => {
  editUserForm.id = user?.id ?? 0
  editUserForm.phone = typeof user?.phone === 'string' ? user.phone : ''
  editUserForm.nickname = typeof user?.nickname === 'string' ? user.nickname : ''
  editUserForm.role = typeof user?.role === 'string' ? user.role : 'user'
  editUserForm.password = ''
  editUserVisible.value = true
}

const handleEditUserCancel = () => {
  if (editUserLoading.value) {
    return
  }
  editUserVisible.value = false
  resetEditUserForm()
}

const handleEditUserSubmit = async () => {
  const phone = editUserForm.phone.trim()
  const nickname = editUserForm.nickname.trim()
  const role = editUserForm.role
  const password = editUserForm.password.trim()

  if (!/^\d{11}$/.test(phone)) {
    message.error('手机号需为11位数字')
    return
  }
  if (!nickname) {
    message.error('请填写昵称')
    return
  }
  if (nickname.length > 50) {
    message.error('昵称长度不能超过50个字符')
    return
  }
  if (password && password.length < 6) {
    message.error('密码至少需要6位')
    return
  }

  const payload: { phone: string; nickname: string; role: string; password?: string } = {
    phone,
    nickname,
    role
  }

  if (password) {
    payload.password = password
  }

  editUserLoading.value = true
  try {
    await adminApi.updateUser(editUserForm.id, payload)
    message.success('用户信息已更新')
    editUserVisible.value = false
    resetEditUserForm()
    await loadUsers()
  } catch (error) {
    // 错误提示由全局拦截器处理
  } finally {
    editUserLoading.value = false
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
const viewRoomDetail = (room: any) => {
  if (!room || typeof room.id !== 'number') {
    return
  }

  if (room.status === 'active') {
    router.push(`/room/${room.id}`)
    return
  }

  resetDissolvedRoomDetailState()
  dissolvedRoomDetailRoomId.value = room.id
  dissolvedRoomDetailVisible.value = true
  dissolvedRoomDetailLoading.value = true
  loadDissolvedRoomDetail(room.id)
}

const loadDissolvedRoomDetail = async (roomId: number, options?: { onlyOperations?: boolean }) => {
  const onlyOperations = options?.onlyOperations === true

  if (onlyOperations) {
    dissolvedRoomOperationsLoading.value = true
  } else {
    dissolvedRoomDetailLoading.value = true
  }

  try {
    const res = await adminApi.getRoomDetails(roomId, {
      op_page: dissolvedRoomDetailOperationsPage.value,
      op_page_size: dissolvedRoomDetailOperationsPageSize.value
    })

    const detail = res.data ?? {}

    if (Array.isArray(detail.operations)) {
      detail.operations = {
        list: detail.operations,
        total: detail.operations.length,
        page: dissolvedRoomDetailOperationsPage.value,
        page_size: dissolvedRoomDetailOperationsPageSize.value
      }
    }

    dissolvedRoomDetail.value = detail

    const operations = detail.operations || {}
    const total =
      typeof operations.total === 'number'
        ? operations.total
        : Array.isArray(operations.list)
        ? operations.list.length
        : 0
    dissolvedRoomDetailOperationsTotal.value = total

    if (typeof operations.page === 'number' && operations.page > 0) {
      dissolvedRoomDetailOperationsPage.value = operations.page
    }
    if (typeof operations.page_size === 'number' && operations.page_size > 0) {
      dissolvedRoomDetailOperationsPageSize.value = operations.page_size
    }
  } catch (error) {
    message.error('加载房间详情失败')
    if (!onlyOperations) {
      dissolvedRoomDetailVisible.value = false
    }
  } finally {
    if (onlyOperations) {
      dissolvedRoomOperationsLoading.value = false
    } else {
      dissolvedRoomDetailLoading.value = false
    }
  }
}

const handleCloseDissolvedDetail = () => {
  if (dissolvedRoomDetailLoading.value) {
    return
  }
  dissolvedRoomDetailVisible.value = false
  resetDissolvedRoomDetailState()
}

const handleOperationsPageChange = (page: number, pageSize: number) => {
  const targetPage = page > 0 ? page : 1
  const targetPageSize = pageSize > 0 ? pageSize : DEFAULT_OPERATION_PAGE_SIZE

  const shouldFetch =
    targetPage !== dissolvedRoomDetailOperationsPage.value ||
    targetPageSize !== dissolvedRoomDetailOperationsPageSize.value

  dissolvedRoomDetailOperationsPage.value = targetPage
  dissolvedRoomDetailOperationsPageSize.value = targetPageSize

  if (!shouldFetch || dissolvedRoomDetailRoomId.value === null) {
    return
  }

  loadDissolvedRoomDetail(dissolvedRoomDetailRoomId.value, { onlyOperations: true })
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

.modal-tip {
  margin-top: 8px;
  color: #999;
  font-size: 12px;
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

.detail-loading {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 40px 0;
}

.detail-loading-inline {
  padding: 24px 0;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-section {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 16px;
  background: #fafafa;
}

.detail-section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
  color: #333;
}

.detail-item {
  font-size: 14px;
  color: #555;
  line-height: 1.8;
}

.detail-empty {
  font-size: 14px;
  color: #999;
  text-align: center;
  padding: 24px 0;
}

.detail-table {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-table-header,
.detail-table-row {
  display: grid;
  grid-template-columns: 1.5fr 1fr 1fr 1.5fr 1.5fr;
  gap: 12px;
  font-size: 14px;
}

.detail-table-header {
  font-weight: 600;
  color: #555;
}

.detail-table-row {
  padding: 8px 0;
  border-top: 1px dashed #e0e0e0;
  color: #666;
}

.detail-table-row:first-of-type {
  border-top: none;
}

.detail-operations {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-operation-item {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 12px 16px;
  background: white;
}

.detail-operation-header {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  margin-bottom: 8px;
  color: #999;
}

.detail-operation-user {
  font-weight: 600;
  color: #555;
}

.detail-operation-desc {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 14px;
  color: #555;
}

.detail-operation-type {
  color: #667eea;
}

.detail-operation-description {
  flex: 1;
}

.detail-operation-amount {
  color: #f56c6c;
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

