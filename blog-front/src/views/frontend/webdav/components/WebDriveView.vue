<template>
  <div class="web-drive-container">
    <!-- 移动端视图 -->
    <MobileView 
      v-if="isMobile" 
      :mobile-files="convertedFiles" 
      @upload="openUploadModal"
      @open="handleMobileFileOpen"
      @share="shareFile"
      @download="downloadFile"
      @rename="handleMobileRename"
      @delete="deleteFile"
    />
    
    <!-- 桌面端视图 -->
    <div v-else class="desktop-view">
      <!-- 文件预览组件 -->
      <FilePreview
        v-if="showFilePreview"
        :file="selectedFile"
        @close="closeFilePreview"
      />
      
      <template v-else>
        <div class="browser-header">
          <div class="header-left">
            <div class="breadcrumb">
              <HomeIcon class="icon-sm" @click="navigateToRoot" />
              <template v-if="pathSegments.length > 0">
                <ChevronRightIcon class="icon-xs" />
                <template v-for="(segment, index) in pathSegments" :key="index">
                  <span 
                    class="path-segment" 
                    @click="navigateToPathSegment(index)"
                  >{{ segment.name }}</span>
                  <ChevronRightIcon v-if="index < pathSegments.length - 1" class="icon-xs" />
                </template>
              </template>
              <span v-else>我的网盘</span>
            </div>
          </div>
          <div class="header-right">
            <button class="icon-btn" @click="openSettings">
              <SettingsIcon class="icon-sm" />
            </button>
          </div>
        </div>

        <div class="toolbar">
          <div class="toolbar-left">
            <button v-if="currentParentId" class="btn-outline" @click="navigateToParent">
              <ArrowLeftIcon class="icon-sm" />
              返回上级
            </button>
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
          <transition name="simple-fade" mode="out-in">
            <div v-if="isLoading" class="loading-container">
              <div class="loading-spinner"></div>
              <p>加载中...</p>
            </div>
            <div v-else :key="currentParentId || 'root'" class="file-container-inner">
              <div class="file-grid">
                <div
                  v-for="(file, index) in filteredFiles"
                  :key="file.id || index"
                  class="file-item"
                  :class="{ 'folder-item': file.type === 'folder' }"
                  @click="handleFileClick(file)"
                  @contextmenu.prevent="showContextMenu($event, file)"
                >
                  <div class="file-content">
                    <div class="file-icon-container">
                      <FolderIcon v-if="file.type === 'folder'" class="folder-icon" />
                      <component 
                        v-else-if="file.icon" 
                        :is="file.icon" 
                        :class="[
                          'file-icon', 
                          file.type === 'image' ? 'image-icon' : '',
                          file.type === 'video' ? 'video-icon' : '',
                          file.type === 'audio' ? 'audio-icon' : '',
                          file.type === 'code' ? 'code-icon' : '',
                          file.type === 'pdf' ? 'pdf-icon' : '',
                          file.type === 'archive' ? 'archive-icon' : '',
                          file.type === 'spreadsheet' ? 'spreadsheet-icon' : '',
                          file.type === 'presentation' ? 'presentation-icon' : ''
                        ]"
                      />
                      <FileIcon v-else class="file-icon" />
                    </div>
                    <div class="file-info">
                      <p class="file-name" :title="file.name">{{ file.name }}</p>
                      <div class="file-details">
                        <p class="file-size">{{ file.size }}</p>
                        <p class="file-modified" v-if="file.modified">{{ file.modified }}</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </transition>
        </div>
      </template>
    </div>
    
    <!-- 设置弹窗 - 按需显示 -->
    <SettingsModal v-if="showSettingsModal" @close="showSettingsModal = false" />
    
    <!-- 上传弹窗 - 按需显示 -->
    <UploadModal 
      v-if="showUploadModal" 
      :upload-progress="uploadProgress" 
      @close="showUploadModal = false" 
      @upload="handleUploadFiles"
    />
    
    <!-- 分享链接弹窗 - 按需显示 -->
    <ShareLinkPopup v-if="showShareLinkPopup" :file="selectedFile" @close="showShareLinkPopup = false" />
    
    <!-- 新建文件夹弹窗 -->
    <div v-if="showNewFolderDialog" class="dialog-overlay" @click.self="cancelDialog">
      <div class="dialog-box">
        <div class="dialog-header">
          <h3>新建文件夹</h3>
          <button class="close-btn" @click="cancelDialog">×</button>
        </div>
        <div class="dialog-body">
          <input 
            type="text" 
            v-model="newFolderName" 
            placeholder="请输入文件夹名称" 
            class="dialog-input"
            ref="folderNameInput"
            @keyup.enter="confirmNewFolder"
          />
        </div>
        <div class="dialog-footer">
          <button class="btn-outline" @click="cancelDialog">取消</button>
          <button class="btn-primary" @click="confirmNewFolder">确定</button>
        </div>
      </div>
    </div>
    
    <!-- 重命名弹窗 -->
    <div v-if="showRenameDialog" class="dialog-overlay" @click.self="cancelDialog">
      <div class="dialog-box">
        <div class="dialog-header">
          <h3>重命名</h3>
          <button class="close-btn" @click="cancelDialog">×</button>
        </div>
        <div class="dialog-body">
          <div v-if="fileToRename && !fileToRename.type.includes('folder')" class="filename-container">
            <input 
              type="text" 
              v-model="fileNameWithoutExt" 
              placeholder="文件名" 
              class="dialog-input filename-input"
              ref="fileNameInput"
              @keyup.enter="confirmRename"
            />
            <div class="extension-container">
              <div 
                class="extension-wrapper"
                @dblclick="enableExtensionEdit"
                :title="editingExtension ? '' : '双击编辑后缀名'"
              >
                <input 
                  type="text" 
                  v-model="fileExtension" 
                  class="dialog-input extension-input"
                  :disabled="!editingExtension"
                  @keyup.enter="confirmRename"
                  ref="extensionInput"
                />
              </div>
              <button 
                class="extension-edit-btn" 
                :class="{ 'active': editingExtension }"
                @click="toggleExtensionEdit"
                :title="editingExtension ? '锁定后缀名' : '编辑后缀名'"
              >
                <span v-if="editingExtension">🔓</span>
                <span v-else>🔒</span>
              </button>
            </div>
          </div>
          <input 
            v-else
            type="text" 
            v-model="newFileName" 
            placeholder="请输入新名称" 
            class="dialog-input"
            ref="folderNameInput"
            @keyup.enter="confirmRename"
          />
        </div>
        <div class="dialog-footer">
          <button class="btn-outline" @click="cancelDialog">取消</button>
          <button class="btn-primary" @click="confirmRename">确定</button>
        </div>
      </div>
    </div>
    
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
import { ref, computed, onMounted, onUnmounted, nextTick, provide } from 'vue'
import type { FileItem } from '../utils/types/file'
import SettingsModal from '../modals/SettingsModal.vue'
import UploadModal from '../modals/UploadModal.vue'
import ShareLinkPopup from '../modals/ShareLinkPopup.vue'
import MobileView from './MobileView.vue'
import FilePreview from './FilePreview.vue'
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
  MoreHorizontalIcon,
  ArrowLeftIcon,
  FileCodeIcon,
  FileZipIcon,
  FilePdfIcon,
  FileSpreadsheetIcon,
  FilePresentationIcon,
} from '../utils/icons'
import { listFiles, createFolder, uploadFile, getDownloadUrl, renameFile as apiRenameFile, deleteFile as apiDeleteFile, FileInfo } from '@/api/file'

