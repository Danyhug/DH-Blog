<template>
  <div class="share-popup">
    <div class="popup-content">
      <div class="popup-header">
        <h3 class="popup-title">分享链接</h3>
        <button class="close-btn" @click="$emit('close')">
          <XIcon class="icon-sm" />
        </button>
      </div>

      <div class="popup-body">
        <div class="file-preview">
          <div class="file-icon-container">
            <component :is="file.icon || FileTextIcon" class="file-icon" />
          </div>
          <div class="file-info">
            <p class="file-name">{{ file.name }}</p>
            <p class="file-size">{{ file.size }}</p>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">分享链接</label>
          <div class="url-input-group">
            <input
              type="text"
              :value="shareUrl"
              readonly
              class="url-input"
              ref="urlInput"
            />
            <button class="copy-btn" @click="copyUrl">复制</button>
          </div>
        </div>

        <div class="option-row">
          <span class="option-label">允许编辑</span>
          <label class="toggle-switch">
            <input type="checkbox" v-model="allowEditing" />
            <span class="toggle-slider"></span>
          </label>
        </div>

        <div class="option-row">
          <span class="option-label">{{ expiryText }}</span>
          <button class="change-btn" @click="changeExpiry">更改</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { XIcon, FileTextIcon } from '../utils/icons'
import type { FileItem } from '../utils/types/file'
import { getDownloadUrl } from '@/api/file'
import { SERVER_URL } from '@/types/Constant'

// 定义属性
interface Props {
  file: FileItem
}

// 发出事件
const emit = defineEmits(['close'])

const props = defineProps<Props>()

// 状态
const allowEditing = ref(false)
const expiryDays = ref(7)
const urlInput = ref<HTMLInputElement | null>(null)
const copySuccess = ref(false)

// 计算属性
const shareUrl = computed(() => {
  if (props.file && props.file.id) {
    // 返回实际的下载链接
    return `${SERVER_URL}${getDownloadUrl(props.file.id)}`;
  }
  return `${SERVER_URL}/files/download/not-found`;
})

const expiryText = computed(() => {
  return `${expiryDays.value}天后过期`
})

// 方法
function generateRandomId() {
  return Math.random().toString(36).substring(2, 10)
}

function copyUrl() {
  if (urlInput.value) {
    urlInput.value.select()
    try {
      navigator.clipboard.writeText(shareUrl.value).then(() => {
        copySuccess.value = true
        ElMessage.success('链接已复制到剪贴板')
        setTimeout(() => {
          copySuccess.value = false
        }, 2000)
      })
    } catch (err) {
      // 降级方案
    document.execCommand('copy')
      copySuccess.value = true
      ElMessage.success('链接已复制到剪贴板')
      setTimeout(() => {
        copySuccess.value = false
      }, 2000)
    }
  }
}

function changeExpiry() {
  const days = prompt('请输入过期天数:', expiryDays.value.toString())
  if (days !== null) {
    const newDays = parseInt(days)
    if (!isNaN(newDays) && newDays > 0) {
      expiryDays.value = newDays
    }
  }
}
</script>

<style scoped>
.share-popup {
  position: absolute;
  top: 8rem;
  left: 50%;
  transform: translateX(-50%) translateX(8rem);
  width: 320px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(24px);
  border-radius: 1rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.2);
  z-index: 25;
}

.popup-content {
  padding: 1.25rem;
}

.popup-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.popup-title {
  font-size: 1rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.close-btn:hover {
  background: rgba(156, 163, 175, 0.1);
}

.popup-body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.file-preview {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: rgba(249, 250, 251, 0.5);
  border-radius: 0.5rem;
}

.file-icon-container {
  width: 2rem;
  height: 2rem;
  background: #dbeafe;
  border-radius: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-icon {
  width: 1rem;
  height: 1rem;
  color: #2563eb;
}

.file-info {
  flex: 1;
}

.file-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: #111827;
  margin: 0;
}

.file-size {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-size: 0.75rem;
  color: #6b7280;
}

.url-input-group {
  display: flex;
  gap: 0.5rem;
}

.url-input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  background: #f9fafb;
}

.copy-btn {
  background: #2563eb;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.copy-btn:hover {
  background: #1d4ed8;
}

.option-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.option-label {
  font-size: 0.75rem;
  color: #6b7280;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  transition: 0.4s;
  border-radius: 24px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.4s;
  border-radius: 50%;
}

input:checked + .toggle-slider {
  background-color: #3b82f6;
}

input:checked + .toggle-slider:before {
  transform: translateX(20px);
}

.change-btn {
  background: none;
  border: none;
  color: #6b7280;
  font-size: 0.75rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  transition: background-color 0.2s;
}

.change-btn:hover {
  background: rgba(156, 163, 175, 0.1);
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}
</style>
