<template>
  <el-drawer
    v-model="visible"
    :title="`养护日志 · ${plantLabel}`"
    size="56%"
    :before-close="handleClose"
    destroy-on-close
  >
    <div class="care-logs">
      <el-alert
        title="同一株植物同一天同一类型只保留一条，再次提交会更新原记录；未来日期无法保存。"
        type="info"
        :closable="false"
        show-icon
      />

      <!-- 最近 7 天各类型次数 -->
      <el-card class="block" shadow="never">
        <template #header>最近 7 天活动</template>
        <el-row :gutter="12">
          <el-col v-for="item in weeklyStat" :key="item.log_type" :xs="12" :sm="8" :md="24 / 5">
            <div class="stat-box" :class="`stat-${item.log_type}`">
              <div class="stat-num">{{ item.count }}</div>
              <div class="stat-label">
                <el-tag :type="tagType(item.log_type)" size="small">{{ typeText(item.log_type) }}</el-tag>
              </div>
            </div>
          </el-col>
        </el-row>
      </el-card>

      <!-- 新增 / 更新当日记录 -->
      <el-card class="block" shadow="never">
        <template #header>{{ editing ? '编辑日志' : '记一条' }}</template>
        <el-form :model="form" label-width="72px" @submit.prevent>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="10">
              <el-form-item label="日期" required>
                <el-date-picker
                  v-model="form.log_date"
                  type="date"
                  value-format="YYYY-MM-DD"
                  :disabled-date="disableFuture"
                  :clearable="false"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="10">
              <el-form-item label="类型" required>
                <el-select v-model="form.log_type" style="width: 100%">
                  <el-option v-for="t in CARE_LOG_TYPES" :key="t" :label="typeText(t)" :value="t" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="备注">
            <el-input v-model="form.note" type="textarea" :rows="3" maxlength="2000" show-word-limit
              placeholder="例如：浇透、发现少量蚜虫、修剪残花……" />
          </el-form-item>
          <el-form-item label="图片">
            <MultiImageUploader v-model="form.images" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="submit">
              {{ editing ? '保存修改' : '保存日志' }}
            </el-button>
            <el-button v-if="editing" @click="resetForm">取消编辑</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 历史列表：按类型筛选，日期从近到远 -->
      <el-card class="block" shadow="never">
        <template #header>
          <div class="list-header">
            <span>历史记录</span>
            <el-radio-group v-model="filterType" size="small" @change="load">
              <el-radio-button label="">全部</el-radio-button>
              <el-radio-button v-for="t in CARE_LOG_TYPES" :key="t" :label="t">{{ typeText(t) }}</el-radio-button>
            </el-radio-group>
          </div>
        </template>
        <el-empty v-if="!logs.length" description="还没有日志，先记一条吧" :image-size="80" />
        <el-timeline v-else>
          <el-timeline-item
            v-for="log in logs"
            :key="log.id"
            :type="tagType(log.log_type)"
            :timestamp="formatDate(log.log_date)"
            placement="top"
          >
            <el-card shadow="hover" class="log-card">
              <div class="log-head">
                <el-tag :type="tagType(log.log_type)" size="small">{{ typeText(log.log_type) }}</el-tag>
                <span class="log-actions">
                  <el-button link type="primary" size="small" @click="startEdit(log)">编辑</el-button>
                  <el-popconfirm title="确定删除这条日志？" @confirm="remove(log.id)">
                    <template #reference>
                      <el-button link type="danger" size="small">删除</el-button>
                    </template>
                  </el-popconfirm>
                </span>
              </div>
              <p v-if="log.note" class="log-note">{{ log.note }}</p>
              <div v-if="log.images?.length" class="log-images">
                <el-image
                  v-for="(img, i) in log.images"
                  :key="i"
                  :src="img"
                  fit="cover"
                  class="log-img"
                  :preview-src-list="log.images"
                  :initial-index="i"
                  preview-teleported
                />
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </el-card>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import MultiImageUploader from '@/components/common/MultiImageUploader.vue'
import { listCareLogs, saveCareLog, updateCareLog, deleteCareLog } from '@/api/careLog'
import { CareLogTypeMap, CareLogTypeTagType, CARE_LOG_TYPES } from '@/constants/careLog'
import { formatDate } from '@/utils/dateFormat'
import type { CareLog, CareLogType, CareLogTypeCount } from '@/types/api'

