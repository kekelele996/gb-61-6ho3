import type { CareLogType } from '@/types/api'

export const CareLogTypeMap: Record<CareLogType, string> = {
  watering: '浇水',
  fertilizing: '施肥',
  medication: '用药',
  pruning: '修剪',
  observation: '观察',
}

export const CARE_LOG_TYPES = Object.keys(CareLogTypeMap) as CareLogType[]

// Element Plus tag type per care log kind, used by logs and stat cards.
export const CareLogTagType: Record<CareLogType, 'primary' | 'success' | 'warning' | 'danger' | 'info'> = {
  watering: 'primary',
  fertilizing: 'success',
  medication: 'danger',
  pruning: 'warning',
  observation: 'info',
}
