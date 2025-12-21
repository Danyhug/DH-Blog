<template>
  <div class="upload-modal">
    <div class="modal-content">
      <div class="modal-header">
        <h3 class="modal-title">文件上传</h3>
        <button class="close-btn" @click="$emit('close')">
          <XIcon class="icon-sm" />
        </button>
      </div>

      <!-- 文件拖放区域 - 始终显示，但在上传时折叠 -->
      <div 
        class="drop-area" 
        :class="{ 
          active: isDragging,
          collapsed: isUploading || uploadResults.length > 0 
        }" 
        @dragover.prevent="isDragging = true"
        @dragleave.prevent="isDragging = false"
        @drop.prevent="handleFileDrop"
      >
        <UploadIcon class="upload-icon" />
        <p class="drop-text">拖放文件至此处上传，或</p>
        <input 
          type="file" 
          ref="fileInput"
          multiple
          class="file-input" 
          @change="handleFileSelect"
        >
        <button class="select-btn" @click="triggerFileInput">选择文件</button>
      </div>

      <div v-if="selectedFiles.length > 0 && !isUploading" class="upload-list">
        <div 
          v-for="(file, index) in selectedFiles" 
          :key="index"
          class="upload-item"
        >
          <div class="file-icon-container" :class="getFileIconClass(file)">
            <component :is="getFileIcon(file)" class="file-icon" />
          </div>
          <div class="file-info">
            <p class="file-name">{{ file.name }}</p>
            <p class="file-size">{{ formatFileSize(file.size) }}</p>
          </div>
          <button class="remove-btn" @click="removeFile(index)">
            <XIcon class="icon-sm" />
          </button>
        </div>
      </div>

      <!-- 上传状态列表 -->
      <div v-if="isUploading || uploadResults.length > 0" class="upload-status-list">
        <!-- 失败的文件显示在上方 -->
        <div 
          v-for="(result, index) in sortedUploadResults" 
          :key="index"
          class="upload-item"
          :class="{ 
            'upload-success': result.status === 'success', 
            'upload-error': result.status === 'error',
            'upload-pending': result.status === 'pending'
          }"
        >
          <div class="file-icon-container" :class="getFileIconClass(result.file)">
            <component :is="getFileIcon(result.file)" class="file-icon" />
          </div>
          <div class="file-info">
            <p class="file-name">{{ result.file.name }}</p>
            <div class="file-status">
              <p class="file-size">{{ formatFileSize(result.file.size) }}
                <span v-if="result.uploadedChunks !== undefined && result.totalChunks !== undefined" class="chunk-progress">
                  ({{ result.uploadedChunks }}/{{ result.totalChunks }})
                </span>
              </p>
              <span v-if="result.status === 'success'" class="status-badge success">成功</span>
              <span v-else-if="result.status === 'error'" class="status-badge error">失败</span>
              <span v-else-if="result.status === 'pending'" class="status-badge pending">
                <span class="loading-spinner"></span>上传中
              </span>
            </div>
            <!-- 横向进度条 -->
            <div v-if="result.status === 'uploading' || result.status === 'pending'" class="upload-progress-bar">
              <div class="progress-track">
                <div 
                  class="progress-fill" 
                  :style="{
                    width: result.totalChunks && result.uploadedChunks !== undefined 
                      ? `${(result.uploadedChunks / result.totalChunks) * 100}%` 
                      : '0%'
                  }"
                ></div>
              </div>
            </div>
            <p v-if="result.status === 'error'" class="error-message">{{ result.error || '上传失败' }}</p>
          </div>
        </div>
      </div>

      <div v-if="selectedFiles.length > 0 && !isUploading" class="upload-options">
        <label class="resume-upload-option">
          <input 
            type="checkbox" 
            v-model="enableResumeUpload"
            class="checkbox-input"
          />
          <span class="checkbox-label">启用断点续传（网络中断时自动恢复上传）</span>
        </label>
        <div v-if="enableResumeUpload" class="retry-config-option">
          <label class="retry-label">
            重试次数（0表示无限重试）：
            <input 
              type="number" 
              v-model.number="maxRetries"
              min="0" 
              max="100"
              class="retry-input"
              placeholder="0"
            />
          </label>
        </div>
      </div>

      <div v-if="selectedFiles.length > 0 && !isUploading" class="upload-actions">
        <button class="upload-btn" @click="uploadFiles">
          开始上传
        </button>
      </div>

      <div v-if="isUploading" class="upload-progress-container">
    <div class="progress-bar">
      <div class="progress-fill" :style="{ width: `${uploadProgress}%` }"></div>
    </div>
    <p class="progress-text">总进度：{{ uploadProgress }}%</p>
    <p class="progress-stats">
      已完成: {{ getCompletedCount() }}/{{ uploadResults.length }}
      <span v-if="getSuccessCount() > 0" class="success-count">(成功: {{ getSuccessCount() }})</span>
      <span v-if="getErrorCount() > 0" class="error-count">(失败: {{ getErrorCount() }})</span>
    </p>
  </div>

      <div v-if="!isUploading && uploadResults.length > 0" class="upload-complete-actions">
        <p class="upload-summary">
          上传完成: {{ getSuccessCount() }} 成功, {{ getErrorCount() }} 失败
        </p>
        <div class="action-buttons">
          <button v-if="getErrorCount() > 0" class="retry-btn" @click="retryFailedUploads">
            重试失败文件
          </button>
          <button class="upload-more-btn" @click="clearResults">
            继续上传
          </button>
          <button class="close-btn-text" @click="$emit('close')">
            关闭
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { XIcon, FileTextIcon, ImageIcon, VideoIcon, MusicIcon, FolderIcon, UploadIcon } from '../utils/icons'

