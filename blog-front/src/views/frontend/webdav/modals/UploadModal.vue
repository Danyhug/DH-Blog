<template>
  <div class="fixed top-1/2 left-1/2 z-[1102] w-[450px] max-w-[95vw] bg-[rgba(255,255,255,0.95)] backdrop-blur-[24px] rounded-2xl shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25)] border border-[rgba(255,255,255,0.2)] pointer-events-auto [transform:translate(-50%,-50%)]">
    <div class="p-6 max-h-[80vh] overflow-y-auto">
      <div class="flex items-center justify-between mb-4">
        <h3 class="modal-title text-[1.125rem] text-[#111827] m-0">文件上传</h3>
        <button class="p-2 rounded-md cursor-pointer transition-[background-color] duration-200 ease-[ease] hover:bg-[rgba(156,163,175,0.1)]" @click="$emit('close')">
          <XIcon class="w-4 h-4" />
        </button>
      </div>

      <!-- 文件拖放区域 - 始终显示，但在上传时折叠 -->
      <div
        class="border-2 border-dashed rounded-lg text-center transition-all duration-300 ease-[ease]"
        :class="[
          isDragging ? 'border-[#3b82f6] bg-[rgba(59,130,246,0.05)]' : 'border-[#e5e7eb]',
          isDropAreaCollapsed ? 'p-3 my-3' : 'p-8 my-6'
        ]"
        @dragover.prevent="isDragging = true"
        @dragleave.prevent="isDragging = false"
        @drop.prevent="handleFileDrop"
      >
        <UploadIcon class="text-[#6b7280] inline-block" :class="isDropAreaCollapsed ? 'w-6 h-6 mb-2' : 'w-10 h-10 mb-4'" />
        <p class="text-[#6b7280]" :class="isDropAreaCollapsed ? 'mb-2 text-[0.875rem]' : 'mb-4'">拖放文件至此处上传，或</p>
        <input
          type="file"
          ref="fileInput"
          multiple
          class="hidden"
          @change="handleFileSelect"
        >
        <button class="bg-[#3b82f6] text-white px-4 py-2 rounded-md font-medium cursor-pointer transition-[background-color] duration-200 ease-[ease] hover:bg-[#2563eb]" @click="triggerFileInput">选择文件</button>
      </div>

      <div v-if="selectedFiles.length > 0 && !isUploading" class="flex flex-col gap-3 max-h-[250px] overflow-y-auto mb-4 w-full">
        <div
          v-for="(file, index) in selectedFiles"
          :key="index"
          class="flex items-center gap-3 p-2 rounded-md w-full box-border bg-[#f9fafb] transition-[background-color] duration-300 ease-[ease]"
        >
          <div class="file-icon-container" :class="getFileIconClass(file)">
            <component :is="getFileIcon(file)" class="file-icon" />
          </div>
          <div class="flex-1 min-w-0 overflow-hidden">
            <p class="text-[0.875rem] font-medium text-[#111827] m-0 whitespace-nowrap overflow-hidden text-ellipsis">{{ file.name }}</p>
            <p class="text-[0.75rem] text-[#6b7280] mt-1 mb-0">{{ formatFileSize(file.size) }}</p>
          </div>
          <button class="p-1 rounded cursor-pointer text-[#9ca3af] transition-[color] duration-200 ease-[ease] hover:text-[#ef4444]" @click="removeFile(index)">
            <XIcon class="w-4 h-4" />
          </button>
        </div>
      </div>

      <!-- 上传状态列表 -->
      <div v-if="isUploading || uploadResults.length > 0" class="flex flex-col gap-3 max-h-[250px] overflow-y-auto mb-4 w-full">
        <!-- 失败的文件显示在上方 -->
        <div
          v-for="(result, index) in sortedUploadResults"
          :key="index"
          class="flex items-center gap-3 p-2 rounded-md w-full box-border transition-[background-color] duration-300 ease-[ease]"
          :class="uploadItemStateClass(result.status)"
        >
          <div class="file-icon-container" :class="getFileIconClass(result.file)">
            <component :is="getFileIcon(result.file)" class="file-icon" />
          </div>
          <div class="flex-1 min-w-0 overflow-hidden">
            <p class="text-[0.875rem] font-medium text-[#111827] m-0 whitespace-nowrap overflow-hidden text-ellipsis">{{ result.file.name }}</p>
            <div class="flex items-center justify-between">
              <p class="text-[0.75rem] text-[#6b7280] mt-1 mb-0">{{ formatFileSize(result.file.size) }}
                <span v-if="result.uploadedChunks !== undefined && result.totalChunks !== undefined" class="text-[#2a8aff] font-medium ml-2">
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
            <div v-if="result.status === 'uploading' || result.status === 'pending'" class="mt-2 w-full">
              <div class="h-1 bg-[#e5e7eb] rounded-[2px] overflow-hidden">
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
            <p v-if="result.status === 'error'" class="text-[0.75rem] text-[#ef4444] mt-1 mb-0">{{ result.error || '上传失败' }}</p>
          </div>
        </div>
      </div>

      <div v-if="selectedFiles.length > 0 && !isUploading" class="my-4 p-3 bg-[#f8fafc] rounded-lg border border-[#e2e8f0]">
        <div class="mt-3 ml-6">
          <label class="text-[0.875rem] text-[#475569] flex items-center gap-2">
            重试次数（0表示无限重试）：
            <input
              type="number"
              v-model.number="maxRetries"
              min="0"
              max="100"
              class="w-16 px-2 py-1 border border-[#d1d5db] rounded text-[0.875rem] text-center focus:outline-none focus:border-[#3b82f6] focus:shadow-[0_0_0_2px_rgba(59,130,246,0.1)]"
              placeholder="0"
            />
          </label>
        </div>
      </div>

      <div v-if="selectedFiles.length > 0 && !isUploading" class="flex justify-end mt-4">
        <button class="bg-[#3b82f6] text-white px-6 py-2 rounded-md font-medium cursor-pointer transition-[background-color] duration-200 ease-[ease] hover:bg-[#2563eb]" @click="uploadFiles">
          开始上传
        </button>
      </div>

      <div v-if="isUploading" class="mt-4">
        <div class="w-full h-2 bg-[#e5e7eb] rounded overflow-hidden">
          <div class="progress-fill" :style="{ width: `${uploadProgress}%` }"></div>
        </div>
        <p class="text-[0.75rem] text-[#6b7280] mt-1 mb-0 text-center">总进度：{{ uploadProgress }}%</p>
        <p class="text-[0.75rem] text-[#6b7280] mt-1 mb-0 text-center">
          已完成: {{ getCompletedCount() }}/{{ uploadResults.length }}
          <span v-if="getSuccessCount() > 0" class="text-[#10b981]">(成功: {{ getSuccessCount() }})</span>
          <span v-if="getErrorCount() > 0" class="text-[#ef4444]">(失败: {{ getErrorCount() }})</span>
        </p>
      </div>

      <div v-if="!isUploading && uploadResults.length > 0" class="mt-4 flex flex-col items-center gap-3">
        <p class="text-[0.875rem] text-[#6b7280] m-0">
          上传完成: {{ getSuccessCount() }} 成功, {{ getErrorCount() }} 失败
        </p>
        <div class="flex gap-3">
          <button v-if="getErrorCount() > 0" class="bg-[#3b82f6] text-white px-4 py-2 rounded-md font-medium cursor-pointer transition-[background-color] duration-200 ease-[ease] hover:bg-[#2563eb]" @click="retryFailedUploads">
            重试失败文件
          </button>
          <button class="bg-[#3b82f6] text-white px-4 py-2 rounded-md font-medium cursor-pointer transition-[background-color] duration-200 ease-[ease] hover:bg-[#2563eb]" @click="clearResults">
            继续上传
          </button>
          <button class="bg-transparent text-[#6b7280] border border-[#d1d5db] px-4 py-2 rounded-md font-medium cursor-pointer transition-all duration-200 ease-[ease] hover:bg-[#f3f4f6] hover:text-[#4b5563]" @click="$emit('close')">
            关闭
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { XIcon, FileTextIcon, ImageIcon, VideoIcon, MusicIcon, UploadIcon } from '../utils/icons'

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
const emit = defineEmits(['close', 'upload', 'retry', 'cancel'])

