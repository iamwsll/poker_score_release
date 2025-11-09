import { get } from './request'

// 获取今晚战绩
export function getTonightRecords(startTime?: string, endTime?: string) {
  const params: any = {}
  if (startTime) params.start_time = startTime
  if (endTime) params.end_time = endTime
  return get('/records/tonight', { params })
}

