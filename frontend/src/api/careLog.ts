import request from '@/utils/request'
import type { CareLog, CareLogListData, CareLogType } from '@/types/api'

export function listCareLogs(params?: { garden_id?: number; type?: CareLogType }) {
  return request.get<never, CareLogListData>('/care-logs', { params })
}

export function saveCareLog(payload: {
  garden_id: number
  log_date: string
  log_type: CareLogType
  note?: string
  image_url?: string
}) {
  return request.post<never, { updated: boolean; log: CareLog }>('/care-logs', payload)
}

export function deleteCareLog(id: number) {
  return request.delete<never, { deleted: boolean }>(`/care-logs/${id}`)
}