// 文件相关状态
const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFiles = ref<File[]>([])
const isUploading = ref(false)
const uploadResults = ref<UploadResult[]>([])
const maxRetries = ref(0) // 默认无限重试（0表示无限重试）

// 排序上传结果：失败的在上方，成功的在下方
const sortedUploadResults = computed(() => {
  return [...uploadResults.value].sort((a, b) => {
    // 先按状态排序：error > pending > uploading > success
    const statusOrder = { error: 0, pending: 1, uploading: 2, success: 3 };
    return statusOrder[a.status] - statusOrder[b.status];
  });
});

// 上传中/已有结果时拖放区折叠（原来写在模板的 :class 里，三处元素共用）
const isDropAreaCollapsed = computed(() => isUploading.value || uploadResults.value.length > 0)

// 上传项的状态底色 + 左侧色条。返回互斥的完整色值，避免基础态与状态态同属性互相覆盖
function uploadItemStateClass(status: UploadResult['status']) {
  if (status === 'success') return 'bg-[rgba(16,185,129,0.1)] border-l-[3px] border-l-[#10b981]'
  if (status === 'error') return 'bg-[rgba(239,68,68,0.1)] border-l-[3px] border-l-[#ef4444]'
  if (status === 'pending') return 'bg-[rgba(59,130,246,0.1)] border-l-[3px] border-l-[#3b82f6]'
  return 'bg-[#f9fafb]'
}

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

// 移除文件：同时通知父组件清理可能已存在的分片上传会话
function removeFile(index: number) {
  const [removed] = selectedFiles.value.splice(index, 1)
  if (removed) {
    emit('cancel', removed)
  }
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
  maxRetries
})
</script>

<style scoped>
/* 字重：style.less 全局规则是无层级的，压过工具类 */
.modal-title {
  font-weight: 600;
}

/* 组件类：4 种文件类型配色，容器底色与图标颜色靠后代选择器配对 */
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

/* 组件类：状态徽标 3 处复用 */
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

/* 必须与下方 @keyframes 同块：Vue 会给 scoped 里的 keyframes 改名，animation 挪进工具类会失配 */
.loading-spinner {
  display: inline-block;
  width: 0.75rem;
  height: 0.75rem;
  border: 2px solid rgba(59, 130, 246, 0.3);
  border-radius: 50%;
  border-top-color: #3b82f6;
  animation: spin 1s linear infinite;
}

.progress-fill {
  height: 100%;
  background: #3b82f6;
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

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
}
</style>
