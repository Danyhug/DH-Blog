<template>
  <div class="settings-modal">
    <div class="modal-content">
      <div class="modal-header">
        <h2 class="modal-title">设置</h2>
        <button class="close-btn" @click="$emit('close')">
          <XIcon class="icon-sm" />
        </button>
      </div>

      <div class="modal-body">
        <div class="section">
          <h3 class="section-title">账户</h3>
          <div class="form-group">
            <label class="form-label">邮箱</label>
            <input type="email" value="user@example.com" readonly disabled class="form-input disabled" />
          </div>
          <div class="form-group">
            <label class="form-label">存储使用情况</label>
            <div class="storage-info">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: '45%' }"></div>
              </div>
              <p class="storage-text">已使用 4.5 GB / 10 GB</p>
            </div>
          </div>
        </div>

        <div class="section border-top">
          <div class="section-header">
            <h3 class="section-title">WebDAV 配置</h3>
            <div class="status-badge connected">
              <div class="status-dot"></div>
              已连接
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">服务器地址</label>
            <input
              type="text"
              v-model="serverUrl"
              class="form-input mono"
            />
          </div>
          <div class="form-group">
            <label class="form-label">用户名</label>
            <input type="text" v-model="username" class="form-input" />
          </div>
          <div class="toggle-group">
            <span class="toggle-label">自动同步</span>
            <label class="toggle-switch">
              <input type="checkbox" v-model="autoSync" />
              <span class="toggle-slider"></span>
            </label>
          </div>
          <button class="btn-primary full-width" @click="reconnect">
            <ServerIcon class="icon-sm" />
            重新连接
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { XIcon, ServerIcon } from './icons'

// 发出事件
const emit = defineEmits(['close'])

// 状态
const serverUrl = ref('https://webdav.example.com/remote.php/dav/files/')
const username = ref('john.doe')
const autoSync = ref(true)

// 方法
function reconnect() {
  // 重新连接逻辑
  alert('正在重新连接到WebDAV服务器...')
  // 在实际应用中，这里应该实现与WebDAV服务器的连接逻辑
}
</script>

<style scoped>
.settings-modal {
  position: absolute;
  top: 4rem;
  right: 4rem;
  width: 400px;
  height: 500px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(24px);
  border-radius: 1rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.2);
  z-index: 20;
}

.modal-content {
  padding: 1.5rem;
  height: 100%;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.modal-title {
  font-size: 1.25rem;
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

.modal-body {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.section.border-top {
  border-top: 1px solid rgba(229, 231, 235, 0.8);
  padding-top: 1.5rem;
}

.section-title {
  font-size: 0.875rem;
  font-weight: 500;
  color: #111827;
  margin: 0;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-badge.connected {
  background: #dcfce7;
  color: #166534;
}

.status-dot {
  width: 0.5rem;
  height: 0.5rem;
  background: #22c55e;
  border-radius: 50%;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.form-label {
  font-size: 0.75rem;
  color: #6b7280;
}

.form-input {
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input.disabled {
  background: #f9fafb;
  color: #6b7280;
  cursor: not-allowed;
}

.form-input.mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.storage-info {
  margin-top: 0.5rem;
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

.storage-text {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
}

.toggle-group {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.toggle-label {
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

.btn-primary {
  background: #2563eb;
  color: white;
  border: none;
  padding: 0.75rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  transition: background-color 0.2s;
  margin-top: 1rem;
}

.btn-primary:hover {
  background: #1d4ed8;
}

.btn-primary.full-width {
  width: 100%;
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}
</style>
