<template>
  <div class="web-drive-container">
    <div class="browser-header">
      <div class="header-left">
        <div class="breadcrumb">
          <HomeIcon class="icon-sm" />
          <ChevronRightIcon class="icon-xs" />
          <span>{{ currentPath || '我的网盘' }}</span>
        </div>
      </div>
      <div class="header-right">
        <button class="icon-btn" @click="viewMode = 'grid'" :class="{ active: viewMode === 'grid' }">
          <Grid3X3Icon class="icon-sm" />
        </button>
        <button class="icon-btn" @click="viewMode = 'list'" :class="{ active: viewMode === 'list' }">
          <ListIcon class="icon-sm" />
        </button>
        <button class="icon-btn" @click="openSettings">
          <SettingsIcon class="icon-sm" />
        </button>
      </div>
    </div>

    <div class="toolbar">
      <div class="toolbar-left">
        <button class="btn-primary" @click="createNewFolder">
          <PlusIcon class="icon-sm" />
          新建文件夹
        </button>
        <button class="btn-outline" @click="openUploadModal">
          <UploadIcon class="icon-sm" />
          上传
        </button>
      </div>
      <div class="toolbar-right">
        <div class="search-container">
          <SearchIcon class="search-icon" />
          <input type="text" v-model="searchQuery" placeholder="搜索文件..." class="search-input" />
        </div>
      </div>
    </div>

    <div class="file-container">
      <div v-if="viewMode === 'grid'" class="file-grid">
        <div
          v-for="(file, index) in filteredFiles"
          :key="index"
          class="file-item"
          @click="handleFileClick(file)"
          @contextmenu.prevent="showContextMenu($event, file)"
        >
          <div class="file-content">
            <div class="file-icon-container">
              <FolderIcon v-if="file.type === 'folder'" class="folder-icon" />
              <component v-else-if="file.icon" :is="file.icon" class="file-icon" />
              <FileIcon v-else class="file-icon" />
            </div>
            <div class="file-info">
              <p class="file-name">{{ file.name }}</p>
              <p class="file-size">{{ file.size }}</p>
              <p class="file-modified" v-if="file.modified">{{ file.modified }}</p>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="file-list">
        <table>
          <thead>
            <tr>
              <th>名称</th>
              <th>大小</th>
              <th>修改日期</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(file, index) in filteredFiles"
              :key="index"
              @click="handleFileClick(file)"
              @contextmenu.prevent="showContextMenu($event, file)"
            >
              <td>
                <div class="file-row">
                  <FolderIcon v-if="file.type === 'folder'" class="icon-sm folder-icon" />
                  <component v-else-if="file.icon" :is="file.icon" class="icon-sm file-icon" />
                  <FileIcon v-else class="icon-sm file-icon" />
                  <span>{{ file.name }}</span>
                </div>
              </td>
              <td>{{ file.size }}</td>
              <td>{{ file.modified || '-' }}</td>
              <td>
                <button class="icon-btn" @click.stop="shareFile(file)">
                  <UploadIcon class="icon-sm" />
                </button>
                <button class="icon-btn" @click.stop="showContextMenu($event, file)">
                  <MoreHorizontalIcon class="icon-sm" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    
    <!-- 设置弹窗 - 按需显示 -->
    <SettingsModal v-if="showSettingsModal" @close="showSettingsModal = false" />
    
    <!-- 上传弹窗 - 按需显示 -->
    <UploadModal v-if="showUploadModal" :upload-progress="uploadProgress" @close="showUploadModal = false" />
    
    <!-- 分享链接弹窗 - 按需显示 -->
    <ShareLinkPopup v-if="showShareLinkPopup" :file="selectedFile" @close="showShareLinkPopup = false" />
    
    <!-- 右键菜单 -->
    <div v-if="contextMenu.show" class="context-menu" :style="contextMenuStyle">
      <ul>
        <li @click="openFile(contextMenu.file)"><FileIcon class="icon-xs" /> 打开</li>
        <li @click="shareFile(contextMenu.file)"><UploadIcon class="icon-xs" /> 分享</li>
        <li @click="downloadFile(contextMenu.file)"><UploadIcon class="icon-xs" transform="rotate(180)" /> 下载</li>
        <li @click="renameFile(contextMenu.file)"><FileTextIcon class="icon-xs" /> 重命名</li>
        <li @click="deleteFile(contextMenu.file)" class="danger"><XIcon class="icon-xs" /> 删除</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { FileItem } from './types/file'
import SettingsModal from './SettingsModal.vue'
import UploadModal from './UploadModal.vue'
import ShareLinkPopup from './ShareLinkPopup.vue'
import {
  HomeIcon,
  ChevronRightIcon,
  Grid3X3Icon,
  ListIcon,
  SettingsIcon,
  PlusIcon,
  UploadIcon,
  SearchIcon,
  FolderIcon,
  FileIcon,
  FileTextIcon,
  ImageIcon,
  VideoIcon,
  MusicIcon,
  XIcon,
  MoreHorizontalIcon
} from './icons'

// 状态变量
const uploadProgress = ref(65)
const viewMode = ref<'grid' | 'list'>('grid')
const currentPath = ref('')
const searchQuery = ref('')
const showSettingsModal = ref(false)
const showUploadModal = ref(false)
const showShareLinkPopup = ref(false)
const selectedFile = ref<FileItem | null>(null)
const contextMenu = ref({
  show: false,
  x: 0,
  y: 0,
  file: null as FileItem | null
})

