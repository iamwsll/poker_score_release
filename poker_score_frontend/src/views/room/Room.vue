<template>
  <div class="room-container" v-if="roomStore.roomInfo">
    <!-- 顶部栏 -->
    <div class="room-header">
      <a-button @click="handleLeaveRoom" danger>离开房间</a-button>
      <span class="nickname">{{ userStore.user?.nickname }}</span>
    </div>

    <!-- 功能按钮行 -->
    <div class="action-row">
      <a-button @click="showInviteModal = true">
        <ShareAltOutlined /> 邀请好友
      </a-button>
      <a-button @click="showRankingModal = true">
        <TrophyOutlined /> 积分排行
      </a-button>
    </div>

    <!-- 积分显示 -->
    <div class="balance-display">
      <div class="balance-item">
        <div class="label">桌面积分</div>
        <div class="value">{{ roomStore.roomInfo.table_balance }}</div>
      </div>
      <div class="balance-item">
        <div class="label">我的积分</div>
        <div class="value" :class="{ negative: roomStore.roomInfo.my_balance < 0 }">
          {{ roomStore.roomInfo.my_balance }}
        </div>
      </div>
    </div>

    <!-- 操作记录 -->
    <div class="operations-container">
      <h3>房间历史操作记录</h3>
      <div class="operations-list">
        <div
          v-for="op in roomStore.operations"
          :key="op.id"
          class="operation-item"
        >
          <span class="time">{{ formatTime(op.created_at) }}</span>
          <span class="desc">{{ formatOperationDescription(op) }}</span>
        </div>
        <div v-if="roomStore.operations.length === 0" class="empty">
          暂无操作记录
        </div>
      </div>
    </div>

    <!-- 底部操作按钮 -->
    <div class="bottom-actions">
      <a-button @click="showMoreModal = true" type="text">
        <MoreOutlined /> 更多功能
      </a-button>
      <a-button
        type="primary"
        size="large"
        @click="roomStore.roomInfo.room_type === 'niuniu' ? showNiuniuBetModal = true : showBetModal = true"
      >
        {{ roomStore.roomInfo.room_type === 'niuniu' ? '下注' : '支出' }}
      </a-button>
      <a-button
        size="large"
        @click="showWithdrawModal = true"
      >
        收回
      </a-button>
    </div>

    <!-- 邀请好友弹窗 -->
    <a-modal v-model:open="showInviteModal" title="邀请好友" :footer="null">
      <div class="invite-content">
        <div class="room-code-display">
          房间号: <span class="code">{{ roomStore.roomInfo.room_code }}</span>
        </div>
        <p style="color: #999; margin-top: 16px">分享房间号给好友即可加入</p>
      </div>
    </a-modal>

    <!-- 积分排行弹窗 -->
    <a-modal
      v-model:open="showRankingModal"
      title="积分排行"
      :footer="null"
      width="90%"
      :style="{ maxWidth: '500px' }"
    >
      <div class="ranking-list">
        <div
          v-for="member in sortedMembers"
          :key="member.user_id"
          class="ranking-item"
        >
          <span class="nickname">{{ member.nickname }}</span>
          <span class="balance" :class="{ negative: member.balance < 0 }">
            {{ member.balance }}
          </span>
        </div>
      </div>
      <a-button
        type="primary"
        block
        size="large"
        @click="handleInitiateSettlement"
        :loading="settlementLoading"
        style="margin-top: 20px"
      >
        结算这局
      </a-button>
    </a-modal>

    <!-- 德扑支出弹窗 -->
    <a-drawer
      v-model:open="showBetModal"
      title="支出"
      placement="bottom"
      :height="400"
    >
      <div class="drawer-content">
        <a-input-number
          v-model:value="betAmount"
          :min="1"
          size="large"
          style="width: 100%"
          placeholder="请输入支出金额"
        />

        <div class="history-amounts" v-if="historyBetAmounts.length > 0">
          <div class="history-title">历史金额</div>
          <div class="amount-buttons">
            <a-button
              v-for="amount in historyBetAmounts"
              :key="amount"
              @click="betAmount = amount"
            >
              {{ amount }}
            </a-button>
          </div>
        </div>

        <a-button
          type="primary"
          block
          size="large"
          @click="handleBet"
          :loading="betLoading"
          style="margin-top: 20px"
        >
          确认
        </a-button>
      </div>
    </a-drawer>

    <!-- 收回弹窗 -->
    <a-drawer
      v-model:open="showWithdrawModal"
      title="收回"
      placement="bottom"
      :height="400"
    >
      <div class="drawer-content">
        <a-input-number
          v-model:value="withdrawAmount"
          :min="0"
          size="large"
          style="width: 100%"
          placeholder="请输入收回金额"
        />

        <div class="history-amounts" v-if="historyWithdrawAmounts.length > 0">
          <div class="history-title">历史金额</div>
          <div class="amount-buttons">
            <a-button
              v-for="amount in historyWithdrawAmounts"
              :key="amount"
              @click="withdrawAmount = amount"
            >
              {{ amount }}
            </a-button>
          </div>
        </div>

        <div style="display: flex; gap: 12px; margin-top: 20px">
          <a-button
            size="large"
            @click="handleWithdrawAll"
            :loading="withdrawLoading"
            style="flex: 1"
          >
            全收
          </a-button>
          <a-button
            type="primary"
            size="large"
            @click="handleWithdraw"
            :loading="withdrawLoading"
            style="flex: 1"
          >
            确认
          </a-button>
        </div>
      </div>
    </a-drawer>

    <!-- 牛牛下注弹窗 -->
    <a-modal
      v-model:open="showNiuniuBetModal"
      title="牛牛下注"
      @ok="handleNiuniuBet"
      :confirmLoading="niuniuBetLoading"
      width="90%"
      :style="{ maxWidth: '500px' }"
    >
      <a-form layout="vertical">
        <a-form-item label="各玩家下注金额">
          <div v-if="roomStore.roomInfo?.members.length" class="niuniu-bet-amounts">
            <div
              v-for="member in roomStore.roomInfo.members"
              :key="member.user_id"
              class="niuniu-bet-row"
            >
              <div class="niuniu-bet-main">
                <span class="niuniu-bet-nickname">{{ member.nickname }}</span>
                <a-input-number
                  v-model:value="niuniuBetForm.amounts[member.user_id]"
                  :min="0"
                  size="large"
                  class="niuniu-bet-input"
                  placeholder="请输入下注金额"
                />
              </div>
              <div class="niuniu-bet-quick">
                <a-button
                  v-for="amount in niuniuQuickAmounts"
                  :key="amount"
                  size="small"
                  @click="setNiuniuBetAmount(member.user_id, amount)"
                >
                  {{ amount }}
                </a-button>
              </div>
            </div>
          </div>
          <div v-else class="niuniu-bet-placeholder">
            暂无可下注玩家
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 更多功能弹窗 -->
    <a-modal
      v-model:open="showMoreModal"
      title="更多功能"
      :footer="null"
    >
      <a-button block size="large" @click="handleKickUser">
        有人走了？
      </a-button>
    </a-modal>

    <!-- 踢人弹窗 -->
    <a-modal
      v-model:open="showKickModal"
      title="选择离开的用户"
      @ok="confirmKick"
      :confirmLoading="kickLoading"
    >
      <a-radio-group v-model:value="kickUserId" style="width: 100%">
        <div v-for="member in roomStore.roomInfo.members" :key="member.user_id" style="margin-bottom: 12px">
          <a-radio :value="member.user_id">{{ member.nickname }}</a-radio>
        </div>
      </a-radio-group>
    </a-modal>

    <!-- 结算方案弹窗 -->
    <a-modal
      v-model:open="showSettlementPlanModal"
      title="结算方案"
      :footer="initiatedByMe ? undefined : null"
      @ok="handleConfirmSettlement"
      :confirmLoading="confirmSettlementLoading"
      width="90%"
      :style="{ maxWidth: '500px' }"
    >
      <div class="settlement-plan">
        <div v-if="settlementPlan.length === 0" class="plan-empty">暂无结算方案</div>
        <div v-else>
          <div v-if="settlementConfirmed && confirmedInfo" class="plan-status">
            已由 {{ confirmedInfo.nickname || '玩家' }} 在
            {{ formatDateTime(confirmedInfo.confirmed_at) }} 确认
          </div>
          <div
            v-for="(item, index) in settlementPlan"
            :key="index"
            class="plan-item"
          >
            {{ item.description }}
          </div>
        </div>
      </div>
      <template #footer v-if="initiatedByMe">
        <a-button @click="showSettlementPlanModal = false">取消</a-button>
        <a-button
          type="primary"
          @click="handleConfirmSettlement"
          :loading="confirmSettlementLoading"
          :disabled="settlementConfirmed"
        >
          确认结算
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  ShareAltOutlined,
  TrophyOutlined,
  MoreOutlined
} from '@ant-design/icons-vue'
import { useUserStore } from '@/stores/user'
import { useRoomStore } from '@/stores/room'
import type { RoomOperation, SettlementPlanItem, SettlementContext, NiuniuBetDetail, RoomMember } from '@/stores/room'
import { storeToRefs } from 'pinia'
import * as roomApi from '@/api/room'

