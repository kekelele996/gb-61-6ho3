import type { CareLogType } from '@/types/api'

// Care log activity types, kept in sync with backend constants.CareLog*.
export const CareLogTypeMap: Record<CareLogType, string> = {
  watering: '浇水',
  fertilizing: '施肥',
  pest_control: '用药',
  pruning: '修剪',
  observation: '观察',
}

export const CARE_LOG_TYPES = Object.keys(CareLogTypeMap) as CareLogType[]

export const CareLogTypeTagType: Record<CareLogType, 'primary' | 'success' | 'warning' | 'danger' | 'info'> = {
  watering: 'primary',
  fertilizing: 'success',
  pest_control: 'danger',
  pruning: 'warning',
  observation: 'info',
}
