<template>
  <div class="w-full h-screen bg-white flex flex-col relative">
      <div class="p-4 border-b border-[#f3f4f6]">
      <div class="flex mb-4">
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
      <div class="relative">
        <SearchIcon class="absolute left-3 top-1/2 [transform:translateY(-50%)] text-[#9ca3af] w-4 h-4" />
        <input type="text" placeholder="搜索文件..." class="w-full py-3 pr-4 pl-10 bg-[#f9fafb] border border-[#e5e7eb] rounded-lg text-[0.875rem]" v-model="searchQuery" />
        </div>
      </div>

    <div class="flex-1 overflow-y-auto py-2 px-4">
        <div
        v-for="(file, index) in filteredFiles"
          :key="index"
          class="flex items-center py-3 border-b border-[#f3f4f6]"
        @click="handleFileClick(file)"
        >
        <div class="mr-4">
          <FolderIcon v-if="file.type === 'folder'" class="w-8 h-8 text-[#2563eb]" />
          <component v-else-if="file.icon" :is="file.icon" class="w-8 h-8 text-[#6b7280]" />
          <FileIcon v-else class="w-8 h-8 text-[#6b7280]" />
          </div>
        <div class="flex-1">
          <div class="flex justify-between items-center">
            <p class="text-[0.875rem] font-medium text-[#111827]">{{ file.name }}</p>
            <button class="p-2 cursor-pointer text-[#6b7280]" @click.stop="showOptions(file)">
              <MoreHorizontalIcon class="w-4 h-4" />
            </button>
          </div>
          <p class="text-[0.75rem] text-[#6b7280] mt-1">{{ file.size }}</p>
        </div>
      </div>
    </div>

    <div class="fixed right-6 bottom-6 w-14 h-14 rounded-[50%] bg-[#2563eb] text-white flex items-center justify-center shadow-[0_4px_6px_-1px_rgba(0,0,0,0.1),0_2px_4px_-1px_rgba(0,0,0,0.06)] cursor-pointer z-10" @click="$emit('upload')">
      <PlusIcon class="icon-sm" />
      </div>

    <div v-if="showOptionsMenu" class="fixed bottom-0 left-0 right-0 bg-white rounded-t-2xl shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] z-20 p-4">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-[1rem] text-[#111827]">{{ selectedFile?.name }}</h3>
        <button class="p-2 cursor-pointer" @click="showOptionsMenu = false">
          <XIcon class="icon-sm" />
          </button>
      </div>
      <div class="flex flex-col">
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
        <div class="option-item text-[#ef4444]" @click="deleteFile">
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
/* 组件类：.tab 3 处、.option-item 5 处、.icon-sm 8 处复用 */
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

.icon-sm {
  width: 1.25rem;
  height: 1.25rem;
}

/* 字重：style.less 全局 h1,h2,h3{font-weight:400} 是无层级规则，压过工具类 */
.options-header h3 {
  font-weight: 600;
}
</style>
