<template>
  <div class="mobile-view">
      <div class="mobile-header">
      <div class="mobile-tabs">
        <div class="tab active">
          <HomeIcon class="icon-sm" />
          <span>首页</span>
        </div>
        <div class="tab">
          <StarIcon class="icon-sm" />
          <span>收藏</span>
          </div>
        <div class="tab">
          <CloudIcon class="icon-sm" />
          <span>云盘</span>
        </div>
      </div>
      <div class="mobile-search">
        <SearchIcon class="search-icon" />
        <input type="text" placeholder="搜索文件..." class="search-input" v-model="searchQuery" />
        </div>
      </div>

    <div class="mobile-files">
        <div
        v-for="(file, index) in filteredFiles"
          :key="index"
          class="mobile-file-item"
        @click="handleFileClick(file)"
        >
        <div class="file-icon-container">
          <FolderIcon v-if="file.type === 'folder'" class="folder-icon" />
          <component v-else-if="file.icon" :is="file.icon" class="file-icon" />
          <FileIcon v-else class="file-icon" />
          </div>
        <div class="file-info">
          <div class="file-name-row">
            <p class="file-name">{{ file.name }}</p>
            <button class="more-btn" @click.stop="showOptions(file)">
              <MoreHorizontalIcon class="icon-xs" />
            </button>
          </div>
          <p class="file-details">{{ file.size }}</p>
        </div>
      </div>
    </div>

    <div class="mobile-fab" @click="$emit('upload')">
      <PlusIcon class="icon-sm" />
      </div>

    <div v-if="showOptionsMenu" class="mobile-options-menu">
      <div class="options-header">
        <h3>{{ selectedFile?.name }}</h3>
        <button @click="showOptionsMenu = false">
          <XIcon class="icon-sm" />
          </button>
      </div>
      <div class="options-list">
        <div class="option-item" @click="openFile">
          <FileIcon class="icon-sm" />
          <span>打开</span>
        </div>
        <div class="option-item" @click="shareFile">
          <UploadIcon class="icon-sm" />
          <span>分享</span>
        </div>
        <div class="option-item" @click="downloadFile">
          <UploadIcon class="icon-sm" transform="rotate(180)" />
          <span>下载</span>
        </div>
        <div class="option-item" @click="renameFile">
          <FileTextIcon class="icon-sm" />
          <span>重命名</span>
        </div>
        <div class="option-item danger" @click="deleteFile">
          <XIcon class="icon-sm" />
          <span>删除</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { FileItem } from '../utils/types/file'
import {
  SearchIcon,
  MoreHorizontalIcon,
  FolderIcon,
  FileIcon,
  HomeIcon,
  StarIcon,
  CloudIcon,
  XIcon,
  PlusIcon,
  UploadIcon,
  FileTextIcon
} from '../utils/icons'

interface Props {
  mobileFiles: FileItem[]
}

const props = defineProps<Props>()
const emit = defineEmits(['upload', 'open', 'share', 'download', 'rename', 'delete'])

// 状态
const searchQuery = ref('')
const showOptionsMenu = ref(false)
const selectedFile = ref<FileItem | null>(null)

// 过滤文件
const filteredFiles = computed(() => {
  if (!searchQuery.value) return props.mobileFiles
  
  const query = searchQuery.value.toLowerCase()
  return props.mobileFiles.filter(file => 
    file.name.toLowerCase().includes(query)
  )
})

// 方法
function handleFileClick(file: FileItem) {
  if (file.type === 'folder') {
    emit('open', file)
  } else {
    openFile()
  }
}

function showOptions(file: FileItem) {
  selectedFile.value = file
  showOptionsMenu.value = true
}

function openFile() {
  if (selectedFile.value) {
    emit('open', selectedFile.value)
    showOptionsMenu.value = false
  }
}

function shareFile() {
  if (selectedFile.value) {
    emit('share', selectedFile.value)
    showOptionsMenu.value = false
  }
}

function downloadFile() {
  if (selectedFile.value) {
    emit('download', selectedFile.value)
    showOptionsMenu.value = false
  }
}

function renameFile() {
  if (selectedFile.value) {
    const newName = prompt('请输入新名称:', selectedFile.value.name)
    if (newName && newName !== selectedFile.value.name) {
      emit('rename', selectedFile.value, newName)
    }
    showOptionsMenu.value = false
  }
}

function deleteFile() {
  if (selectedFile.value) {
    if (confirm(`确定要删除 "${selectedFile.value.name}" 吗？`)) {
      emit('delete', selectedFile.value)
    }
    showOptionsMenu.value = false
  }
}
</script>

<style scoped>
.mobile-view {
  width: 100%;
  height: 100vh;
  background: white;
  display: flex;
  flex-direction: column;
  position: relative;
}

.mobile-header {
  padding: 1rem;
  border-bottom: 1px solid #f3f4f6;
}

.mobile-tabs {
  display: flex;
  margin-bottom: 1rem;
}

.tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  padding: 0.5rem;
  color: #6b7280;
  font-size: 0.75rem;
}

.tab.active {
  color: #2563eb;
}

.mobile-search {
  position: relative;
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: #9ca3af;
  width: 1rem;
  height: 1rem;
}

.search-input {
  width: 100%;
  padding: 0.75rem 1rem 0.75rem 2.5rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  font-size: 0.875rem;
}

.mobile-files {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 1rem;
}

.mobile-file-item {
  display: flex;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid #f3f4f6;
}

.file-icon-container {
  margin-right: 1rem;
}

.folder-icon {
  width: 2rem;
  height: 2rem;
  color: #2563eb;
}

.file-icon {
  width: 2rem;
  height: 2rem;
  color: #6b7280;
}

.file-info {
  flex: 1;
}

.file-name-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.file-name {
  font-size: 0.875rem;
  font-weight: 500;
  margin: 0;
  color: #111827;
}

.more-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: #6b7280;
}

.file-details {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
}

.mobile-fab {
  position: fixed;
  right: 1.5rem;
  bottom: 1.5rem;
  width: 3.5rem;
  height: 3.5rem;
  border-radius: 50%;
  background: #2563eb;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  z-index: 10;
}

.mobile-options-menu {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: white;
  border-top-left-radius: 1rem;
  border-top-right-radius: 1rem;
  box-shadow: 0 -4px 6px -1px rgba(0, 0, 0, 0.1);
  z-index: 20;
  padding: 1rem;
}

.options-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.options-header h3 {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
  color: #111827;
}

.options-header button {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
}

.options-list {
  display: flex;
  flex-direction: column;
}

.option-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  cursor: pointer;
}

.option-item:hover {
  background: #f9fafb;
  border-radius: 0.5rem;
}

.option-item.danger {
  color: #ef4444;
}

.icon-sm {
  width: 1.25rem;
  height: 1.25rem;
}

.icon-xs {
  width: 1rem;
  height: 1rem;
}
</style>
