<template>
  <div class="multi-uploader">
    <div v-for="(url, idx) in modelValue" :key="url + idx" class="thumb-wrap">
      <el-image
        :src="url"
        fit="cover"
        class="thumb"
        :preview-src-list="modelValue"
        :initial-index="idx"
        preview-teleported
      >
        <template #placeholder>
          <div class="thumb-loading">加载中…</div>
        </template>
      </el-image>
      <el-button class="remove-btn" size="small" type="danger" circle @click="removeAt(idx)">
        <el-icon><Close /></el-icon>
      </el-button>
    </div>
    <el-upload
      v-if="modelValue.length < max"
      :show-file-list="false"
      :http-request="doUpload"
      accept="image/*"
      :before-upload="beforeUpload"
    >
      <div class="add-box" v-loading="loading">
        <el-icon><Plus /></el-icon>
      </div>
    </el-upload>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Plus, Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import { useAuthStore } from '@/stores/authStore'

const props = withDefaults(defineProps<{ modelValue: string[]; max?: number }>(), { max: 9 })
const emit = defineEmits<{ (e: 'update:modelValue', urls: string[]): void }>()
const loading = ref(false)

function beforeUpload(file: File): boolean {
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.error('图片不能超过 5MB')
    return false
  }
  return true
}

function removeAt(idx: number) {
  const next = [...props.modelValue]
  next.splice(idx, 1)
  emit('update:modelValue', next)
}

async function doUpload(option: { file: File }) {
  const auth = useAuthStore()
  const form = new FormData()
  form.append('file', option.file)
  loading.value = true
  try {
    const res = await axios.post('/api/v1/uploads', form, {
      headers: { Authorization: `Bearer ${auth.token}`, 'Content-Type': 'multipart/form-data' },
    })
    emit('update:modelValue', [...props.modelValue, res.data.data.url])
    ElMessage.success('上传成功')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.multi-uploader { display: flex; flex-wrap: wrap; gap: 10px; }
.thumb-wrap { position: relative; width: 88px; height: 88px; }
.thumb { width: 88px; height: 88px; border-radius: 6px; }
.thumb-loading { display: flex; align-items: center; justify-content: center; width: 100%; height: 100%; background: #f5f7fa; color: #909399; font-size: 12px; }
.remove-btn { position: absolute; top: -8px; right: -8px; width: 22px; height: 22px; min-height: 22px; padding: 0; }
.add-box {
  width: 88px; height: 88px; border: 1px dashed #d9d9d9; border-radius: 6px;
  display: flex; align-items: center; justify-content: center; color: #909399; cursor: pointer;
}
.add-box:hover { border-color: var(--el-color-primary); color: var(--el-color-primary); }
</style>
