<template>
  <div class="page">
    <div class="page-head">
    <h1>养护日志</h1>
    <el-button text type="primary" @click="$router.push('/garden')">← 返回我的花园</el-button>
    </div>

    <div class="stats">
      <el-card v-for="t in CARE_LOG_TYPES" :key="t" shadow="never" class="stat-card">
        <el-tag :type="CareLogTagType[t]" size="small" effect="plain">{{ CareLogTypeMap[t] }}</el-tag>
        <div class="stat-num">{{ recent[t] || 0 }}<span class="stat-unit"> 次</span></div>
        <div class="stat-label">近 7 天</div>
      </el-card>
    </div>

    <el-row :gutter="16">
      <el-col :xs="24" :md="9">
        <el-card>
          <template #header>{{ editing ? '编辑日志' : '记录养护' }}</template>
          <el-alert
            v-if="gardens.length === 0"
            type="warning"
            :closable="false"
            title="花园里还没有植物"
            description="请先在「我的花园」中添加植物，再记录养护日志。"
            class="empty-garden"
          />
          <el-form :model="form" label-width="72px" :disabled="gardens.length === 0">
            <el-form-item label="植物" required>
              <el-select v-model="form.garden_id" placeholder="选择花园中的植物" style="width: 100%">
                <el-option
                  v-for="g in gardens"
                  :key="g.id"
                  :label="gardenLabel(g)"
                  :value="g.id"
                />
              </el-select>
            </el-form-item>
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
            <el-form-item label="类型" required>
              <el-radio-group v-model="form.log_type">
                <el-radio-button v-for="t in CARE_LOG_TYPES" :key="t" :value="t">
                  {{ CareLogTypeMap[t] }}
                </el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.note" type="textarea" :rows="3" maxlength="512" show-word-limit
                placeholder="例如：盆土干透后浇透；施缓释肥一小勺……" />
            </el-form-item>
            <el-form-item label="图片">
              <ImageUploader v-model="form.image_url" />
              <el-image
                v-if="form.image_url"
                :src="form.image_url"
                fit="cover"
                class="form-preview"
                :preview-src-list="[form.image_url]"
                preview-teleported
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving" @click="submit">
                {{ editing ? '保存修改' : '保存日志' }}
              </el-button>
              <el-button v-if="editing" @click="resetForm">取消编辑</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="15">
        <el-card>
          <template #header>
            <div class="list-head">
              <span>日志记录</span>
              <div class="filters">
                <el-select v-model="filterGarden" placeholder="全部植物" clearable size="small" style="width: 150px" @change="load">
                  <el-option v-for="g in gardens" :key="g.id" :label="gardenLabel(g)" :value="g.id" />
                </el-select>
                <el-select v-model="filterType" placeholder="全部类型" clearable size="small" style="width: 120px" @change="load">
                  <el-option v-for="t in CARE_LOG_TYPES" :key="t" :label="CareLogTypeMap[t]" :value="t" />
                </el-select>
              </div>
            </div>
          </template>
          <el-table :data="logs" empty-text="还没有养护记录，左边开始记录第一条吧">
            <el-table-column label="日期" width="108">
              <template #default="{ row }">{{ formatDate(row.log_date) }}</template>
            </el-table-column>
            <el-table-column label="类型" width="80">
              <template #default="{ row }">
                <el-tag :type="CareLogTagType[row.log_type as CareLogType]" size="small">
                  {{ CareLogTypeMap[row.log_type as CareLogType] }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="植物" width="140">
              <template #default="{ row }">
                <div class="plant-cell">
                  <span class="plant-name">{{ row.plant_name || `品种#${row.plant_species_id}` }}</span>
                  <span v-if="row.nickname" class="plant-nick">{{ row.nickname }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="备注 / 图片" min-width="180">
              <template #default="{ row }">
                <span class="note-text">{{ row.note || '—' }}</span>
                <el-image
                  v-if="row.image_url"
                  :src="row.image_url"
                  fit="cover"
                  class="row-thumb"
                  :preview-src-list="imagesOf(row)"
                  :initial-index="imageIndex(row)"
                  preview-teleported
                />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="edit(row)">编辑</el-button>
                <el-popconfirm title="确定删除这条日志吗？" @confirm="remove(row.id)">
                  <template #reference>
                    <el-button size="small" link type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import ImageUploader from '@/components/common/ImageUploader.vue'
import { listGardens } from '@/api/garden'
import { listCareLogs, saveCareLog, deleteCareLog } from '@/api/careLog'
import { getPlant } from '@/api/plant'
import { CareLogTypeMap, CareLogTagType, CARE_LOG_TYPES } from '@/constants/careLog'
import { formatDate } from '@/utils/dateFormat'
import type { CareLog, CareLogType, UserGarden } from '@/types/api'

const route = useRoute()
const gardens = ref<UserGarden[]>([])
const plantNames = ref<Record<number, string>>({})
const logs = ref<CareLog[]>([])
const recent = ref<Record<string, number>>({})
const saving = ref(false)
const editing = ref<CareLog | null>(null)
const filterGarden = ref<number | undefined>(undefined)
const filterType = ref<CareLogType | undefined>(undefined)

const today = formatDate(new Date())
const form = reactive({
  garden_id: undefined as number | undefined,
  log_date: today,
  log_type: 'watering' as CareLogType,
  note: '',
  image_url: '',
})

onMounted(async () => {
  gardens.value = await listGardens()
  await loadPlantNames()
  const q = Number(route.query.garden_id)
  if (q && gardens.value.some((g) => g.id === q)) {
    form.garden_id = q
    filterGarden.value = q
  }
  await load()
})

async function loadPlantNames() {
  const ids = [...new Set(gardens.value.map((g) => g.plant_species_id))]
  await Promise.all(
    ids.map(async (id) => {
      if (!plantNames.value[id]) {
        try {
          plantNames.value[id] = (await getPlant(id)).name
        } catch {
          // species may have been removed; fall back to id label
        }
      }
    }),
  )
}

function gardenLabel(g: UserGarden): string {
  const name = plantNames.value[g.plant_species_id] || `品种#${g.plant_species_id}`
  return g.nickname ? `${name}（${g.nickname}）` : name
}

function disableFuture(date: Date): boolean {
  return date.getTime() > Date.now()
}

async function load() {
  const data = await listCareLogs({ garden_id: filterGarden.value, type: filterType.value })
  logs.value = data.list
  recent.value = data.recent_7_days
}

async function submit() {
  if (!form.garden_id) {
    ElMessage.warning('请选择花园中的植物')
    return
  }
  if (!form.log_date) {
    ElMessage.warning('请选择日期')
    return
  }
  if (form.log_date > today) {
    ElMessage.warning('不能记录未来日期的养护')
    return
  }
  saving.value = true
  try {
    const res = await saveCareLog({
      garden_id: form.garden_id,
      log_date: form.log_date,
      log_type: form.log_type,
      note: form.note,
      image_url: form.image_url,
    })
    ElMessage.success(res.updated ? '当天同类型记录已更新' : '养护日志已保存')
    resetForm()
    await load()
  } finally {
    saving.value = false
  }
}

function edit(row: CareLog) {
  editing.value = row
  form.garden_id = row.garden_id
  form.log_date = row.log_date
  form.log_type = row.log_type
  form.note = row.note
  form.image_url = row.image_url
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function resetForm() {
  editing.value = null
  form.garden_id = filterGarden.value
  form.log_date = today
  form.log_type = 'watering'
  form.note = ''
  form.image_url = ''
}

async function remove(id: number) {
  await deleteCareLog(id)
  ElMessage.success('日志已删除')
  if (editing.value?.id === id) resetForm()
  await load()
}

function imagesOf(row: CareLog): string[] {
  const sameDate = logs.value.filter((l) => l.log_date === row.log_date && l.image_url)
  return sameDate.map((l) => l.image_url)
}
function imageIndex(row: CareLog): number {
  return Math.max(0, imagesOf(row).indexOf(row.image_url))
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
.page-head { display: flex; align-items: center; justify-content: space-between; }
.stats { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.stat-card { flex: 1 1 130px; text-align: center; min-width: 130px; }
.stat-num { font-size: 26px; font-weight: 700; color: #3c8d5c; margin-top: 6px; }
.stat-unit { font-size: 13px; font-weight: 400; color: #888; }
.stat-label { font-size: 12px; color: #999; }
.empty-garden { margin-bottom: 8px; }
.form-preview { width: 120px; height: 90px; border-radius: 6px; margin-top: 8px; display: block; }
.list-head { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.filters { display: flex; gap: 8px; }
.plant-cell { display: flex; flex-direction: column; }
.plant-name { font-weight: 600; }
.plant-nick { font-size: 12px; color: #999; }
.note-text { white-space: pre-wrap; word-break: break-all; }
.row-thumb { width: 56px; height: 56px; border-radius: 4px; margin-top: 4px; display: block; }
</style>
