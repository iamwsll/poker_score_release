<template>
  <div class="record-container">
    <div class="header">
      <a-button @click="router.push('/')" type="text">
        <template #icon><LeftOutlined /></template>
        返回
      </a-button>
      <h2>今晚战绩</h2>
      <div style="width: 60px"></div>
    </div>

    <div class="content">
      <!-- 时间段选择 -->
      <div class="time-selector">
        <a-range-picker
          v-model:value="timeRange"
          show-time
          format="YYYY-MM-DD HH:mm"
          @change="loadRecords"
          :style="{ width: '100%' }"
        />
        <a-button @click="setDefaultTime" style="margin-top: 12px">
          使用默认时间
        </a-button>
      </div>

      <!-- 当前在的房间 -->
      <div class="current-rooms" v-if="recordData && recordData.current_rooms.length > 0">
        <h3>当前在的房间</h3>
        <div class="room-list">
          <div
            v-for="room in recordData.current_rooms"
            :key="room.room_id"
            class="room-item"
            @click="handleEnterRoom(room)"
          >
            <span>房间号: {{ room.room_code }}</span>
            <span class="room-type">{{ room.room_type === 'texas' ? '德扑' : '牛牛' }}</span>
          </div>
        </div>
      </div>

      <!-- 战绩表格 -->
      <div class="records-section">
        <h3>好友战绩</h3>
        <div class="records-table" v-if="recordData">
          <div class="table-row header-row">
            <div class="cell">昵称</div>
            <div class="cell">积分盈亏</div>
            <div class="cell">人民币盈亏</div>
          </div>
          <div
            v-for="record in sortedRecords"
            :key="record.user_id"
            class="table-row"
            :class="{ highlight: record.is_me }"
          >
            <div class="cell">
              {{ record.nickname }}
              <span v-if="record.is_me" class="me-badge">我</span>
            </div>
            <div class="cell" :class="getBalanceClass(record.total_chip)">
              {{ record.total_chip > 0 ? '+' : '' }}{{ record.total_chip }}
            </div>
            <div class="cell" :class="getBalanceClass(record.total_rmb)">
              {{ record.total_rmb > 0 ? '+' : '' }}¥{{ record.total_rmb.toFixed(2) }}
            </div>
          </div>
          <div class="table-row summary-row">
            <div class="cell">合计</div>
            <div class="cell">-</div>
            <div class="cell">¥{{ totalCheck }}</div>
          </div>
        </div>
        <div v-else class="empty">
          <a-spin />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { LeftOutlined } from '@ant-design/icons-vue'
import * as recordApi from '@/api/record'
import * as roomApi from '@/api/room'
import dayjs, { Dayjs } from 'dayjs'

const router = useRouter()

const timeRange = ref<[Dayjs, Dayjs]>()
const recordData = ref<any>(null)

type CurrentRoom = {
  room_id: number
  room_code: string
  room_type: string
}

// 计算属性：排序后的记录（盈利者在上）
const sortedRecords = computed(() => {
  if (!recordData.value) return []
  return [...recordData.value.friends_records].sort((a, b) => b.total_rmb - a.total_rmb)
})

// 计算属性：总和校验
const totalCheck = computed(() => {
  if (!recordData.value) return '0.00'
  return recordData.value.total_check.toFixed(2)
})

// 获取余额样式类
const getBalanceClass = (value: number) => {
  if (value > 0) return 'positive'
  if (value < 0) return 'negative'
  return ''
}

// 设置默认时间
const setDefaultTime = () => {
  const now = dayjs()
  const hour = now.hour()
  
  let start, end
  if (hour < 7) {
    // 当前时间在7:00am之前，统计昨天7:00am到今天7:00am
    start = now.subtract(1, 'day').hour(7).minute(0).second(0)
    end = now.hour(7).minute(0).second(0)
  } else {
    // 当前时间在7:00am之后，统计今天7:00am到明天7:00am
    start = now.hour(7).minute(0).second(0)
    end = now.add(1, 'day').hour(7).minute(0).second(0)
  }
  
  timeRange.value = [start, end]
  loadRecords()
}

// 加载战绩
const loadRecords = async () => {
  try {
    let startTime, endTime
    if (timeRange.value) {
      startTime = timeRange.value[0].toISOString()
      endTime = timeRange.value[1].toISOString()
    }
    
    const res = await recordApi.getTonightRecords(startTime, endTime)
    recordData.value = res.data
  } catch (error) {
    message.error('加载战绩失败')
  }
}

// 从今晚战绩进入房间
const handleEnterRoom = async (room: CurrentRoom) => {
  try {
    await roomApi.returnToRoom(room.room_id)
  } catch (error) {
    console.warn('记录返回房间失败', error)
  } finally {
    router.push(`/room/${room.room_id}`)
  }
}

onMounted(() => {
  setDefaultTime()
})
</script>

<style scoped>
.record-container {
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

.time-selector {
  background: white;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
}

.current-rooms {
  background: white;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
}

.current-rooms h3 {
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
}

.room-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.room-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f5f5f5;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
}

.room-item:active {
  transform: scale(0.98);
}

.room-type {
  color: #667eea;
  font-weight: 500;
}

.records-section {
  background: white;
  border-radius: 12px;
  padding: 20px;
}

.records-section h3 {
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
}

.records-table {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}

.table-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
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

.summary-row {
  background: #fafafa;
  font-weight: 600;
}

.highlight {
  background: #f0f5ff;
}

.cell {
  padding: 12px;
  font-size: 14px;
  color: #666;
  display: flex;
  align-items: center;
  gap: 8px;
}

.me-badge {
  background: #667eea;
  color: white;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.positive {
  color: #52c41a;
  font-weight: 600;
}

.negative {
  color: #ff4d4f;
  font-weight: 600;
}

.empty {
  text-align: center;
  padding: 40px 0;
  color: #999;
}
</style>

