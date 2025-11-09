import { defineStore } from 'pinia'
import { ref } from 'vue'
import { message as antdMessage } from 'ant-design-vue'
import router from '@/router'
import { useUserStore } from '@/stores/user'

export interface RoomMember {
  user_id: number
  nickname: string
  balance: number
  status: string
}

type RoomMemberUpdate = Partial<Omit<RoomMember, 'user_id'>> & Pick<RoomMember, 'user_id'>

export interface NiuniuBetDetail {
  to_user_id: number
  to_nickname?: string
  amount: number
}

export interface SettlementDetail {
  user_id: number
  nickname?: string
  chip_amount: number
  rmb_amount?: number
}

export interface SettlementSummary {
  batch?: string
  chip_rate?: string
  settled_at?: string
  details: SettlementDetail[]
}

export interface RoomInfo {
  room_id: number
  room_code: string
  room_type: 'texas' | 'niuniu'
  chip_rate: string
  status: string
  created_by: number
  table_balance: number
  my_balance: number
  members: RoomMember[]
}

export interface RoomOperation {
  id: number
  user_id: number
  nickname: string
  operation_type: string
  amount?: number
  description: string
  target_user_id?: number
  target_nickname?: string
  created_at: string
  bets?: NiuniuBetDetail[]
  settlement_summary?: SettlementSummary
}

export interface SettlementPlanItem {
  from_user_id: number
  from_nickname: string
  to_user_id: number
  to_nickname: string
  chip_amount: number
  rmb_amount: number
  description: string
}

export interface SettlementContext {
  initiated_by: number
  initiated_by_nickname: string
  initiated_at: string
  settlement_plan: SettlementPlanItem[]
  table_balance: number
  confirmed: boolean
  confirmed_by?: number
  confirmed_by_nickname?: string
  confirmed_at?: string
}