defineOptions({
  name: 'RoomPage'
})
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const roomStore = useRoomStore()
const { settlementContext } = storeToRefs(roomStore)

const roomId = ref(Number(route.params.id))

// 弹窗状态
const showInviteModal = ref(false)
const showRankingModal = ref(false)
const showBetModal = ref(false)
const showWithdrawModal = ref(false)
const showNiuniuBetModal = ref(false)
const showMoreModal = ref(false)
const showKickModal = ref(false)
const showSettlementPlanModal = ref(false)

// 表单数据
const betAmount = ref<number>()
const withdrawAmount = ref<number>()
const kickUserId = ref<number>()
const niuniuBetForm = reactive({
  amounts: {} as Record<number, number | undefined | null>
})

const niuniuQuickAmounts = [1, 2, 3, 4, 5]

// 加载状态
const betLoading = ref(false)
const withdrawLoading = ref(false)
const niuniuBetLoading = ref(false)
const kickLoading = ref(false)
const settlementLoading = ref(false)
const confirmSettlementLoading = ref(false)

// 历史金额
const historyBetAmounts = ref<number[]>([])
const historyWithdrawAmounts = ref<number[]>([])

// 结算相关
const settlementPlan = ref<SettlementPlanItem[]>([])
const initiatedByMe = ref(false)
const settlementConfirmed = ref(false)
const confirmedInfo = ref<{ nickname: string; confirmed_at: string } | null>(null)