const props = defineProps<{ modelValue: boolean; gardenId: number; plantLabel: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})
const logs = ref<CareLog[]>([])
const weeklyStat = ref<CareLogTypeCount[]>([])
const filterType = ref<'' | CareLogType>('')
const saving = ref(false)
const editing = ref<CareLog | null>(null)

// (Re)load whenever the drawer opens for a plant.
watch(
  () => [props.modelValue, props.gardenId] as const,
  ([open]) => {
    if (open) {
      resetForm()
      filterType.value = ''
      load()
    }
  },
)

function today(): string {
  return formatDate(new Date())
}

const form = reactive<{ log_date: string; log_type: CareLogType; note: string; images: string[] }>({
  log_date: today(),
  log_type: 'watering',
  note: '',
  images: [],
})

function typeText(t: CareLogType): string {
  return CareLogTypeMap[t] || t
}
function tagType(t: CareLogType): 'primary' | 'success' | 'warning' | 'danger' | 'info' {
  return CareLogTypeTagType[t] || 'info'
}
function disableFuture(d: Date): boolean {
  return d.getTime() > Date.now()
}

function resetForm() {
  editing.value = null
  form.log_date = today()
  form.log_type = 'watering'
  form.note = ''
  form.images = []
}

function startEdit(log: CareLog) {
  editing.value = log
  form.log_date = formatDate(log.log_date)
  form.log_type = log.log_type
  form.note = log.note || ''
  form.images = log.images ? [...log.images] : []
}

async function load() {
  const data = await listCareLogs(props.gardenId, filterType.value || undefined)
  logs.value = data.list
  weeklyStat.value = data.weekly_stat
}

async function submit() {
  if (!form.log_date) {
    ElMessage.warning('请选择日期')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateCareLog(editing.value.id, {
        log_date: form.log_date,
        log_type: form.log_type,
        note: form.note,
        images: form.images,
      })
      ElMessage.success('养护日志已更新')
    } else {
      await saveCareLog(props.gardenId, {
        log_date: form.log_date,
        log_type: form.log_type,
        note: form.note,
        images: form.images,
      })
      ElMessage.success('养护日志已保存')
    }
    resetForm()
    await load()
  } finally {
    saving.value = false
  }
}

async function remove(id: number) {
  await deleteCareLog(id)
  ElMessage.success('已删除')
  if (editing.value?.id === id) {
    resetForm()
  }
  await load()
}

function handleClose(done: () => void) {
  resetForm()
  filterType.value = ''
  emit('update:modelValue', false)
  done()
}

defineExpose({ load })
</script>

<style scoped>
.care-logs { display: flex; flex-direction: column; gap: 4px; }
.block { margin-top: 14px; }
.stat-box { text-align: center; padding: 12px 8px; border-radius: 8px; background: #f5f7fa; margin-bottom: 8px; }
.stat-num { font-size: 26px; font-weight: 700; line-height: 1.2; }
.stat-label { margin-top: 4px; }
.stat-watering .stat-num { color: var(--el-color-primary); }
.stat-fertilizing .stat-num { color: var(--el-color-success); }
.stat-pest_control .stat-num { color: var(--el-color-danger); }
.stat-pruning .stat-num { color: var(--el-color-warning); }
.stat-observation .stat-num { color: var(--el-color-info); }
.list-header { display: flex; align-items: center; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
.log-card :deep(.el-card__body) { padding: 10px 14px; }
.log-head { display: flex; align-items: center; justify-content: space-between; }
.log-note { margin: 8px 0 0; white-space: pre-wrap; line-height: 1.6; }
.log-images { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
.log-img { width: 72px; height: 72px; border-radius: 6px; cursor: pointer; }
</style>
