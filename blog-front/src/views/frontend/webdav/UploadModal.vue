<template>
  <div class="upload-modal">
    <div class="modal-content">
      <div class="modal-header">
        <h3 class="modal-title">文件上传</h3>
        <button class="close-btn" @click="$emit('close')">
          <XIcon class="icon-sm" />
        </button>
      </div>

      <div class="upload-list">
        <div class="upload-item completed">
          <div class="file-icon-container blue">
            <FileTextIcon class="file-icon" />
          </div>
          <div class="file-info">
            <p class="file-name">年度报告.pdf</p>
            <p class="file-size">2.4 MB</p>
          </div>
          <CheckIcon class="check-icon" />
        </div>

        <div class="upload-item uploading">
          <div class="file-icon-container purple">
            <ImageIcon class="file-icon" />
          </div>
          <div class="file-info">
            <p class="file-name">演示幻灯片.pptx</p>
            <div class="progress-container">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: `${uploadProgress}%` }"></div>
              </div>
              <p class="progress-text">{{ uploadProgress }}% • 1.2 MB / 1.8 MB</p>
            </div>
          </div>
        </div>

        <div class="upload-item waiting">
          <div class="file-icon-container gray">
            <VideoIcon class="file-icon" />
          </div>
          <div class="file-info">
            <p class="file-name">演示视频.mp4</p>
            <p class="file-status">等待中...</p>
          </div>
        </div>
      </div>

      <div class="upload-summary">
        <div class="summary-row">
          <span class="summary-label">已上传 2/3 个文件</span>
          <span class="summary-speed">3.6 MB/s</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  uploadProgress: number
}

const props = defineProps<Props>()
const emit = defineEmits(['close'])

import { XIcon, FileTextIcon, ImageIcon, VideoIcon, CheckIcon } from './icons'
</script>

<style scoped>
.upload-modal {
  position: absolute;
  top: 50%;
  left: 4rem;
  transform: translateY(-50%);
  width: 350px;
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

.upload-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.upload-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.upload-item.waiting {
  opacity: 0.5;
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
}

.file-size {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
}

.file-status {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
}

.progress-container {
  margin-top: 0.25rem;
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
}

.check-icon {
  width: 1.25rem;
  height: 1.25rem;
  color: #22c55e;
}

.upload-summary {
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid #f3f4f6;
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.875rem;
}

.summary-label {
  color: #6b7280;
}

.summary-speed {
  color: #2563eb;
  font-weight: 500;
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}
</style>
