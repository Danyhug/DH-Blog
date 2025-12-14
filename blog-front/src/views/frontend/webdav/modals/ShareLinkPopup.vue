<template>
  <div class="share-popup">
    <div class="popup-content">
      <div class="popup-header">
        <h3 class="popup-title">创建分享链接</h3>
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

        <!-- 分享设置 -->
        <template v-if="!shareCreated">
          <div class="option-row">
            <span class="option-label">设置密码</span>
            <label class="toggle-switch">
              <input type="checkbox" v-model="usePassword" />
              <span class="toggle-slider"></span>
            </label>
          </div>

          <div v-if="usePassword" class="form-group">
            <input
              type="text"
              v-model="password"
              placeholder="请输入访问密码"
              class="url-input"
              maxlength="32"
            />
          </div>

          <div class="option-row">
            <span class="option-label">过期时间</span>
            <select v-model="expireDays" class="select-input">
              <option :value="0">永不过期</option>
              <option :value="1">1天</option>
              <option :value="7">7天</option>
              <option :value="30">30天</option>
              <option :value="90">90天</option>
            </select>
          </div>

          <div class="option-row">
            <span class="option-label">下载次数限制</span>
            <select v-model="maxDownloadCount" class="select-input">
              <option :value="0">不限制</option>
              <option :value="1">1次</option>
              <option :value="10">10次</option>
              <option :value="50">50次</option>
              <option :value="100">100次</option>
            </select>
          </div>

          <button class="create-btn" @click="createShareLink" :disabled="creating">
            {{ creating ? '创建中...' : '创建分享链接' }}
          </button>
        </template>

        <!-- 分享链接展示 -->
        <template v-else>
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

          <div v-if="shareInfo?.password" class="form-group">
            <label class="form-label">访问密码</label>
            <div class="url-input-group">
              <input
                type="text"
                :value="displayPassword"
                readonly
                class="url-input"
              />
              <button class="copy-btn" @click="copyPassword">复制</button>
            </div>
          </div>

          <div class="share-tips">
            <p v-if="expireDays > 0">
              <span class="tip-icon">⏰</span> {{ expireDays }}天后过期
            </p>
            <p v-else>
              <span class="tip-icon">✨</span> 永不过期
            </p>
            <p v-if="maxDownloadCount > 0">
              <span class="tip-icon">📥</span> 最多下载{{ maxDownloadCount }}次
            </p>
          </div>

          <button class="create-btn secondary" @click="resetShare">
            创建新的分享
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { XIcon, FileTextIcon } from '../utils/icons'
import type { FileItem } from '../utils/types/file'
import { createShare, generateShareLink, type ShareInfo } from '@/api/share'

// 定义属性
interface Props {
  file: FileItem
}

// 发出事件
const emit = defineEmits(['close'])

const props = defineProps<Props>()

// 状态
const usePassword = ref(false)
const password = ref('')
const displayPassword = ref('') // 用于显示的原始密码
const expireDays = ref(7)
const maxDownloadCount = ref(0)
const urlInput = ref<HTMLInputElement | null>(null)
const creating = ref(false)
const shareCreated = ref(false)
const shareInfo = ref<ShareInfo | null>(null)

// 计算属性
const shareUrl = computed(() => {
  if (shareInfo.value) {
    return generateShareLink(shareInfo.value.share_id)
  }
  return ''
})

// 创建分享链接
async function createShareLink() {
  if (!props.file.id) {
    ElMessage.error('文件ID无效')
    return
  }

  if (usePassword.value && !password.value) {
    ElMessage.warning('请输入访问密码')
    return
  }

  try {
    creating.value = true

    const data: any = {
      file_key: props.file.id
    }

    if (usePassword.value && password.value) {
      data.password = password.value
      displayPassword.value = password.value // 保存原始密码用于显示
    }

    if (expireDays.value > 0) {
      data.expire_days = expireDays.value
    }

    if (maxDownloadCount.value > 0) {
      data.max_download_count = maxDownloadCount.value
    }

    shareInfo.value = await createShare(data)
    shareCreated.value = true
    ElMessage.success('分享链接创建成功')
  } catch (err: any) {
    ElMessage.error(err.message || '创建分享链接失败')
  } finally {
    creating.value = false
  }
}

// 复制链接
function copyUrl() {
  if (urlInput.value) {
    urlInput.value.select()
    try {
      navigator.clipboard.writeText(shareUrl.value).then(() => {
        ElMessage.success('链接已复制到剪贴板')
      })
    } catch (err) {
      document.execCommand('copy')
      ElMessage.success('链接已复制到剪贴板')
    }
  }
}

// 复制密码
function copyPassword() {
  try {
    navigator.clipboard.writeText(displayPassword.value).then(() => {
      ElMessage.success('密码已复制到剪贴板')
    })
  } catch (err) {
    ElMessage.error('复制失败')
  }
}

// 重置分享表单
function resetShare() {
  shareCreated.value = false
  shareInfo.value = null
  usePassword.value = false
  password.value = ''
  displayPassword.value = ''
  expireDays.value = 7
  maxDownloadCount.value = 0
}
</script>

<style scoped>
.share-popup {
  position: absolute;
  top: 8rem;
  left: 50%;
  transform: translateX(-50%) translateX(8rem);
  width: 360px;
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
  word-break: break-all;
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
  background: #f9fafb;
}

.url-input:focus {
  outline: none;
  border-color: #2563eb;
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
  white-space: nowrap;
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
  font-size: 0.875rem;
  color: #374151;
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

.select-input {
  padding: 0.375rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background: #f9fafb;
  cursor: pointer;
}

.select-input:focus {
  outline: none;
  border-color: #2563eb;
}

.create-btn {
  width: 100%;
  padding: 0.75rem 1rem;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-top: 0.5rem;
}

.create-btn:hover:not(:disabled) {
  background: #1d4ed8;
}

.create-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.create-btn.secondary {
  background: #6b7280;
}

.create-btn.secondary:hover {
  background: #4b5563;
}

.share-tips {
  padding: 0.75rem;
  background: #f0fdf4;
  border-radius: 0.5rem;
  border: 1px solid #bbf7d0;
}

.share-tips p {
  margin: 0;
  font-size: 0.75rem;
  color: #166534;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.share-tips p + p {
  margin-top: 0.5rem;
}

.tip-icon {
  font-size: 1rem;
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}
</style>