// 状态变量
const uploadProgress = ref(0)
const viewMode = ref<'grid'>('grid')
const currentPath = ref('')
const currentParentId = ref('')
const searchQuery = ref('')
const showSettingsModal = ref(false)
const showUploadModal = ref(false)
const showShareLinkPopup = ref(false)
const isLoading = ref(false)
const selectedFile = ref<FileItem>({
  name: '',
  type: 'file',
  size: ''
})
const showFilePreview = ref(false)
const contextMenu = ref({
  show: false,
  x: 0,
  y: 0,
  file: {
    name: '',
    type: 'file',
    size: ''
  } as FileItem
})
const isMobile = ref(false)
const showNewFolderDialog = ref(false)
const showRenameDialog = ref(false)
const newFolderName = ref('新建文件夹')
const newFileName = ref('')
const fileToRename = ref<FileItem | null>(null)
const fileNameWithoutExt = ref('')
const fileExtension = ref('')
const editingExtension = ref(false)
const folderNameInput = ref<HTMLInputElement | null>(null)
const fileNameInput = ref<HTMLInputElement | null>(null)
const extensionInput = ref<HTMLInputElement | null>(null)

// 路径导航历史
interface PathSegment {
  id: string;
  name: string;
}
const pathSegments = ref<PathSegment[]>([])