interface Props {
  uploadProgress: number
}

interface UploadResult {
  file: File
  status: 'pending' | 'success' | 'error' | 'uploading'
  error?: string
  uploadedChunks?: number
  totalChunks?: number
}

const props = defineProps<Props>()
const emit = defineEmits(['close', 'upload', 'retry'])

// 文件相关状态
const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFiles = ref<File[]>([])
const isUploading = ref(false)
const uploadResults = ref<UploadResult[]>([])
const enableResumeUpload = ref(true) // 默认启用断点续传
const maxRetries = ref(0) // 默认无限重试（0表示无限重试）

// 排序上传结果：失败的在上方，成功的在下方
const sortedUploadResults = computed(() => {
  return [...uploadResults.value].sort((a, b) => {
    // 先按状态排序：error > pending > uploading > success
    const statusOrder = { error: 0, pending: 1, uploading: 2, success: 3 };
    return statusOrder[a.status] - statusOrder[b.status];
  });
});

// 触发文件选择
function triggerFileInput() {
  fileInput.value?.click()
}

// 处理文件选择
function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files) {
    // 如果有上一次的上传记录，先清空
    if (uploadResults.value.length > 0) {
      clearResults()
    }
    const newFiles = Array.from(input.files)
    selectedFiles.value.push(...newFiles)
  }
}

// 处理文件拖放
function handleFileDrop(event: DragEvent) {
  isDragging.value = false
  if (event.dataTransfer?.files) {
    // 如果有上一次的上传记录，先清空
    if (uploadResults.value.length > 0) {
      clearResults()
    }
    const newFiles = Array.from(event.dataTransfer.files)
    selectedFiles.value.push(...newFiles)
  }
}

// 移除文件
function removeFile(index: number) {
  selectedFiles.value.splice(index, 1)
}

// 上传文件
function uploadFiles() {
  if (selectedFiles.value.length === 0) return

  isUploading.value = true
  // 初始化上传结果数组
  uploadResults.value = selectedFiles.value.map(file => ({
    file,
    status: 'pending'
  }))
  
  emit('upload', selectedFiles.value)
  // 上传后清空选择的文件列表，因为现在我们有了uploadResults来跟踪
  selectedFiles.value = []
}

// 重试失败的上传
function retryFailedUploads() {
  const failedFiles = uploadResults.value
    .filter(result => result.status === 'error')
    .map(result => result.file)
  
  if (failedFiles.length > 0) {
    // 将失败的文件重新设置为待上传状态
    uploadResults.value = uploadResults.value.filter(result => result.status !== 'error')
    isUploading.value = true
    emit('retry', failedFiles)
  }
}