export const useRoomStore = defineStore('room', () => {
  // 状态
  const roomInfo = ref<RoomInfo | null>(null)
  const operations = ref<RoomOperation[]>([])
  const ws = ref<WebSocket | null>(null)
  const currentRoomId = ref<number | null>(null)
  const settlementContext = ref<SettlementContext | null>(null)
  const userStore = useUserStore()

  // 设置房间信息
  function setRoomInfo(info: RoomInfo) {
    roomInfo.value = info
  }

  // 更新房间成员
  function updateMember(member: RoomMemberUpdate) {
    if (!roomInfo.value) return
    const index = roomInfo.value.members.findIndex((m) => m.user_id === member.user_id)

    if (index !== -1) {
      const existing = roomInfo.value.members[index]

      if (existing) {
        roomInfo.value.members[index] = {
          ...existing,
          ...member,
          nickname: member.nickname ?? existing.nickname,
          balance: member.balance ?? existing.balance,
          status: member.status ?? existing.status ?? 'online',
        }
      } else {
        roomInfo.value.members[index] = {
          user_id: member.user_id,
          nickname: member.nickname ?? '',
          balance: member.balance ?? 0,
          status: member.status ?? 'online',
        }
      }
    } else {
      roomInfo.value.members.push({
        ...member,
        nickname: member.nickname ?? '',
        balance: member.balance ?? 0,
        status: member.status ?? 'online',
      })
    }
  }

  // 移除房间成员
  function removeMember(userId: number) {
    if (!roomInfo.value) return
    roomInfo.value.members = roomInfo.value.members.filter((m) => m.user_id !== userId)
  }

  // 添加操作记录
  function parseNiuniuBetDescription(description?: string): NiuniuBetDetail[] | null {
    if (!description) {
      return null
    }

    try {
      const parsed = JSON.parse(description)
      if (!Array.isArray(parsed)) {
        return null
      }

      const bets: NiuniuBetDetail[] = []
      parsed.forEach((item: any) => {
        const toUserId = Number(item?.to_user_id)
        const amount = Number(item?.amount)

        if (!Number.isFinite(toUserId) || !Number.isFinite(amount)) {
          return
        }

        const betDetail: NiuniuBetDetail = {
          to_user_id: toUserId,
          amount,
        }

        if (typeof item?.to_nickname === 'string' && item.to_nickname.trim().length > 0) {
          betDetail.to_nickname = item.to_nickname
        }

        bets.push(betDetail)
      })

      return bets
    } catch (error) {
      console.error('解析牛牛下注描述失败:', error, description)
      return null
    }
  }

  function parseSettlementDescription(description?: string): SettlementSummary | null {
    if (!description) {
      return null
    }

    let parsed: unknown
    try {
      parsed = JSON.parse(description)
    } catch (error) {
      return null
    }

    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return null
    }

    const record = parsed as {
      batch?: unknown
      chip_rate?: unknown
      settled_at?: unknown
      details?: unknown
    }

    const summary: SettlementSummary = {
      details: [],
    }

    if (typeof record.batch === 'string' && record.batch.trim().length > 0) {
      summary.batch = record.batch
    }

    if (typeof record.chip_rate === 'string' && record.chip_rate.trim().length > 0) {
      summary.chip_rate = record.chip_rate
    }

    if (typeof record.settled_at === 'string' && record.settled_at.trim().length > 0) {
      summary.settled_at = record.settled_at
    }

    if (Array.isArray(record.details)) {
      record.details.forEach((item) => {
        const detail = item as {
          user_id?: unknown
          nickname?: unknown
          chip_amount?: unknown
          rmb_amount?: unknown
        }

        const userId = Number(detail.user_id)
        const chipAmount = Number(detail.chip_amount)

        if (!Number.isFinite(userId) || !Number.isFinite(chipAmount)) {
          return
        }

        const settlementDetail: SettlementDetail = {
          user_id: userId,
          chip_amount: chipAmount,
        }

        if (typeof detail.nickname === 'string' && detail.nickname.trim().length > 0) {
          settlementDetail.nickname = detail.nickname
        }

        const rmbAmount = Number(detail.rmb_amount)
        if (Number.isFinite(rmbAmount)) {
          settlementDetail.rmb_amount = rmbAmount
        }

        summary.details.push(settlementDetail)
      })
    }

    if (!summary.batch && !summary.chip_rate && !summary.settled_at && summary.details.length === 0) {
      return null
    }

    return summary
  }

  function normalizeOperation(operation: RoomOperation): RoomOperation {
    let normalized = operation

    if (operation.operation_type === 'niuniu_bet') {
      if (!operation.bets || operation.bets.length === 0) {
        const bets = parseNiuniuBetDescription(operation.description)
        if (bets) {
          normalized = {
            ...normalized,
            bets,
          }
        }
      }
    }

    if (operation.operation_type === 'settlement_confirmed') {
      const summary = parseSettlementDescription(operation.description)
      if (summary) {
        normalized = {
          ...normalized,
          settlement_summary: summary,
        }
      }
    }

    return normalized
  }

  function addOperation(operation: RoomOperation) {
    operations.value.unshift(normalizeOperation(operation))
  }

  // 设置操作记录
  function setOperations(ops: RoomOperation[]) {
    operations.value = ops.map((op) => normalizeOperation(op))
  }

  // 更新我的积分
  function updateMyBalance(balance: number) {
    if (roomInfo.value) {
      roomInfo.value.my_balance = balance
    }
  }

  // 更新桌面积分
  function updateTableBalance(balance: number) {
    if (roomInfo.value) {
      roomInfo.value.table_balance = balance
    }
  }

  function resolveBackendOrigin() {
    const overrideOrigin = (import.meta.env.VITE_BACKEND_ORIGIN as string | undefined)?.trim()
    if (overrideOrigin) {
      return overrideOrigin
    }

    const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
    const hostname = window.location.hostname || 'localhost'
    const defaultPort = '8080'

    if (import.meta.env.DEV || window.location.port === '5173') {
      return `${protocol}//${hostname}:${defaultPort}`
    }

    const rawApiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim()
    if (rawApiBase) {
      try {
        const resolved = new URL(rawApiBase, window.location.origin)
        return resolved.origin
      } catch (error) {
        console.warn('无法解析 VITE_API_BASE_URL，已回退到当前站点', error)
      }
    }

    return window.location.origin
  }

  // 连接WebSocket
  function connectWebSocket(roomId: number) {
    if (ws.value) {
      const state = ws.value.readyState
      if (currentRoomId.value === roomId && (state === WebSocket.OPEN || state === WebSocket.CONNECTING)) {
        return
      }

      ws.value.close()
      ws.value = null
    }

    const baseOrigin = resolveBackendOrigin()

    const wsUrl = new URL(`/api/ws/room/${roomId}`, baseOrigin)
    wsUrl.protocol = wsUrl.protocol === 'https:' ? 'wss:' : 'ws:'
    
    // 添加session_id到查询参数（用于移动端浏览器）
    const sessionId = localStorage.getItem('session_id')
    if (sessionId) {
      wsUrl.searchParams.set('session_id', sessionId)
    }

    ws.value = new WebSocket(wsUrl.toString())
    currentRoomId.value = roomId

    ws.value.onopen = () => {
      console.log('WebSocket连接成功')
    }

    ws.value.onmessage = (event) => {
      const payloads = String(event.data).split('\n').filter((item) => item.trim().length > 0)

      payloads.forEach((payload) => {
        try {
          const message = JSON.parse(payload)
          handleWebSocketMessage(message)
        } catch (error) {
          console.error('解析WebSocket消息失败:', error, payload)
        }
      })
    }

    ws.value.onerror = (error) => {
      console.error('WebSocket错误:', error)
    }

    ws.value.onclose = () => {
      console.log('WebSocket连接关闭')
      ws.value = null
      currentRoomId.value = null
    }
  }

  // 断开WebSocket
  function disconnectWebSocket() {
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    currentRoomId.value = null
  }

  // 处理WebSocket消息
  function handleWebSocketMessage(message: any) {
    switch (message.type) {
      case 'user_joined':
        // 用户加入
        updateMember({
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          balance: message.data.balance ?? 0,
          status: message.data.status ?? 'online',
        })
        addOperation({
          id: Date.now(),
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          operation_type: 'join',
          description: '加入了房间',
          created_at: message.data.joined_at,
        })
        break

      case 'user_returned': {
        const balanceValue =
          typeof message.data.balance === 'number' ? message.data.balance : undefined
        const status =
          typeof message.data.status === 'string' && message.data.status.trim().length > 0
            ? message.data.status
            : 'online'

        updateMember({
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          status,
          ...(balanceValue !== undefined ? { balance: balanceValue } : {}),
        })

        if (message.data.user_id === userStore.user?.id && balanceValue !== undefined) {
          updateMyBalance(balanceValue)
        }

        addOperation({
          id: Date.now(),
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          operation_type: 'return',
          description: '返回了房间',
          created_at: message.data.returned_at ?? new Date().toISOString(),
        })
        break
      }

      case 'user_left': {
        // 用户离开通知（保留成员身份）
        const status = typeof message.data.status === 'string' ? message.data.status : 'offline'
        updateMember({
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          status,
        })
        addOperation({
          id: Date.now(),
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          operation_type: 'leave',
          description: '离开了房间',
          created_at: message.data.left_at,
        })
        break
      }

      case 'user_kicked':
        // 用户被踢出
        updateMember({
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          status: typeof message.data.status === 'string' ? message.data.status : 'offline',
        })
        addOperation({
          id: Date.now(),
          user_id: message.data.kicked_by,
          nickname: message.data.kicked_by_nickname,
          operation_type: 'kick',
          description: `踢出了${message.data.nickname}`,
          target_user_id: message.data.user_id,
          target_nickname: message.data.nickname,
          created_at: message.data.kicked_at ?? new Date().toISOString(),
        })
        if (message.data.user_id === userStore.user?.id) {
          antdMessage.warning('您已被移出房间')
          clearRoomInfo()
          router.push('/')
        }
        break

      case 'bet': {
        // 下注
        if (typeof message.data.table_balance === 'number') {
          updateTableBalance(message.data.table_balance)
        }
        updateMember({
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          balance: message.data.balance ?? 0,
          status:
            roomInfo.value?.members.find((m) => m.user_id === message.data.user_id)?.status ??
            'online',
        })
        if (
          message.data.user_id === userStore.user?.id &&
          typeof message.data.balance === 'number'
        ) {
          updateMyBalance(message.data.balance)
        }
        addOperation({
          id: Date.now(),
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          operation_type: 'bet',
          amount: message.data.amount,
          description: `下注了${message.data.amount}积分`,
          created_at: message.data.created_at,
        })
        break
      }

      case 'niuniu_bet': {
        if (typeof message.data.table_balance === 'number') {
          updateTableBalance(message.data.table_balance)
        }

        const betDetails: NiuniuBetDetail[] = Array.isArray(message.data.bets)
          ? (message.data.bets as unknown[])
              .map((item): NiuniuBetDetail | null => {
                const record = item as {
                  to_user_id?: unknown
                  amount?: unknown
                  to_nickname?: unknown
                }

                const toUserId = Number(record?.to_user_id)
                const amount = Number(record?.amount)

                if (!Number.isFinite(toUserId) || !Number.isFinite(amount)) {
                  return null
                }

                const detail: NiuniuBetDetail = {
                  to_user_id: toUserId,
                  amount,
                }

                if (
                  typeof record?.to_nickname === 'string' &&
                  record.to_nickname.trim().length > 0
                ) {
                  detail.to_nickname = record.to_nickname
                }

                return detail
              })
              .filter((detail): detail is NiuniuBetDetail => detail !== null)
          : []

        updateMember({
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          balance: message.data.balance ?? 0,
          status:
            roomInfo.value?.members.find((m) => m.user_id === message.data.user_id)?.status ??
            'online',
        })

        if (
          message.data.user_id === userStore.user?.id &&
          typeof message.data.balance === 'number'
        ) {
          updateMyBalance(message.data.balance)
        }

        const totalAmount = Number(message.data.total_amount)

        addOperation({
          id: Date.now(),
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          operation_type: 'niuniu_bet',
          amount: Number.isFinite(totalAmount) ? totalAmount : undefined,
          description: JSON.stringify(betDetails),
          bets: betDetails,
          created_at: message.data.created_at,
        })

        break
      }

      case 'withdraw': {
        // 收回
        updateTableBalance(message.data.table_balance)
        updateMember({
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          balance: message.data.balance ?? 0,
          status:
            roomInfo.value?.members.find((m) => m.user_id === message.data.user_id)?.status ??
            'online',
        })
        addOperation({
          id: Date.now(),
          user_id: message.data.user_id,
          nickname: message.data.nickname,
          operation_type: 'withdraw',
          amount: message.data.amount,
          description: `收回了${message.data.amount}积分`,
          created_at: message.data.created_at,
        })
        break
      }

      case 'force_transfer': {
        if (typeof message.data.table_balance === 'number') {
          updateTableBalance(message.data.table_balance)
        }

        const actorId = Number(message.data.user_id)
        const actorNickname =
          typeof message.data.nickname === 'string' ? message.data.nickname : ''
        const actorBalanceValue = Number(message.data.actor_balance)
        const actorExisting =
          Number.isFinite(actorId) && actorId > 0
            ? roomInfo.value?.members.find((m) => m.user_id === actorId)
            : undefined

        if (Number.isFinite(actorId) && actorId > 0) {
          updateMember({
            user_id: actorId,
            nickname: actorNickname || actorExisting?.nickname || '',
            ...(Number.isFinite(actorBalanceValue) ? { balance: actorBalanceValue } : {}),
            status: actorExisting?.status ?? 'online',
          })

          if (actorId === userStore.user?.id && Number.isFinite(actorBalanceValue)) {
            updateMyBalance(actorBalanceValue)
          }
        }

        const targetId = Number(message.data.target_user_id)
        const targetNickname =
          typeof message.data.target_nickname === 'string' ? message.data.target_nickname : ''
        const targetBalanceValue = Number(message.data.target_balance)
        const targetExisting =
          Number.isFinite(targetId) && targetId > 0
            ? roomInfo.value?.members.find((m) => m.user_id === targetId)
            : undefined

        if (Number.isFinite(targetId) && targetId > 0) {
          updateMember({
            user_id: targetId,
            nickname: targetNickname || targetExisting?.nickname || '',
            ...(Number.isFinite(targetBalanceValue) ? { balance: targetBalanceValue } : {}),
            status: targetExisting?.status ?? 'online',
          })

          if (targetId === userStore.user?.id && Number.isFinite(targetBalanceValue)) {
            updateMyBalance(targetBalanceValue)
          }
        }

        const amountValue = Number(message.data.amount)

        addOperation({
          id: Date.now(),
          user_id: Number.isFinite(actorId) && actorId > 0 ? actorId : 0,
          nickname: actorNickname || actorExisting?.nickname || '',
          operation_type: 'force_transfer',
          amount: Number.isFinite(amountValue) ? amountValue : undefined,
          description: '执行了积分强制转移',
          target_user_id: Number.isFinite(targetId) && targetId > 0 ? targetId : undefined,
          target_nickname: targetNickname || targetExisting?.nickname || undefined,
          created_at: message.data.created_at ?? new Date().toISOString(),
        })
        break
      }

      case 'settlement_initiated':
        // 发起结算
        settlementContext.value = {
          initiated_by: message.data.initiated_by,
          initiated_by_nickname: message.data.initiated_by_nickname,
          initiated_at: message.data.initiated_at ?? new Date().toISOString(),
          settlement_plan: message.data.settlement_plan ?? [],
          table_balance: message.data.table_balance ?? roomInfo.value?.table_balance ?? 0,
          confirmed: false,
        }
        addOperation({
          id: Date.now(),
          user_id: message.data.initiated_by,
          nickname: message.data.initiated_by_nickname,
          operation_type: 'settlement_initiated',
          description: '发起了结算',
          created_at: message.data.initiated_at ?? new Date().toISOString(),
        })
        break

      case 'settlement_confirmed': {
        // 确认结算
        const settledAt = message.data.settled_at ?? new Date().toISOString()
        const currentContext = settlementContext.value

        let description = '确认了结算'
        let settlementSummary: SettlementSummary | undefined

        const summaryPayload = message.data.settlement_summary
        if (summaryPayload) {
          try {
            description = JSON.stringify(summaryPayload)
          } catch (error) {
            console.warn('序列化实时结算摘要失败:', error, summaryPayload)
          }

          const parsedSummary = parseSettlementDescription(description)
          if (parsedSummary) {
            settlementSummary = parsedSummary
          }
        }

        settlementContext.value = {
          initiated_by: currentContext?.initiated_by ?? message.data.confirmed_by,
          initiated_by_nickname:
            currentContext?.initiated_by_nickname ?? message.data.confirmed_by_nickname ?? '',
          initiated_at: currentContext?.initiated_at ?? settledAt,
          settlement_plan: currentContext?.settlement_plan ?? [],
          table_balance: currentContext?.table_balance ?? roomInfo.value?.table_balance ?? 0,
          confirmed: true,
          confirmed_by: message.data.confirmed_by,
          confirmed_by_nickname: message.data.confirmed_by_nickname,
          confirmed_at: settledAt,
        }
        addOperation({
          id: Date.now(),
          user_id: message.data.confirmed_by,
          nickname: message.data.confirmed_by_nickname,
          operation_type: 'settlement_confirmed',
          description,
          settlement_summary: settlementSummary,
          created_at: message.data.settled_at ?? new Date().toISOString(),
        })

        if (roomInfo.value) {
          roomInfo.value.my_balance = 0
          roomInfo.value.table_balance = 0
          roomInfo.value.members.forEach((m) => {
            m.balance = 0
          })
        }
        break
      }

      case 'room_dissolved': {
        const dissolvedAt =
          typeof message.data?.dissolved_at === 'string'
            ? message.data.dissolved_at
            : new Date().toISOString()

        addOperation({
          id: Date.now(),
          user_id: 0,
          nickname: '',
          operation_type: 'room_dissolved',
          description: '房间已解散',
          created_at: dissolvedAt,
        })

        antdMessage.info('房间已解散')
        clearRoomInfo()
        router.push('/')
        break
      }
    }
  }

  // 清空房间信息
  function clearRoomInfo() {
    roomInfo.value = null
    operations.value = []
    disconnectWebSocket()
    settlementContext.value = null
  }

  function setSettlementContext(context: SettlementContext | null) {
    settlementContext.value = context
  }

  return {
    roomInfo,
    operations,
    settlementContext,
    setRoomInfo,
    updateMember,
    removeMember,
    addOperation,
    setOperations,
    updateMyBalance,
    updateTableBalance,
    connectWebSocket,
    disconnectWebSocket,
    clearRoomInfo,
    setSettlementContext,
  }
})