// 模拟文件数据
const files = ref<FileItem[]>([
  { name: "文档", type: "folder", size: "12 个项目", modified: "今天" },
  { name: "照片", type: "folder", size: "248 个项目", modified: "昨天" },
  { name: "视频", type: "folder", size: "15 个项目", modified: "2 天前" },
  { name: "音乐", type: "folder", size: "89 个项目", modified: "1 周前" },
  { name: "演示文稿.pptx", type: "file", size: "2.4 MB", modified: "今天", icon: FileTextIcon },
  { name: "假期照片.jpg", type: "image", size: "1.8 MB", modified: "昨天", icon: ImageIcon },
  { name: "会议记录.mp4", type: "video", size: "45.2 MB", modified: "2 天前", icon: VideoIcon },
  { name: "我的音乐.mp3", type: "audio", size: "3.2 MB", modified: "3 天前", icon: MusicIcon },
])

// 过滤文件列表
const filteredFiles = computed(() => {
  if (!searchQuery.value) return files.value
  
  const query = searchQuery.value.toLowerCase()
  return files.value.filter(file => 
    file.name.toLowerCase().includes(query)
  )
})

// 计算右键菜单位置
const contextMenuStyle = computed(() => {
  return {
    top: `${contextMenu.value.y}px`,
    left: `${contextMenu.value.x}px`
  }
})

// 方法
function openSettings() {
  showSettingsModal.value = true
}

function openUploadModal() {
  showUploadModal.value = true
}

function createNewFolder() {
  // 实现创建文件夹逻辑
  const folderName = prompt('请输入文件夹名称:', '新建文件夹')
  if (folderName) {
    files.value.unshift({
      name: folderName,
      type: 'folder',
      size: '0 个项目',
      modified: '刚刚'
    })
  }
}

function handleFileClick(file: FileItem) {
  if (file.type === 'folder') {
    currentPath.value = file.name
    // 在实际应用中，这里应该加载该文件夹的内容
  } else {
    openFile(file)
  }
}

function openFile(file: FileItem) {
  // 根据文件类型打开文件
  console.log('打开文件:', file.name)
}

function shareFile(file: FileItem) {
  selectedFile.value = file
  showShareLinkPopup.value = true
}

function downloadFile(file: FileItem) {
  // 下载文件逻辑
  console.log('下载文件:', file.name)
}

function renameFile(file: FileItem) {
  const newName = prompt('请输入新名称:', file.name)
  if (newName && newName !== file.name) {
    const index = files.value.findIndex(f => f === file)
    if (index !== -1) {
      files.value[index].name = newName
    }
  }
}

function deleteFile(file: FileItem) {
  if (confirm(`确定要删除 "${file.name}" 吗？`)) {
    const index = files.value.findIndex(f => f === file)
    if (index !== -1) {
      files.value.splice(index, 1)
    }
  }
}

function showContextMenu(event: MouseEvent, file: FileItem) {
  contextMenu.value.show = true
  contextMenu.value.x = event.clientX
  contextMenu.value.y = event.clientY
  contextMenu.value.file = file
}

function hideContextMenu() {
  contextMenu.value.show = false
}

// 点击外部关闭右键菜单
onMounted(() => {
  document.addEventListener('click', hideContextMenu)
})

onUnmounted(() => {
  document.removeEventListener('click', hideContextMenu)
})
</script>

<style scoped>
.web-drive-container {
  width: 100%;
  height: 100%;
  background: white;
  position: relative;
  border-radius: 0.5rem;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.browser-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-bottom: 1px solid #f0f0f0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: #6b7280;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.icon-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
  color: #6b7280;
}

.icon-btn:hover, .icon-btn.active {
  background: rgba(0, 0, 0, 0.05);
  color: #111827;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #f0f0f0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.btn-primary {
  background: #2563eb;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  transition: background-color 0.2s;
}

.btn-primary:hover {
  background: #1d4ed8;
}

.btn-outline {
  background: white;
  border: 1px solid #d1d5db;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  transition: all 0.2s;
}

.btn-outline:hover {
  background: rgba(0, 0, 0, 0.02);
}

.toolbar-right {
  display: flex;
  align-items: center;
}

.search-container {
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
  padding: 0.5rem 0.75rem 0.5rem 2.5rem;
  width: 16rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  font-size: 0.875rem;
}

.search-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.file-container {
  flex: 1;
  overflow: auto;
  padding: 1rem;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 1rem;
}

.file-item {
  padding: 1rem;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
}

.file-item:hover {
  background: #f9fafb;
}

.file-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 0.75rem;
}

.file-icon-container {
  width: 3rem;
  height: 3rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.folder-icon {
  width: 2.5rem;
  height: 2.5rem;
  color: #2563eb;
}

.file-icon {
  width: 2.5rem;
  height: 2.5rem;
  color: #6b7280;
}

.file-info {
  width: 100%;
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

.file-modified {
  font-size: 0.75rem;
  color: #9ca3af;
  margin: 0.25rem 0 0 0;
}

.file-list {
  width: 100%;
}

.file-list table {
  width: 100%;
  border-collapse: collapse;
}

.file-list th {
  text-align: left;
  padding: 0.75rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: #6b7280;
  border-bottom: 1px solid #e5e7eb;
}

.file-list td {
  padding: 0.75rem;
  font-size: 0.875rem;
  color: #111827;
  border-bottom: 1px solid #f3f4f6;
}

.file-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.context-menu {
  position: fixed;
  z-index: 50;
  background: white;
  border-radius: 0.375rem;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  width: 180px;
  overflow: hidden;
}

.context-menu ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.context-menu li {
  padding: 0.5rem 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.context-menu li:hover {
  background: #f3f4f6;
}

.context-menu li.danger {
  color: #ef4444;
}

.context-menu li.danger:hover {
  background: #fee2e2;
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}

.icon-xs {
  width: 0.75rem;
  height: 0.75rem;
}
</style> 