// 清除结果，准备继续上传
function clearResults() {
  uploadResults.value = []
  isUploading.value = false
  selectedFiles.value = [] // 确保选择文件列表也被清空
}

// 获取已完成的上传数量
function getCompletedCount() {
  return uploadResults.value.filter(result => 
    result.status === 'success' || result.status === 'error'
  ).length
}

// 获取成功的上传数量
function getSuccessCount() {
  return uploadResults.value.filter(result => result.status === 'success').length
}

// 获取失败的上传数量
function getErrorCount() {
  return uploadResults.value.filter(result => result.status === 'error').length
}

// 获取文件图标
function getFileIcon(file: File) {
  const fileType = file.type
  
  if (fileType.startsWith('image/')) {
    return ImageIcon
  } else if (fileType.startsWith('video/')) {
    return VideoIcon
  } else if (fileType.startsWith('audio/')) {
    return MusicIcon
  } else {
    return FileTextIcon
  }
}

// 获取文件图标样式类
function getFileIconClass(file: File) {
  const fileType = file.type
  
  if (fileType.startsWith('image/')) {
    return 'purple'
  } else if (fileType.startsWith('video/')) {
    return 'orange'
  } else if (fileType.startsWith('audio/')) {
    return 'blue'
  } else {
    return 'gray'
  }
}