// 计算属性：排序后的成员列表
const sortedMembers = computed(() => {
  if (!roomStore.roomInfo) return []
  return [...roomStore.roomInfo.members].sort((a, b) => b.balance - a.balance)
})

const memberNicknameMap = computed(() => {
  const map = new Map<number, string>()
  if (roomStore.roomInfo) {
    roomStore.roomInfo.members.forEach((member) => {
      map.set(member.user_id, member.nickname)
    })
  }
  return map
})

const clearNiuniuBetAmounts = () => {
  Object.keys(niuniuBetForm.amounts).forEach((key) => {
    delete niuniuBetForm.amounts[Number(key)]
  })
}

const syncNiuniuBetMembers = (members?: RoomMember[]) => {
  if (!members || members.length === 0) {
    clearNiuniuBetAmounts()
    return
  }

  const availableIds = new Set(members.map((member) => member.user_id))

  Object.keys(niuniuBetForm.amounts).forEach((key) => {
    const userId = Number(key)
    if (!availableIds.has(userId)) {
      delete niuniuBetForm.amounts[userId]
    }
  })

  members.forEach((member) => {
    if (!(member.user_id in niuniuBetForm.amounts)) {
      niuniuBetForm.amounts[member.user_id] = undefined
    }
  })
}

const setNiuniuBetAmount = (userId: number, amount: number) => {
  niuniuBetForm.amounts[userId] = amount
}

watch(
  settlementContext,
  (context) => {
    if (context) {
      settlementPlan.value = context.settlement_plan ?? []
      initiatedByMe.value = context.initiated_by === (userStore.user?.id ?? 0)
      settlementConfirmed.value = context.confirmed ?? false
      confirmedInfo.value = context.confirmed
        ? {
            nickname: context.confirmed_by_nickname ?? '',
            confirmed_at: context.confirmed_at ?? context.initiated_at,
          }
        : null
      showSettlementPlanModal.value = true
    } else {
      settlementPlan.value = []
      initiatedByMe.value = false
      settlementConfirmed.value = false
      confirmedInfo.value = null
      if (showSettlementPlanModal.value) {
        showSettlementPlanModal.value = false
      }
    }
  },
  { deep: true }
)

