<template>
  <div class="upload-modal">
    <div class="modal-content">
      <div class="modal-header">
        <h3 class="modal-title">文件上传</h3>
        <button class="close-btn" @click="$emit('close')">
          <XIcon class="icon-sm" />
        </button>
      </div>

      <!-- 文件拖放区域 -->
      <div 
        class="drop-area" 
        :class="{ active: isDragging }" 
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

      <div v-if="selectedFiles.length > 0" class="upload-list">
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

      <div v-if="selectedFiles.length > 0" class="upload-actions">
        <button class="upload-btn" @click="uploadFiles">
          开始上传
        </button>
      </div>

      <div v-if="isUploading" class="upload-progress-container">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${uploadProgress}%` }"></div>
        </div>
        <p class="progress-text">总进度：{{ uploadProgress }}%</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { XIcon, FileTextIcon, ImageIcon, VideoIcon, MusicIcon, FolderIcon, UploadIcon } from '../utils/icons'

interface Props {
  uploadProgress: number
}

const props = defineProps<Props>()
const emit = defineEmits(['close', 'upload'])

// 文件相关状态
const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFiles = ref<File[]>([])
const isUploading = ref(false)

// 触发文件选择
function triggerFileInput() {
  fileInput.value?.click()
}

// 处理文件选择
function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files) {
    const newFiles = Array.from(input.files)
    selectedFiles.value.push(...newFiles)
  }
}

// 处理文件拖放
function handleFileDrop(event: DragEvent) {
  isDragging.value = false
  if (event.dataTransfer?.files) {
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
  emit('upload', selectedFiles.value)
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
</script>

<style scoped>
.upload-modal {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 450px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(24px);
  border-radius: 1rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.2);
  z-index: 30;
}

.modal-content {
  padding: 1.5rem;
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
  transition: all 0.2s;
}

.drop-area.active {
  border-color: #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.upload-icon {
  width: 2.5rem;
  height: 2.5rem;
  color: #6b7280;
  margin-bottom: 1rem;
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

.upload-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-height: 250px;
  overflow-y: auto;
  margin-bottom: 1rem;
}

.upload-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem;
  border-radius: 0.375rem;
  background-color: #f9fafb;
}

.file-icon-container {
  width: 2.5rem;
  height: 2.5rem;
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

.file-size {
  font-size: 0.75rem;
  color: #6b7280;
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

.icon-sm {
  width: 1rem;
  height: 1rem;
}
</style>
