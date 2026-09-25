import request from '@/utils/request'
import type { CareLog, CareLogListData, CareLogType } from '@/types/api'

// Care logs for one plant in the current user's garden. The list endpoint
// always returns the last-7-days per-type counts alongside the (optionally
// type-filtered) logs, newest first.
export function listCareLogs(gardenId: number, type?: CareLogType | '' | undefined) {
  return request.get<never, CareLogListData>(`/gardens/${gardenId}/care-logs`, {
    params: type ? { type } : {},
  })
}

// Creates a log; same plant + day + type updates the existing record instead
// of inserting a second one (backend responds 200 vs 201 accordingly).
export function saveCareLog(
  gardenId: number,
  payload: { log_date: string; log_type: CareLogType; note?: string; images?: string[] },
) {
  return request.post<never, CareLog>(`/gardens/${gardenId}/care-logs`, payload)
}

export function updateCareLog(
  id: number,
  payload: Partial<{ log_date: string; log_type: CareLogType; note: string; images: string[] }>,
) {
  return request.put<never, CareLog>(`/care-logs/${id}`, payload)
}

export function deleteCareLog(id: number) {
  return request.delete<never, { deleted: boolean }>(`/care-logs/${id}`)
}