watch(
  () => showSettlementPlanModal.value,
  (open) => {
    if (!open && settlementContext.value) {
      roomStore.setSettlementContext(null)
    }
  }
)

watch(
  () => roomStore.roomInfo?.members,
  (members) => {
    syncNiuniuBetMembers(members)
  },
  { immediate: true, deep: true }
)

// 格式化时间
const formatTime = (dateStr: string) => {
  const date = new Date(dateStr)
  return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`
}

const formatDateTime = (dateStr: string) => {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) {
    return dateStr
  }
  const y = date.getFullYear()
  const m = (date.getMonth() + 1).toString().padStart(2, '0')
  const d = date.getDate().toString().padStart(2, '0')
  const hh = date.getHours().toString().padStart(2, '0')
  const mm = date.getMinutes().toString().padStart(2, '0')
  const ss = date.getSeconds().toString().padStart(2, '0')
  return `${y}-${m}-${d} ${hh}:${mm}:${ss}`
}

const extractNiuniuBetDetails = (op: RoomOperation): NiuniuBetDetail[] => {
  if (op.bets && op.bets.length > 0) {
    return op.bets
  }

  if (!op.description) {
    return []
  }

  try {
    const parsed = JSON.parse(op.description)
    if (!Array.isArray(parsed)) {
      return []
    }

    const details: NiuniuBetDetail[] = []
    parsed.forEach((item) => {
      if (!item || typeof item !== 'object') {
        return
      }

      const record = item as {
        to_user_id?: unknown
        amount?: unknown
        to_nickname?: unknown
      }

      const toUserId = Number(record.to_user_id)
      const amount = Number(record.amount)

      if (!Number.isFinite(toUserId) || !Number.isFinite(amount)) {
        return
      }

      const detail: NiuniuBetDetail = {
        to_user_id: toUserId,
        amount,
      }

      if (typeof record.to_nickname === 'string' && record.to_nickname.trim().length > 0) {
        detail.to_nickname = record.to_nickname
      }

      details.push(detail)
    })

    return details
  } catch (error) {
    console.error('解析牛牛下注操作描述失败:', error, op.description)
    return []
  }
}

// 格式化操作描述
const formatOperationDescription = (op: RoomOperation) => {
  const parts: string[] = []

  if (op.nickname) {
    parts.push(op.nickname)
  }

  switch (op.operation_type) {
    case 'kick':
      if (op.target_nickname) {
        parts.push(`踢出了${op.target_nickname}`)
      } else {
        parts.push(op.description)
      }
      break
    case 'niuniu_bet': {
      const bets = extractNiuniuBetDetails(op)

      if (bets.length > 0) {
        const total = typeof op.amount === 'number'
          ? op.amount
          : bets.reduce((sum, bet) => sum + (bet.amount ?? 0), 0)

        const detailText = bets
          .map((bet) => {
            const nickname = bet.to_nickname && bet.to_nickname.trim().length > 0
              ? bet.to_nickname
              : memberNicknameMap.value.get(bet.to_user_id) ?? `用户${bet.to_user_id}`
            return `给${nickname}下了${bet.amount}积分`
          })
          .join('，')

        if (total > 0) {
          parts.push(`共下注${total}积分（${detailText}）`)
        } else {
          parts.push(detailText)
        }
      } else {
        parts.push(op.description)
      }
      break
    }
    default:
      parts.push(op.description)
  }

  return parts.join(' ')
}

// 加载房间信息
const loadRoomInfo = async () => {
  try {
    const res = await roomApi.getRoomDetails(roomId.value)
    roomStore.setRoomInfo(res.data)

    // 加载操作历史
    const opsRes = await roomApi.getOperations(roomId.value)
    roomStore.setOperations(opsRes.data.operations)

    // 加载历史金额
    const historyRes = await roomApi.getHistoryAmounts(roomId.value)
    historyBetAmounts.value = historyRes.data.bet_amounts || []
    historyWithdrawAmounts.value = historyRes.data.withdraw_amounts || []

    // 连接WebSocket
    roomStore.connectWebSocket(roomId.value)
  } catch (error) {
    console.error(error)
    message.error('加载房间信息失败')
    router.push('/')
  }
}

// 离开房间
const handleLeaveRoom = () => {
  Modal.confirm({
    title: '确认离开',
    content: '您确定要离开房间吗？',
    onOk: async () => {
      try {
        await roomApi.leaveRoom(roomId.value)
        roomStore.clearRoomInfo()
        router.push('/')
      } catch (error) {
        console.error(error)
        // 错误已处理
      }
    }
  })
}

// 支出（德扑）
const handleBet = async () => {
  if (!betAmount.value || betAmount.value <= 0) {
    message.warning('请输入有效的支出金额')
    return
  }

  betLoading.value = true
  try {
			const res = await roomApi.bet(roomId.value, betAmount.value)
    message.success('支出成功')
    showBetModal.value = false
    betAmount.value = undefined
			roomStore.updateMyBalance(res.data.my_balance)
			roomStore.updateTableBalance(res.data.table_balance)
  } catch (error) {
    console.error(error)
    // 错误已处理
  } finally {
    betLoading.value = false
  }
}

// 收回
const handleWithdraw = async () => {
  if (!withdrawAmount.value || withdrawAmount.value <= 0) {
    message.warning('请输入有效的收回金额')
    return
  }

  withdrawLoading.value = true
  try {
			const res = await roomApi.withdraw(roomId.value, withdrawAmount.value)
    message.success('收回成功')
    showWithdrawModal.value = false
    withdrawAmount.value = undefined
			roomStore.updateMyBalance(res.data.my_balance)
			roomStore.updateTableBalance(res.data.table_balance)
  } catch (error) {
    console.error(error)
    // 错误已处理
  } finally {
    withdrawLoading.value = false
  }
}

// 全收
const handleWithdrawAll = async () => {
  withdrawLoading.value = true
  try {
			const res = await roomApi.withdraw(roomId.value, 0)
    message.success('全收成功')
    showWithdrawModal.value = false
    withdrawAmount.value = undefined
			roomStore.updateMyBalance(res.data.my_balance)
			roomStore.updateTableBalance(res.data.table_balance)
  } catch (error) {
    console.error(error)
    // 错误已处理
  } finally {
    withdrawLoading.value = false
  }
}

// 牛牛下注
const handleNiuniuBet = async () => {
  const entries = Object.entries(niuniuBetForm.amounts)

  if (entries.length === 0) {
    message.warning('暂无可下注玩家')
    return
  }

  const invalidEntry = entries.find(([, value]) => value !== undefined && value !== null && (!Number.isFinite(value) || value < 0))

  if (invalidEntry) {
    const [userId] = invalidEntry
    const nickname = memberNicknameMap.value.get(Number(userId)) ?? ''
    const targetName = nickname ? `「${nickname}」` : '该玩家'
    message.warning(`${targetName}的下注金额无效`)
    return
  }

  const bets = entries
    .filter(([, value]) => value !== undefined && value !== null && value > 0)
    .map(([userId, value]) => ({
      to_user_id: Number(userId),
      amount: value as number
    }))

  if (bets.length === 0) {
    message.warning('请至少为一位玩家输入大于0的下注金额')
    return
  }

  niuniuBetLoading.value = true
  try {
    const res = await roomApi.niuniuBet(roomId.value, bets)
    message.success('下注成功')
    showNiuniuBetModal.value = false
    clearNiuniuBetAmounts()
    syncNiuniuBetMembers(roomStore.roomInfo?.members)
    roomStore.updateMyBalance(res.data.my_balance)
  } catch (error) {
    console.error(error)
    // 错误已处理
  } finally {
    niuniuBetLoading.value = false
  }
}

// 踢人
const handleKickUser = () => {
  showMoreModal.value = false
  showKickModal.value = true
}

const confirmKick = async () => {
  if (!kickUserId.value) {
    message.warning('请选择要踢出的用户')
    return
  }

  kickLoading.value = true
  try {
    await roomApi.kickUser(roomId.value, kickUserId.value)
    message.success('已踢出用户')
    showKickModal.value = false
    kickUserId.value = undefined
    loadRoomInfo()
  } catch (error) {
    console.error(error)
    // 错误已处理
  } finally {
    kickLoading.value = false
  }
}

// 发起结算
const handleInitiateSettlement = async () => {
  settlementLoading.value = true
  try {
    const res = await roomApi.initiateSettlement(roomId.value)
    if (res.data.can_settle) {
      showRankingModal.value = false
      const context: SettlementContext = {
        initiated_by: userStore.user?.id ?? 0,
        initiated_by_nickname: userStore.user?.nickname ?? '',
        initiated_at: new Date().toISOString(),
        settlement_plan: res.data.settlement_plan ?? [],
        table_balance: res.data.table_balance ?? 0,
        confirmed: false,
      }
      roomStore.setSettlementContext(context)
    }
  } catch (error) {
    console.error(error)
    // 错误已处理
  } finally {
    settlementLoading.value = false
  }
}

// 确认结算
const handleConfirmSettlement = async () => {
  confirmSettlementLoading.value = true
  try {
    await roomApi.confirmSettlement(roomId.value)
    message.success('结算完成')
    showSettlementPlanModal.value = false
    loadRoomInfo()
  } catch (error) {
    console.error(error)
    // 错误已处理
  } finally {
    confirmSettlementLoading.value = false
  }
}

onMounted(() => {
  loadRoomInfo()
})

onUnmounted(() => {
  roomStore.disconnectWebSocket()
})
</script>

<style scoped>
.room-container {
  height: 100vh;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
  overflow: hidden;
}

.room-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.nickname {
  font-weight: 600;
  font-size: 16px;
}

.action-row {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: white;
  margin-bottom: 1px;
}

.action-row button {
  flex: 1;
}

.balance-display {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  padding: 24px 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.balance-item {
  background: rgba(255, 255, 255, 0.95);
  border-radius: 12px;
  padding: 20px;
  text-align: center;
}

.balance-item .label {
  color: #666;
  font-size: 14px;
  margin-bottom: 8px;
}

.balance-item .value {
  color: #333;
  font-size: 32px;
  font-weight: bold;
}

.balance-item .value.negative {
  color: #ff4d4f;
}

.operations-container {
  flex: 1;
  padding: 16px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.operations-container h3 {
  margin: 0 0 12px;
  font-size: 16px;
  color: #333;
}

.operations-list {
  flex: 1;
  overflow-y: auto;
  background: white;
  border-radius: 12px;
  padding: 12px;
}

.operation-item {
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  gap: 12px;
}

.operation-item:last-child {
  border-bottom: none;
}

.time {
  color: #999;
  font-size: 12px;
  flex-shrink: 0;
}

.desc {
  color: #333;
  font-size: 14px;
}

.empty {
  text-align: center;
  padding: 40px 0;
  color: #999;
}

.bottom-actions {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: white;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.05);
}

.bottom-actions button:first-child {
  flex-shrink: 0;
}

.bottom-actions button:not(:first-child) {
  flex: 1;
}

.invite-content {
  text-align: center;
  padding: 20px 0;
}

.room-code-display {
  font-size: 18px;
  color: #333;
}

.code {
  font-size: 32px;
  font-weight: bold;
  color: #667eea;
  letter-spacing: 4px;
}

.ranking-list {
  max-height: 400px;
  overflow-y: auto;
}

.ranking-item {
  display: flex;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.ranking-item:last-child {
  border-bottom: none;
}

.ranking-item .balance.negative {
  color: #ff4d4f;
}

.drawer-content {
  padding: 20px 0;
}

.history-amounts {
  margin-top: 20px;
}

.history-title {
  font-size: 14px;
  color: #666;
  margin-bottom: 12px;
}

.amount-buttons {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.settlement-plan {
  max-height: 400px;
  overflow-y: auto;
}

.plan-item {
  padding: 12px;
  background: #f5f5f5;
  border-radius: 8px;
  margin-bottom: 8px;
  font-size: 14px;
  color: #333;
}

.plan-empty {
  text-align: center;
  color: #999;
  padding: 24px 0;
}

.plan-status {
  margin-bottom: 12px;
  padding: 12px;
  border-radius: 8px;
  background: rgba(24, 144, 255, 0.1);
  color: #1890ff;
  font-size: 14px;
}

.niuniu-bet-amounts {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.niuniu-bet-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
}

.niuniu-bet-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.niuniu-bet-nickname {
  flex: 1;
  font-size: 14px;
  color: #333;
}

.niuniu-bet-input {
  width: 140px;
}

.niuniu-bet-quick {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.niuniu-bet-placeholder {
  color: #999;
  font-size: 13px;
  text-align: center;
  padding: 24px 0;
}
</style>

