<template>
  <div class="page">
    <h1>我的花园</h1>
    <el-row :gutter="16">
      <el-col :xs="24" :md="16">
        <el-card>
          <template #header>花园清单</template>
          <el-table :data="gardenItems" empty-text="花园还是空的，去品种库添加吧">
            <el-table-column label="植物">
              <template #default="{ row }">
                <span class="garden-name">{{ plantLabel(row) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="位置" prop="location" />
            <el-table-column label="拥有时间">
              <template #default="{ row }">{{ formatDate(row.owned_since) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <el-button size="small" type="primary" plain @click="openLogs(row)">养护日志</el-button>
                <el-button size="small" type="danger" @click="remove(row.id)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-card class="block">
          <template #header>我的养护提醒</template>
          <ReminderList :reminders="reminders" @done="markDone" @remove="removeReminder" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="8">
        <el-card>
          <template #header>我的收藏</template>
          <el-table :data="favorites" empty-text="暂无收藏">
            <el-table-column prop="target_type" label="类型" width="80">
              <template #default="{ row }">{{ FavoriteTargetTypeMap[row.target_type as FavoriteTargetType] }}</template>
            </el-table-column>
            <el-table-column prop="target_id" label="目标 ID" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <CareLogDrawer v-model="logDrawerVisible" :garden-id="activeGardenId" :plant-label="activePlantLabel" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import ReminderList from '@/components/common/ReminderList.vue'
import CareLogDrawer from '@/components/garden/CareLogDrawer.vue'
import { listGardens, removeGarden } from '@/api/garden'
import { listPlants } from '@/api/plant'
import { listFavorites } from '@/api/favorite'
import { listReminders, deleteReminder, updateReminderStatus } from '@/api/reminder'
import { FavoriteTargetTypeMap, type Favorite, type FavoriteTargetType } from '@/constants/favorite'
import type { PlantSpecies } from '@/constants/plant'
import { formatDate } from '@/utils/dateFormat'
import type { CareReminder, UserGarden } from '@/types/api'

const gardenItems = ref<UserGarden[]>([])
const favorites = ref<Favorite[]>([])
const reminders = ref<CareReminder[]>([])
const plants = ref<PlantSpecies[]>([])

const logDrawerVisible = ref(false)
const activeGardenId = ref(0)
const activePlantLabel = ref('')

onMounted(async () => {
  const [gardens, plantPage, favs, rems] = await Promise.all([
    listGardens(),
    listPlants({ page: 1, page_size: 500 }),
    listFavorites(),
    listReminders(),
  ])
  gardenItems.value = gardens
  plants.value = plantPage.list
  favorites.value = favs
  reminders.value = rems
})

function plantLabel(row: UserGarden): string {
  if (row.nickname) return row.nickname
  return plants.value.find((p) => p.id === row.plant_species_id)?.name || `植物 #${row.plant_species_id}`
}

function openLogs(row: UserGarden) {
  activeGardenId.value = row.id
  activePlantLabel.value = plantLabel(row)
  logDrawerVisible.value = true
}

async function remove(id: number) {
  await removeGarden(id)
  gardenItems.value = await listGardens()
  ElMessage.success('已移除')
}
async function markDone(id: number) {
  await updateReminderStatus(id, 'done')
  reminders.value = await listReminders()
}
async function removeReminder(id: number) {
  await deleteReminder(id)
  reminders.value = await listReminders()
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
.block { margin-top: 16px; }
.garden-name { font-weight: 600; }
</style>
