export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface UserInfo {
  id: number
  username: string
  email: string
  nickname: string
  avatar: string
  bio: string
  role: 'user' | 'admin'
  created_at: string
}

export interface CareReminder {
  id: number
  user_id: number
  plant_species_id: number
  task_title: string
  remind_date: string
  frequency: string
  status: 'pending' | 'done' | 'overdue'
  created_at: string
}

export interface UserGarden {
  id: number
  user_id: number
  plant_species_id: number
  nickname: string
  owned_since: string
  location: string
  care_reminder_id: number
  created_at: string
}

export type CareLogType = 'watering' | 'fertilizing' | 'medication' | 'pruning' | 'observation'

export interface CareLog {
  id: number
  user_id: number
  garden_id: number
  plant_species_id: number
  plant_name: string
  nickname: string
  location: string
  log_date: string
  log_type: CareLogType
  note: string
  image_url: string
  created_at: string
  updated_at: string
}

export interface CareLogListData {
  list: CareLog[]
  recent_7_days: Record<string, number>
}

export interface DiseasePest {
  id: number
  plant_species_id: number
  name: string
  symptoms: string
  cause: string
  treatment: string
  recommended_medicine: string
  images: string
  keywords: string
  created_at: string
}

export interface Question {
  id: number
  user_id: number
  title: string
  content: string
  images: string
  status: string
  created_at: string
}

export interface Answer {
  id: number
  question_id: number
  user_id: number
  content: string
  is_best: boolean
  like_count: number
  created_at: string
}