// 为FilePreview组件提供路径导航功能
provide('pathSegments', pathSegments.value)
provide('navigateToRoot', navigateToRoot)
provide('navigateToPathSegment', navigateToPathSegment)

// 文件数据
const files = ref<FileItem[]>([])
const apiFiles = ref<FileInfo[]>([])

// 移动端文件列表
const mobileFiles = ref<FileItem[]>([])

// 将API返回的文件数据转换为组件使用的格式
const convertedFiles = computed<FileItem[]>(() => {
  console.log('WebDriveView - API返回的文件数据:', apiFiles.value);
  
  return apiFiles.value.map(file => {
    // 确定文件图标
    let icon;
    let fileType: 'file' | 'folder' | 'image' | 'video' | 'audio' | 'code' | 'pdf' | 'archive' | 'spreadsheet' | 'presentation' = file.is_folder ? 'folder' : 'file';
    
    if (file.is_folder) {
      icon = FolderIcon;
    } else if (file.mimeType) {
      // 根据MIME类型确定图标
      if (file.mimeType.startsWith('image/')) {
        icon = ImageIcon;
        fileType = 'image';
      } else if (file.mimeType.startsWith('video/')) {
        icon = VideoIcon;
        fileType = 'video';
      } else if (file.mimeType.startsWith('audio/')) {
        icon = MusicIcon;
        fileType = 'audio';
      } else if (file.mimeType.startsWith('application/pdf')) {
        icon = FilePdfIcon;
        fileType = 'pdf';
      } else if (
        file.mimeType.includes('zip') || 
        file.mimeType.includes('compressed') || 
        file.mimeType.includes('archive') ||
        file.mimeType.includes('x-tar') ||
        file.mimeType.includes('x-rar')
      ) {
        icon = FileZipIcon;
        fileType = 'archive';
      } else if (
        file.mimeType.includes('excel') || 
        file.mimeType.includes('spreadsheet') ||
        file.mimeType.includes('csv')
      ) {
        icon = FileSpreadsheetIcon;
        fileType = 'spreadsheet';
      } else if (
        file.mimeType.includes('powerpoint') || 
        file.mimeType.includes('presentation')
      ) {
        icon = FilePresentationIcon;
        fileType = 'presentation';
      } else if (
        file.mimeType.includes('javascript') || 
        file.mimeType.includes('json') || 
        file.mimeType.includes('html') || 
        file.mimeType.includes('css') ||
        file.mimeType.includes('xml') ||
        file.mimeType.includes('text/plain')
      ) {
        icon = FileCodeIcon;
        fileType = 'code';
      } else {
        icon = FileTextIcon;
      }
    } else {
      // 如果没有MIME类型，尝试通过文件扩展名判断
      const extension = file.name.split('.').pop()?.toLowerCase();
      if (extension) {
        if (['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'].includes(extension)) {
          icon = ImageIcon;
          fileType = 'image';
        } else if (['mp4', 'webm', 'avi', 'mov', 'wmv', 'flv', 'mkv'].includes(extension)) {
          icon = VideoIcon;
          fileType = 'video';
        } else if (['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a'].includes(extension)) {
          icon = MusicIcon;
          fileType = 'audio';
        } else if (extension === 'pdf') {
          icon = FilePdfIcon;
          fileType = 'pdf';
        } else if (['zip', 'rar', '7z', 'tar', 'gz', 'bz2'].includes(extension)) {
          icon = FileZipIcon;
          fileType = 'archive';
        } else if (['xls', 'xlsx', 'csv', 'ods'].includes(extension)) {
          icon = FileSpreadsheetIcon;
          fileType = 'spreadsheet';
        } else if (['ppt', 'pptx', 'odp'].includes(extension)) {
          icon = FilePresentationIcon;
          fileType = 'presentation';
        } else if (['js', 'ts', 'html', 'css', 'xml', 'json', 'txt', 'md', 'py', 'java', 'c', 'cpp', 'go', 'php', 'rb'].includes(extension)) {
          icon = FileCodeIcon;
          fileType = 'code';
        } else {
          icon = FileTextIcon;
        }
      } else {
        icon = FileTextIcon;
      }
    }

    // 格式化文件大小
    const formatSize = (size: number): string => {
      if (file.is_folder) return '文件夹';
      if (size < 1024) return `${size} B`;
      if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
      if (size < 1024 * 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`;
      return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    };

    // 格式化修改日期
    const formatDate = (dateStr: string): string => {
      if (!dateStr) return '-';
      const date = new Date(dateStr);
      return date.toLocaleString('zh-CN', { 
        year: 'numeric', 
        month: 'long', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    };

    // 确保文件ID存在且转换为字符串
    const fileId = file.id ? file.id.toString() : '';
    console.log('WebDriveView - 处理文件:', file.name, '原始ID:', file.id, '转换后ID:', fileId);

    return {
      id: fileId, // 确保ID存在且为字符串
      name: file.name,
      type: fileType,
      size: formatSize(file.size),
      modified: formatDate(file.updateTime),
      icon,
      originalFile: file // 保留原始数据，以便后续操作
    } as FileItem;
  });
});

// 过滤文件列表
const filteredFiles = computed(() => {
  if (!searchQuery.value) return convertedFiles.value;
  
  const query = searchQuery.value.toLowerCase();
  return convertedFiles.value.filter(file => 
    file.name.toLowerCase().includes(query)
  );
})

// 计算右键菜单位置
const contextMenuStyle = computed(() => {
  return {
    top: `${contextMenu.value.y}px`,
    left: `${contextMenu.value.x}px`
  }
})

// 获取文件列表
const fetchFiles = async (parentId: string = '') => {
  try {
    isLoading.value = true;
    const response = await listFiles(parentId);
    apiFiles.value = response;
    // 更新移动端文件列表
    mobileFiles.value = convertedFiles.value;
    isLoading.value = false;
  } catch (error) {
    console.error('获取文件列表失败:', error);
    ElMessage.error('获取文件列表失败');
    isLoading.value = false;
  }
};

// 检测移动设备
function checkMobile() {
  isMobile.value = window.innerWidth < 768;
}

// 导航到根目录
function navigateToRoot() {
  currentPath.value = '';
  currentParentId.value = '';
  pathSegments.value = []; // 清空路径导航历史
  fetchFiles();
}

// 导航到特定路径段
function navigateToPathSegment(index: number) {
  if (index < 0 || index >= pathSegments.value.length) return;
  
  // 获取目标路径段
  const targetSegment = pathSegments.value[index];
  
  // 更新当前路径和父ID
  currentParentId.value = targetSegment.id;
  
  // 更新路径段历史（保留到当前点击的段）
  pathSegments.value = pathSegments.value.slice(0, index + 1);
  
  // 重新构建当前路径
  currentPath.value = pathSegments.value.map(segment => segment.name).join('/');
  
  // 获取文件列表
  fetchFiles(targetSegment.id);
}

// 导航到上一级目录
function navigateToParent() {
  if (pathSegments.value.length <= 1) {
    // 如果只有一级或没有，则返回根目录
    navigateToRoot();
  } else {
    // 否则返回上一级
    const parentIndex = pathSegments.value.length - 2;
    navigateToPathSegment(parentIndex);
  }
}

// 打开设置弹窗
function openSettings() {
  showSettingsModal.value = true;
}

// 打开上传弹窗
function openUploadModal() {
  showUploadModal.value = true;
}

// 创建新文件夹
function createNewFolder() {
  showNewFolderDialog.value = true;
  newFolderName.value = '新建文件夹';
  
  // 在下一个DOM更新周期后聚焦输入框并选中文本
  nextTick(() => {
    if (folderNameInput.value) {
      folderNameInput.value.focus();
      folderNameInput.value.select();
    }
  });
}

// 确认创建新文件夹
async function confirmNewFolder() {
  if (newFolderName.value.trim()) {
    try {
      await createFolder(currentParentId.value, newFolderName.value);
      ElMessage.success('文件夹创建成功');
      // 刷新文件列表
      fetchFiles(currentParentId.value);
      showNewFolderDialog.value = false;
    } catch (error) {
      console.error('创建文件夹失败:', error);
      ElMessage.error('创建文件夹失败');
    }
  }
}

// 重命名文件
function renameFile(file: FileItem) {
  if (!file.id) return;
  
  fileToRename.value = file;
  
  if (file.type !== 'folder') {
    // 分离文件名和扩展名
    const lastDotIndex = file.name.lastIndexOf('.');
    if (lastDotIndex > 0) {
      fileNameWithoutExt.value = file.name.substring(0, lastDotIndex);
      fileExtension.value = file.name.substring(lastDotIndex);
    } else {
      fileNameWithoutExt.value = file.name;
      fileExtension.value = '';
    }
    newFileName.value = file.name;
    editingExtension.value = false;
  } else {
    newFileName.value = file.name;
  }
  
  showRenameDialog.value = true;
  closeContextMenu();
  
  // 在下一个DOM更新周期后聚焦输入框并选中文本
  nextTick(() => {
    if (file.type !== 'folder' && fileNameInput.value) {
      fileNameInput.value.focus();
      fileNameInput.value.select();
    } else if (folderNameInput.value) {
      folderNameInput.value.focus();
      folderNameInput.value.select();
    }
  });
}

// 切换扩展名编辑状态
function toggleExtensionEdit() {
  editingExtension.value = !editingExtension.value;
  
  // 如果启用了扩展名编辑，聚焦到扩展名输入框
  if (editingExtension.value) {
    nextTick(() => {
      const extensionInput = document.querySelector('.extension-input') as HTMLInputElement;
      if (extensionInput) {
        extensionInput.focus();
        extensionInput.select();
      }
    });
  }
}

// 启用扩展名编辑（双击时调用）
function enableExtensionEdit(event: MouseEvent) {
  // 阻止事件冒泡，防止触发其他点击事件
  event.stopPropagation();
  
  console.log('Double click detected!'); // 调试信息
  
  if (!editingExtension.value) {
    editingExtension.value = true;
    
    // 延迟一下再聚焦，确保禁用状态已经解除
    nextTick(() => {
      if (extensionInput.value) {
        extensionInput.value.disabled = false;
        extensionInput.value.focus();
        extensionInput.value.select();
        console.log('Extension input focused'); // 调试信息
      } else {
        console.log('Extension input not found'); // 调试信息
      }
    });
  }
}

// 确认重命名
async function confirmRename() {
  // 构建完整的新文件名
  if (fileToRename.value && fileToRename.value.type !== 'folder') {
    newFileName.value = fileNameWithoutExt.value + fileExtension.value;
  }
  
  if (fileToRename.value && fileToRename.value.id && newFileName.value.trim() && newFileName.value !== fileToRename.value.name) {
    try {
      await apiRenameFile(fileToRename.value.id, newFileName.value);
      ElMessage.success('重命名成功');
      fetchFiles(currentParentId.value);
      showRenameDialog.value = false;
    } catch (error) {
      console.error('重命名失败:', error);
      ElMessage.error('重命名失败');
    }
  } else if (newFileName.value === fileToRename.value?.name) {
    showRenameDialog.value = false;
  }
}

// 取消对话框
function cancelDialog() {
  showNewFolderDialog.value = false;
  showRenameDialog.value = false;
}

// 处理文件点击
function handleFileClick(file: FileItem) {
  console.log('WebDriveView - 点击文件:', file.name, '文件ID:', file.id, '文件类型:', file.type);
  
  if (file.type === 'folder') {
    // 如果是文件夹，进入该文件夹
    const folderId = file.id as string;
    currentParentId.value = folderId;
    
    // 更新路径导航历史
    pathSegments.value.push({
      id: folderId,
      name: file.name
    });
    
    // 更新当前路径
    currentPath.value = pathSegments.value.map(segment => segment.name).join('/');
    
    // 获取文件列表
    fetchFiles(folderId);
  } else {
    // 如果是文件，打开预览
    console.log('WebDriveView - 准备预览文件:', file);
    selectedFile.value = file;
    showFilePreview.value = true;
  }
}

// 关闭文件预览
function closeFilePreview() {
  showFilePreview.value = false;
}

// 处理移动端文件打开
function handleMobileFileOpen(file: FileItem) {
  handleFileClick(file);
}

// 处理移动端重命名
function handleMobileRename(file: FileItem) {
  renameFile(file);
}

// 打开文件
function openFile(file: FileItem) {
  handleFileClick(file);
  closeContextMenu();
}

// 分享文件
function shareFile(file: FileItem) {
  selectedFile.value = file;
  showShareLinkPopup.value = true;
  closeContextMenu();
}

// 下载文件
function downloadFile(file: FileItem) {
  if (file.type === 'file' && file.id) {
    const downloadUrl = getDownloadUrl(file.id);
    window.open(downloadUrl, '_blank');
  }
  closeContextMenu();
}

// 删除文件
function deleteFile(file: FileItem) {
  if (!file.id) return;
  
  if (confirm(`确定要删除 ${file.name} 吗？`)) {
    apiDeleteFile(file.id)
      .then(() => {
        ElMessage.success('删除成功');
        fetchFiles(currentParentId.value);
      })
      .catch((error: any) => {
        console.error('删除失败:', error);
        ElMessage.error('删除失败');
      });
  }
  closeContextMenu();
}

// 显示上下文菜单
function showContextMenu(event: MouseEvent, file: FileItem) {
  event.preventDefault();
  contextMenu.value.show = true;
  contextMenu.value.x = event.clientX;
  contextMenu.value.y = event.clientY;
  contextMenu.value.file = file;

  // 添加全局点击事件监听器，用于关闭上下文菜单
  document.addEventListener('click', closeContextMenu);
}

// 关闭上下文菜单
function closeContextMenu() {
  contextMenu.value.show = false;
  document.removeEventListener('click', closeContextMenu);
}

// 处理文件上传
async function handleUploadFiles(files: File[]) {
  if (!files.length) return;

  try {
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      uploadProgress.value = Math.round((i / files.length) * 100);
      
      await uploadFile(currentParentId.value, file);
    }
    
    uploadProgress.value = 100;
    ElMessage.success('上传完成');
    showUploadModal.value = false;
    fetchFiles(currentParentId.value);
  } catch (error) {
    console.error('上传失败:', error);
    ElMessage.error('上传失败');
  }
}

onMounted(() => {
  checkMobile();
  window.addEventListener('resize', checkMobile);
  // 初始化时获取文件列表
  fetchFiles();
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile);
})
</script>

<style lang="less">
.web-drive-container {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 40px); /* 减去padding的高度 */
  background-color: #ffffff;
  
  .desktop-view {
    display: flex;
    flex-direction: column;
    height: 100%;
    flex: 1;
    background-color: #ffffff;
  }
  
  .browser-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    flex-shrink: 0;
    
    .header-left {
      .breadcrumb {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        background-color: #f8f9fa;
        padding: 10px 16px;
        border-radius: 50px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
        
        .icon-sm {
          cursor: pointer;
          color: #666;
          
          &:hover {
            color: #2a8aff;
          }
        }
        
        .path-segment {
          cursor: pointer;
          color: #666;
          font-weight: 500;
          padding: 2px 8px;
          border-radius: 4px;
          transition: all 0.2s ease;
          
          &:hover {
            color: #2a8aff;
            background-color: rgba(42, 138, 255, 0.1);
            text-decoration: none;
          }
        }
        
        .icon-xs {
          color: #aaa;
        }
      }
    }
    
    .header-right {
      display: flex;
      gap: 10px;
      
      .icon-btn {
        background: none;
        border: none;
        padding: 8px;
        cursor: pointer;
        border-radius: 50%;
        transition: all 0.2s ease;
        
        &.active {
          background-color: #e6f0ff;
          color: #2a8aff;
        }
        
        &:hover {
          background-color: #f0f5ff;
          transform: translateY(-2px);
        }
        
        .icon-sm {
          width: 20px;
          height: 20px;
        }
      }
    }
  }
  
  .toolbar {
    display: flex;
    justify-content: space-between;
    margin-bottom: 20px;
    flex-shrink: 0;
    
    .toolbar-left {
      display: flex;
      gap: 10px;
      
      button {
        display: flex;
        align-items: center;
        gap: 5px;
        padding: 8px 15px;
        border-radius: 4px;
        cursor: pointer;
        font-size: 14px;
        
        .icon-sm {
          width: 16px;
          height: 16px;
        }
      }
      
      .btn-primary {
        background-color: #2a8aff;
        color: white;
        border: none;
        
        &:hover {
          background-color: #1a7aef;
        }
      }
      
      .btn-outline {
        background-color: white;
        color: #666;
        border: 1px solid #ddd;
        
        &:hover {
          background-color: #f5f5f5;
        }
      }
    }
    
    .toolbar-right {
      .search-container {
        position: relative;
        
        .search-icon {
          position: absolute;
          left: 10px;
          top: 50%;
          transform: translateY(-50%);
          width: 16px;
          height: 16px;
          color: #999;
        }
        
        .search-input {
          padding: 8px 10px 8px 35px;
          border: 1px solid #ddd;
          border-radius: 4px;
          width: 250px;
          font-size: 14px;
          
          &:focus {
            outline: none;
            border-color: #2a8aff;
          }
        }
      }
    }
  }
  
  .file-container {
    flex: 1;
    overflow: auto;
    display: flex;
    flex-direction: column;
    min-height: 400px; /* 最小高度，确保在内容少时也有一定高度 */
    background-color: #ffffff;
    
    .file-container-inner {
      display: flex;
      flex-direction: column;
    }
    
    .loading-container {
      flex: 1;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
    }
    
    .file-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
      gap: 10px;
      flex: 1;
      background-color: #ffffff;
      
      .file-item {
        cursor: pointer;
        border-radius: 8px;
        padding: 15px;
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
        height: 140px; /* 固定高度 */
        display: flex;
        background-color: #ffffff;
        
        &:hover {
          background-color: #f5f5f5;
          transform: translateY(-3px);
          box-shadow: 0 5px 15px rgba(0, 0, 0, 0.05);
        }
        
        &.folder-item {
          &:hover {
            .file-icon-container {
              .folder-icon {
                transform: scale(1.1);
              }
            }
          }
        }
        
        .file-content {
          display: flex;
          flex-direction: column;
          align-items: center;
          height: 100%;
          width: 100%;
          
          .file-icon-container {
            width: 60px;
            height: 60px;
            display: flex;
            align-items: center;
            justify-content: center;
            
            .folder-icon, .file-icon {
              width: 40px;
              height: 40px;
              transition: transform 0.3s ease;
            }
            
            .folder-icon {
              color: #2a8aff;
            }
            
            .file-icon {
              color: #2a8aff;
              
              &.image-icon {
                color: #4CAF50; /* 绿色 */
              }
              
              &.video-icon {
                color: #FF5722; /* 橙红色 */
              }
              
              &.audio-icon {
                color: #9C27B0; /* 紫色 */
              }
              
              &.code-icon {
                color: #607D8B; /* 蓝灰色 */
              }
              
              &.pdf-icon {
                color: #F44336; /* 红色 */
              }
              
              &.archive-icon {
                color: #795548; /* 棕色 */
              }
              
              &.spreadsheet-icon {
                color: #4CAF50; /* 绿色 */
              }
              
              &.presentation-icon {
                color: #FF9800; /* 橙色 */
              }
            }
          }
          
          .file-info {
            text-align: center;
            width: 100%;
            display: flex;
            flex-direction: column;
            height: 50px;
            
            .file-name {
              font-size: 14px;
              margin: 0 0 3px 0;
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
              max-width: 100%;
              line-height: 1.2;
            }
            
            .file-details {
              height: 30px;
              
              .file-size, .file-modified {
                font-size: 12px;
                color: #999;
                margin: 0;
                line-height: 1.2;
              }
            }
          }
        }
      }
    }
  }
  
  .context-menu {
    position: fixed;
    background: white;
    border-radius: 4px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    z-index: 1000;
    
    ul {
      list-style: none;
      padding: 0;
      margin: 0;
      
      li {
        padding: 10px 15px;
        display: flex;
        align-items: center;
        gap: 10px;
        cursor: pointer;
        
        &:hover {
          background-color: #f5f5f5;
        }
        
        &.danger {
          color: #ff4d4f;
        }
        
        .icon-xs {
          width: 14px;
          height: 14px;
        }
      }
    }
  }
}

// 简单的淡入淡出效果
.simple-fade-enter-active,
.simple-fade-leave-active {
  transition: opacity 0.25s ease;
}

.simple-fade-enter-from,
.simple-fade-leave-to {
  opacity: 0;
}

// 图标尺寸
.icon-xs {
  width: 12px;
  height: 12px;
}

.icon-sm {
  width: 16px;
  height: 16px;
}

// 加载动画
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 300px;
  flex: 1;
  
  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #f3f3f3;
    border-top: 3px solid #2a8aff;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 15px;
  }
  
  p {
    color: #666;
    font-size: 14px;
  }
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

// 对话框样式
.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1100;
}

.dialog-box {
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  width: 400px;
  max-width: 90%;
  overflow: hidden;
  animation: dialog-appear 0.2s ease-out;
  
  .dialog-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #eee;
    
    h3 {
      margin: 0;
      font-size: 18px;
      font-weight: 500;
      color: #333;
    }
    
    .close-btn {
      background: none;
      border: none;
      font-size: 20px;
      color: #999;
      cursor: pointer;
      padding: 0;
      
      &:hover {
        color: #666;
      }
    }
  }
  
  .dialog-body {
    padding: 20px;
    
    .filename-container {
      display: flex;
      align-items: center;
      gap: 5px;
      width: 100%;
      
      .filename-input {
        flex: 1;
      }
      
      .extension-container {
        display: flex;
        align-items: center;
        position: relative;
        
        .extension-wrapper {
          width: 80px;
          cursor: pointer;
        }
        
        .extension-input {
          width: 100%;
          background-color: #f8f8f8;
          color: #666;
          
          &:disabled {
            cursor: pointer; /* 改为指针，提示可以交互 */
            opacity: 0.8;
            pointer-events: none; /* 禁用事件，让父元素处理双击 */
          }
          
          &:not(:disabled) {
            background-color: #fff;
            color: #333;
          }
        }
        
        .extension-edit-btn {
          position: absolute;
          right: 8px;
          background: none;
          border: none;
          cursor: pointer;
          padding: 0;
          font-size: 14px;
          color: #999;
          display: flex;
          align-items: center;
          justify-content: center;
          
          &.active {
            color: #2a8aff;
          }
          
          &:hover {
            color: #666;
          }
        }
      }
    }
    
    .dialog-input {
      width: 100%;
      padding: 10px 12px;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-size: 14px;
      
      &:focus {
        outline: none;
        border-color: #2a8aff;
        box-shadow: 0 0 0 2px rgba(42, 138, 255, 0.2);
      }
    }
  }
  
  .dialog-footer {
    padding: 16px 20px;
    border-top: 1px solid #eee;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    
    button {
      padding: 8px 16px;
      border-radius: 4px;
      cursor: pointer;
      font-size: 14px;
      
      &.btn-outline {
        background-color: white;
        color: #666;
        border: 1px solid #ddd;
        
        &:hover {
          background-color: #f5f5f5;
        }
      }
      
      &.btn-primary {
        background-color: #2a8aff;
        color: white;
        border: none;
        
        &:hover {
          background-color: #1a7aef;
        }
      }
    }
  }
}

@keyframes dialog-appear {
  from {
    opacity: 0;
    transform: translateY(-20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style> 