// 格式化文件大小
function formatFileSize(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`
  return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

// 暴露方法给父组件调用
defineExpose({
  updateFileStatus: (fileIndex: number, status: 'success' | 'error' | 'uploading', message?: string, uploadedChunks?: number, totalChunks?: number) => {
    if (uploadResults.value[fileIndex]) {
      uploadResults.value[fileIndex].status = status
      if (message) {
        uploadResults.value[fileIndex].error = message
      }
      if (uploadedChunks !== undefined) {
        uploadResults.value[fileIndex].uploadedChunks = uploadedChunks
      }
      if (totalChunks !== undefined) {
        uploadResults.value[fileIndex].totalChunks = totalChunks
      }
    }
  },
  uploadResults,
  enableResumeUpload,
  maxRetries
})
</script>

<style scoped>
.upload-modal {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 450px;
  max-width: 95vw;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(24px);
  border-radius: 1rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.2);
  z-index: 1102; /* 确保显示在蒙版之上 */
  pointer-events: auto; /* 确保弹窗可以接收点击事件 */
}

.modal-content {
  padding: 1.5rem;
  max-height: 80vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.modal-title {
  font-size: 1.125rem;
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

.drop-area {
  margin: 1.5rem 0;
  padding: 2rem;
  border: 2px dashed #e5e7eb;
  border-radius: 0.5rem;
  text-align: center;
  transition: all 0.3s;
}

.drop-area.active {
  border-color: #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.drop-area.collapsed {
  padding: 0.75rem;
  margin: 0.75rem 0;
}

.drop-area.collapsed .upload-icon {
  width: 1.5rem;
  height: 1.5rem;
  margin-bottom: 0.5rem;
}

.drop-area.collapsed .drop-text {
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
}

.upload-icon {
  width: 2.5rem;
  height: 2.5rem;
  color: #6b7280;
  margin-bottom: 1rem;
  display: inline-block;
}

.drop-text {
  margin-bottom: 1rem;
  color: #6b7280;
}

.file-input {
  display: none;
}

.select-btn {
  background-color: #3b82f6;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.select-btn:hover {
  background-color: #2563eb;
}

.upload-list,
.upload-status-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-height: 250px;
  overflow-y: auto;
  margin-bottom: 1rem;
  width: 100%;
}

.upload-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem;
  border-radius: 0.375rem;
  background-color: #f9fafb;
  transition: background-color 0.3s ease;
  width: 100%;
  box-sizing: border-box;
}

.upload-item.upload-success {
  background-color: rgba(16, 185, 129, 0.1);
  border-left: 3px solid #10b981;
}

.upload-item.upload-error {
  background-color: rgba(239, 68, 68, 0.1);
  border-left: 3px solid #ef4444;
}

.upload-item.upload-pending {
  background-color: rgba(59, 130, 246, 0.1);
  border-left: 3px solid #3b82f6;
}

.file-icon-container {
  width: 2.5rem;
  height: 2.5rem;
  min-width: 2.5rem;
  border-radius: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-icon-container.blue {
  background: #dbeafe;
}

.file-icon-container.purple {
  background: #e9d5ff;
}

.file-icon-container.orange {
  background: #ffedd5;
}

.file-icon-container.gray {
  background: #f3f4f6;
}

.file-icon {
  width: 1.25rem;
  height: 1.25rem;
}

.file-icon-container.blue .file-icon {
  color: #2563eb;
}

.file-icon-container.purple .file-icon {
  color: #7c3aed;
}

.file-icon-container.orange .file-icon {
  color: #ea580c;
}

.file-icon-container.gray .file-icon {
  color: #6b7280;
}

.file-info {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.file-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: #111827;
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.file-size {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
}

.chunk-progress {
  color: #2a8aff;
  font-weight: 500;
  margin-left: 8px;
}

.upload-progress-bar {
  margin-top: 8px;
  width: 100%;
}

.progress-track {
  height: 4px;
  background-color: #e5e7eb;
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #2563eb);
  border-radius: 2px;
  transition: width 0.3s ease;
  animation: progress-animation 2s ease-in-out infinite;
}

@keyframes progress-animation {
  0% {
    background-position: 0% 50%;
  }
  50% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0% 50%;
  }
}

.progress-fill {
  background-size: 200% 100%;
  animation: progress-animation 2s ease-in-out infinite;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.125rem 0.5rem;
  border-radius: 1rem;
  font-size: 0.75rem;
  font-weight: 500;
}

.status-badge.success {
  background-color: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.status-badge.error {
  background-color: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-badge.pending {
  background-color: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.loading-spinner {
  display: inline-block;
  width: 0.75rem;
  height: 0.75rem;
  border: 2px solid rgba(59, 130, 246, 0.3);
  border-radius: 50%;
  border-top-color: #3b82f6;
  animation: spin 1s linear infinite;
}

.error-message {
  font-size: 0.75rem;
  color: #ef4444;
  margin: 0.25rem 0 0 0;
}

.remove-btn {
  background: none;
  border: none;
  padding: 0.25rem;
  border-radius: 0.25rem;
  cursor: pointer;
  color: #9ca3af;
  transition: color 0.2s;
}

.remove-btn:hover {
  color: #ef4444;
}

.upload-options {
  margin: 1rem 0;
  padding: 0.75rem;
  background-color: #f8fafc;
  border-radius: 0.5rem;
  border: 1px solid #e2e8f0;
}

.resume-upload-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  user-select: none;
}

.retry-config-option {
  margin-top: 0.75rem;
  margin-left: 1.5rem;
}

.retry-label {
  font-size: 0.875rem;
  color: #475569;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.retry-input {
  width: 4rem;
  padding: 0.25rem 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  text-align: center;
}

.retry-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.checkbox-input {
  width: 1rem;
  height: 1rem;
  accent-color: #3b82f6;
  cursor: pointer;
}

.checkbox-label {
  font-size: 0.875rem;
  color: #475569;
  cursor: pointer;
}

.upload-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.upload-btn {
  background-color: #3b82f6;
  color: white;
  border: none;
  padding: 0.5rem 1.5rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.upload-btn:hover {
  background-color: #2563eb;
}

.upload-progress-container {
  margin-top: 1rem;
}

.progress-bar {
  width: 100%;
  height: 0.5rem;
  background: #e5e7eb;
  border-radius: 0.25rem;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: #3b82f6;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
  text-align: center;
}

.progress-stats {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
  text-align: center;
}

.success-count {
  color: #10b981;
}

.error-count {
  color: #ef4444;
}

.upload-complete-actions {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.upload-summary {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

.action-buttons {
  display: flex;
  gap: 0.75rem;
}

.retry-btn,
.upload-more-btn {
  background-color: #3b82f6;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.retry-btn:hover,
.upload-more-btn:hover {
  background-color: #2563eb;
}

.close-btn-text {
  background-color: transparent;
  color: #6b7280;
  border: 1px solid #d1d5db;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.close-btn-text:hover {
  background-color: #f3f4f6;
  color: #4b5563;
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
}
</style>
