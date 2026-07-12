<template>
  <div class="web-drive-container">
    <!-- 响应式文件管理视图 -->
    <div class="responsive-view">
      <!-- 文件预览组件 -->
      <FilePreview v-if="showFilePreview" :file="selectedFile" @close="closeFilePreview" />

      <template v-else>
        <div class="browser-header">
          <div class="header-left">
            <div class="breadcrumb">
              <HomeIcon class="icon-sm" @click="navigateToRoot" />
              <template v-if="pathSegments.length > 0">
                <ChevronRightIcon class="icon-xs" />
                <template v-for="(segment, index) in pathSegments" :key="index">
                  <span class="path-segment" @click="navigateToPathSegment(index)">{{ segment.name }}</span>
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
            <button v-if="selectedFiles.size > 0 && !isFolderSelected" class="btn-primary" @click="downloadSelectedFiles">
              <UploadIcon class="icon-sm" transform="rotate(180)" />
              下载 ({{ selectedFiles.size }})
            </button>
            <button v-if="selectedFiles.size > 0 && !isFolderSelected" class="btn-outline" @click="shareSelectedFiles">
              <UploadIcon class="icon-sm" />
              分享 ({{ selectedFiles.size }})
            </button>
            <button v-if="selectedFiles.size === 1 && !isFolderSelected" class="btn-outline mobile-preview-btn" @click="previewSelectedFile">
              <FileIcon class="icon-sm" />
              预览
            </button>
            <button v-if="selectedFiles.size === 1 && isFolderSelected" class="btn-primary enter-folder-btn" @click="enterSelectedFolder">
              <FolderIcon class="icon-sm" />
              进入文件夹
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
                <div v-for="(file, index) in filteredFiles" :key="file.id || index" class="file-item"
                  :class="{ 
                    'folder-item': file.type === 'folder',
                    'new-uploaded-file': newUploadedFileIds.includes(file.id || ''),
                    'selected': file.id && selectedFiles.has(file.id)
                  }" 
                  @click="handleFileClick(file)"
                  @dblclick="handleFileDoubleClick(file)"
                  @contextmenu.prevent="showContextMenu($event, file)">
                  <div class="file-content">
                    <div class="file-icon-container">
                      <FolderIcon v-if="file.type === 'folder'" class="folder-icon" />
                      <component v-else-if="file.icon" :is="file.icon" :class="[
                        'file-icon',
                        file.type === 'image' ? 'image-icon' : '',
                        file.type === 'video' ? 'video-icon' : '',
                        file.type === 'audio' ? 'audio-icon' : '',
                        file.type === 'code' ? 'code-icon' : '',
                        file.type === 'pdf' ? 'pdf-icon' : '',
                        file.type === 'archive' ? 'archive-icon' : '',
                        file.type === 'spreadsheet' ? 'spreadsheet-icon' : '',
                        file.type === 'presentation' ? 'presentation-icon' : ''
                      ]" />
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
    <div v-if="showUploadModal" class="upload-modal-container">
      <div class="upload-modal-overlay" @click="closeUploadModal"></div>
      <UploadModal ref="uploadModalRef" :upload-progress="uploadProgress" @close="closeUploadModal"
        @upload="handleUploadFiles" @retry="handleRetryUpload" />
    </div>

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
          <input type="text" v-model="newFolderName" placeholder="请输入文件夹名称" class="dialog-input" ref="folderNameInput"
            @keyup.enter="confirmNewFolder" />
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
            <input type="text" v-model="fileNameWithoutExt" placeholder="文件名" class="dialog-input filename-input"
              ref="fileNameInput" @keyup.enter="confirmRename" />
            <div class="extension-container">
              <div class="extension-wrapper" @dblclick="enableExtensionEdit" :title="editingExtension ? '' : '双击编辑后缀名'">
                <input type="text" v-model="fileExtension" class="dialog-input extension-input"
                  :disabled="!editingExtension" @keyup.enter="confirmRename" ref="extensionInput" />
              </div>
              <button class="extension-edit-btn" :class="{ 'active': editingExtension }" @click="toggleExtensionEdit"
                :title="editingExtension ? '锁定后缀名' : '编辑后缀名'">
                <span v-if="editingExtension">🔓</span>
                <span v-else>🔒</span>
              </button>
            </div>
          </div>
          <input v-else type="text" v-model="newFileName" placeholder="请输入新名称" class="dialog-input"
            ref="folderNameInput" @keyup.enter="confirmRename" />
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
        <li @click="openFile(contextMenu.file)">
          <FileIcon class="icon-xs" /> 打开
        </li>
        <li @click="shareFile(contextMenu.file)">
          <UploadIcon class="icon-xs" /> 分享
        </li>
        <li @click="downloadFile(contextMenu.file)">
          <UploadIcon class="icon-xs" transform="rotate(180)" /> 下载
        </li>
        <li @click="renameFile(contextMenu.file)">
          <FileTextIcon class="icon-xs" /> 重命名
        </li>
        <li @click="deleteFile(contextMenu.file)" class="danger">
          <XIcon class="icon-xs" /> 删除
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, provide, type ComponentPublicInstance } from 'vue'
import type { FileItem } from '../utils/types/file'
import SettingsModal from '../modals/SettingsModal.vue'
import UploadModal from '../modals/UploadModal.vue'
import ShareLinkPopup from '../modals/ShareLinkPopup.vue'
import FilePreview from './FilePreview.vue'
import {
  HomeIcon,
  ChevronRightIcon,
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
  ArrowLeftIcon,
  FileCodeIcon,
  FileZipIcon,
  FilePdfIcon,
  FileSpreadsheetIcon,
  FilePresentationIcon,
} from '../utils/icons'
import { listFiles, createFolder, uploadFile, getDownloadUrl, renameFile as apiRenameFile, deleteFile as apiDeleteFile, initChunkUpload, uploadChunk, completeChunkUpload, getUploadedChunks, FileInfo } from '@/api/file'
import { getSystemConfig } from '@/api/system'

// 状态变量
const uploadProgress = ref(0)
const currentPath = ref('')
const currentParentId = ref('')
const searchQuery = ref('')
const showSettingsModal = ref(false)
const showUploadModal = ref(false)
const showShareLinkPopup = ref(false)
const isLoading = ref(false)
const selectedFiles = ref<Set<string>>(new Set()) // 存储选中的文件ID
// 定义UploadModalInstance类型
type UploadModalInstance = ComponentPublicInstance & {
  updateFileStatus: (fileIndex: number, status: 'success' | 'error' | 'uploading', message?: string, uploadedChunks?: number, totalChunks?: number) => void
}
// 使用正确的类型
const uploadModalRef = ref<UploadModalInstance | null>(null)
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

// 添加一个新的状态来跟踪新上传的文件ID
const newUploadedFileIds = ref<string[]>([]);

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
const apiFiles = ref<FileInfo[]>([])

// 将API返回的文件数据转换为组件使用的格式
const convertedFiles = computed<FileItem[]>(() => {
  return apiFiles.value.map(file => {
    // 确定文件图标
    let icon;
    let fileType: 'file' | 'folder' | 'image' | 'video' | 'audio' | 'code' | 'pdf' | 'archive' | 'spreadsheet' | 'presentation' | 'text' = file.is_folder ? 'folder' : 'file';

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
        file.mimeType.includes('text/plain') ||
        file.mimeType.includes('text/markdown') ||
        file.mimeType.includes('text/')
      ) {
        // 所有文本类型文件都归为text类型，以支持预览
        icon = FileCodeIcon;
        fileType = 'text';
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
        } else if (
          // 所有文本类型文件和编程语言文件都归为text类型，以支持预览
          ['txt', 'md', 'markdown', 'text', 'log', 'rtf', 'js', 'ts', 'html', 'css', 'xml', 'json', 'py', 'java', 'c', 'cpp', 'go', 'php', 'rb', 'sh', 'bat', 'ps1', 'sql', 'yaml', 'yml', 'toml', 'ini', 'conf', 'config'].includes(extension)
        ) {
          // 根据文件类型使用不同的图标，但都归为text类型以支持预览
          if (['js', 'ts', 'html', 'css', 'xml', 'json', 'py', 'java', 'c', 'cpp', 'go', 'php', 'rb', 'sh', 'sql', 'yaml', 'yml'].includes(extension)) {
            icon = FileCodeIcon;
          } else {
            icon = FileTextIcon;
          }
          fileType = 'text';
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

// 判断选中的是否为文件夹
const isFolderSelected = computed(() => {
  if (selectedFiles.value.size !== 1) return false
  const selectedId = Array.from(selectedFiles.value)[0]
  const selectedFile = convertedFiles.value.find(f => f.id === selectedId)
  return selectedFile?.type === 'folder'
})

// 获取文件列表
const fetchFiles = async (parentId: string = '') => {
  try {
    isLoading.value = true;
    const response = await listFiles(parentId);
    apiFiles.value = response;
  
    isLoading.value = false;
  } catch (error) {
    console.error('获取文件列表失败:', error);
    ElMessage.error('获取文件列表失败');
    isLoading.value = false;
  }
};



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

  if (!editingExtension.value) {
    editingExtension.value = true;

    // 延迟一下再聚焦，确保禁用状态已经解除
    nextTick(() => {
      if (extensionInput.value) {
        extensionInput.value.disabled = false;
        extensionInput.value.focus();
        extensionInput.value.select();
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

// 处理文件单击（选择文件）
function handleFileClick(file: FileItem) {
  if (file.id) {
    if (selectedFiles.value.has(file.id)) {
      selectedFiles.value.delete(file.id);
    } else {
      selectedFiles.value.add(file.id);
    }
  }
}

// 处理文件双击（打开文件或进入文件夹）
function handleFileDoubleClick(file: FileItem) {
  // 双击时清空选择状态
  selectedFiles.value.clear();
  
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
    selectedFile.value = file;
    showFilePreview.value = true;
  }
}

// 关闭文件预览
function closeFilePreview() {
  showFilePreview.value = false;
}

// 预览选中的文件（移动端专用）
function previewSelectedFile() {
  if (selectedFiles.value.size === 1) {
    const fileId = Array.from(selectedFiles.value)[0];
    const file = filteredFiles.value.find(f => f.id === fileId);
    if (file && file.type !== 'folder') {
      selectedFile.value = file;
      showFilePreview.value = true;
    }
  }
}

// 进入选中的文件夹
function enterSelectedFolder() {
  if (selectedFiles.value.size === 1) {
    const folderId = Array.from(selectedFiles.value)[0];
    const folder = filteredFiles.value.find(f => f.id === folderId && f.type === 'folder');
    if (folder) {
      handleFileDoubleClick(folder);
    }
  }
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

// 批量下载选中的文件
function downloadSelectedFiles() {
  if (selectedFiles.value.size === 0) {
    ElMessage.warning('请先选择要下载的文件');
    return;
  }

  const selectedFileItems = filteredFiles.value.filter(file => 
    file.id && selectedFiles.value.has(file.id) && file.type !== 'folder'
  );

  if (selectedFileItems.length === 0) {
    ElMessage.warning('没有可下载的文件');
    return;
  }

  // 逐个下载所有选中的文件
  selectedFileItems.forEach(file => {
    if (file.id) {
      const downloadUrl = getDownloadUrl(file.id);
      window.open(downloadUrl, '_blank');
    }
  });

  // 下载完成后清空选择
  selectedFiles.value.clear();
}

// 批量分享选中的文件
function shareSelectedFiles() {
  if (selectedFiles.value.size === 0) {
    ElMessage.warning('请先选择要分享的文件');
    return;
  }

  const selectedFileItems = filteredFiles.value.filter(file => 
    file.id && selectedFiles.value.has(file.id)
  );

  if (selectedFileItems.length === 0) {
    ElMessage.warning('没有选择有效的文件');
    return;
  }

  if (selectedFileItems.length > 1) {
    ElMessage.warning('暂不支持批量分享多个文件');
    return;
  }

  // 分享单个文件
  const file = selectedFileItems[0];
  selectedFile.value = file;
  showShareLinkPopup.value = true;
  
  // 分享后清空选择
  selectedFiles.value.clear();
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

  let successCount = 0;
  let failCount = 0;
  const totalFiles = files.length;
  // 清空新上传文件ID列表
  newUploadedFileIds.value = [];

  // 显示上传进度
  uploadProgress.value = 0;
  
  for (let i = 0; i < totalFiles; i++) {
    const file = files[i];
    
    try {
      // 所有文件都使用分片上传接口
      await uploadLargeFile(file, i);
      successCount++;
    } catch (error) {
      // 单个文件上传失败，记录但不中断其他文件上传
      console.error(`文件 "${file.name}" 上传失败:`, error);
      // 更新文件状态为失败
      uploadModalRef.value?.updateFileStatus(i, 'error', error instanceof Error ? error.message : '上传失败');
      failCount++;
    }
    
    // 更新总进度
    uploadProgress.value = Math.round(((i + 1) / totalFiles) * 100);
  }

  // 所有文件上传完成后，只显示一个总结提示
  if (failCount === 0) {
    ElMessage.success(`全部 ${successCount} 个文件上传成功`);
  } else if (successCount === 0) {
    ElMessage.error(`全部 ${failCount} 个文件上传失败`);
  } else {
    ElMessage.warning(`上传完成: ${successCount} 个成功, ${failCount} 个失败`);
  }

  // 重新获取文件列表
  fetchFiles(currentParentId.value);
  
  // 不再自动清除高亮效果
  // 用户可以通过刷新页面或导航到其他目录来清除高亮
}

// 处理大文件分片上传
async function uploadLargeFile(file: File, fileIndex: number) {
  // 从系统配置获取分片大小（KB转字节）
  let chunkSize = 5120 * 1024; // 默认5120KB（5MB）
  try {
    const config = await getSystemConfig();
    if (config && config.webdav_chunk_size) {
      // 配置中的webdav_chunk_size单位是KB，需要转换为字节
      chunkSize = config.webdav_chunk_size * 1024; // KB转字节
    }
  } catch (error) {
    console.warn('获取分片大小配置失败，使用默认值:', error);
  }
  
  const totalChunks = Math.ceil(file.size / chunkSize);
  
  try {
    // 获取是否启用断点续传的设置
    const enableResumeUpload = (uploadModalRef.value as any)?.enableResumeUpload ?? true;
    
    // 如果不启用断点续传，直接使用普通上传
    if (!enableResumeUpload) {
      const response = await uploadFile(currentParentId.value, file);
      if (response && response.id) {
        newUploadedFileIds.value.push(response.id.toString());
      }
      uploadModalRef.value?.updateFileStatus(fileIndex, 'success', undefined, 1, 1);
      return;
    }
    
    // 网盘项目不限制文件大小，通过分片上传支持任意大小的文件
    
    // 生成基于文件名的稳定uploadId，支持断点续传
    const stableUploadId = `upload_${currentParentId.value || 'root'}_${file.name}_${file.size}`;
    
    // 尝试获取已存在的上传会话
    let uploadId = stableUploadId;
    let initResponse;
    
    try {
      const chunksResponse = await getUploadedChunks(stableUploadId);
      if (chunksResponse.totalChunks > 0) {
        initResponse = {
          uploadId: stableUploadId,
          chunkSize: chunkSize,
          totalChunks: totalChunks,
          fileName: file.name,
          fileSize: file.size,
          parentId: currentParentId.value
        };
      } else {
        // 会话存在但没有分片，视为新会话
        initResponse = await initChunkUpload(currentParentId.value, file.name, file.size, chunkSize, stableUploadId);
        uploadId = initResponse.uploadId;
      }
    } catch (error: any) {
      // 会话不存在或其他错误，静默创建新的上传会话
      const errorMessage = error?.response?.data?.error || error?.message || '';
      if (!errorMessage.includes('上传会话不存在') && !errorMessage.includes('会话不存在')) {
        console.warn('获取上传会话信息失败:', error);
      }
      initResponse = await initChunkUpload(currentParentId.value, file.name, file.size, chunkSize, stableUploadId);
      uploadId = initResponse.uploadId;
    }
    
    // 获取已上传的分片列表（断点续传）
    let uploadedChunks: number[] = [];
    try {
      const chunksResponse = await getUploadedChunks(uploadId);
      uploadedChunks = chunksResponse.chunks || [];
    } catch (error: any) {
      // 静默处理会话不存在的错误，这是预期行为
      const errorMessage = error?.response?.data?.error || error?.message || '';
      if (!errorMessage.includes('上传会话不存在') && !errorMessage.includes('会话不存在')) {
        console.warn('获取已上传分片列表失败:', error);
      }
      uploadedChunks = [];
    }
    
    // 计算需要上传的分片
    const pendingChunks = [];
    for (let i = 0; i < totalChunks; i++) {
      if (!uploadedChunks.includes(i)) {
        pendingChunks.push(i);
      }
    }
    
    // 如果所有分片都已上传，直接完成上传
    if (pendingChunks.length === 0) {
      const completeResponse = await completeChunkUpload(uploadId);
      if (completeResponse && completeResponse.id) {
        newUploadedFileIds.value.push(completeResponse.id.toString());
      }
      uploadModalRef.value?.updateFileStatus(fileIndex, 'success');
      return;
    }
    
    // 更新上传状态
    const uploadedCount = uploadedChunks.length;
    uploadModalRef.value?.updateFileStatus(fileIndex, 'uploading', 
      `断点续传 (${uploadedCount}/${totalChunks}已上传，剩余${pendingChunks.length}个)`,
      uploadedCount, totalChunks);
    
    // 上传未完成的分片
    let completedCount = 0;
    
    // 获取用户配置的重试次数
    const maxRetries = (uploadModalRef.value as any)?.maxRetries ?? 0;
    
    // 分批处理，避免内存溢出
    // 优化批次大小：默认5个分片，根据文件大小智能调整
    const getOptimalBatchSize = (fileSize: number) => {
      if (fileSize <= 10 * 1024 * 1024) { // 小文件(<=10MB)：5-8个分片
        return Math.min(8, Math.max(5, Math.ceil(fileSize / (512 * 1024))));
      } else if (fileSize <= 100 * 1024 * 1024) { // 中等文件(10-100MB)：3-5个分片
        return Math.min(5, Math.max(3, Math.ceil(fileSize / (2 * 1024 * 1024))));
      } else { // 大文件(>100MB)：1-3个分片
        return Math.min(3, Math.max(1, Math.ceil(fileSize / (10 * 1024 * 1024))));
      }
    };
    
    const BATCH_SIZE = getOptimalBatchSize(file.size); // 智能批次大小
    const CHUNK_DELAY = 10; // 分片间延迟10ms，减少CPU占用
    
    // 带重试的分片上传函数
    const uploadChunkWithRetry = async (chunkIndex: number, chunk: Blob) => {
      let retryCount = 0;
      const isInfiniteRetry = maxRetries === 0;
      
      while (isInfiniteRetry || retryCount < maxRetries) {
        try {
          await uploadChunk(uploadId, chunkIndex, chunk);
          return; // 上传成功，退出重试循环
        } catch (error) {
          retryCount++;
          
          if (!isInfiniteRetry && retryCount >= maxRetries) {
            // 超过最大重试次数，抛出错误
            throw new Error(`分片 ${chunkIndex + 1} 上传失败，已重试 ${maxRetries} 次: ${error instanceof Error ? error.message : '未知错误'}`);
          }
          
          // 记录重试信息
          const retryInfo = isInfiniteRetry 
            ? `第 ${retryCount} 次重试` 
            : `第 ${retryCount}/${maxRetries} 次重试`;
          console.warn(`分片 ${chunkIndex + 1} 上传失败，${3}秒后${retryInfo}:`, error);
          
          // 更新状态显示重试信息
          uploadModalRef.value?.updateFileStatus(fileIndex, 'uploading', 
            `分片 ${chunkIndex + 1} 上传失败，${3}秒后重试 (${retryInfo})`,
            uploadedCount + completedCount, totalChunks);
          
          // 等待3秒后重试
          await new Promise(resolve => setTimeout(resolve, 3000));
        }
      }
    };
    
    // 分批处理分片上传
    for (let batchStart = 0; batchStart < pendingChunks.length; batchStart += BATCH_SIZE) {
      const batchEnd = Math.min(batchStart + BATCH_SIZE, pendingChunks.length);
      const batchChunks = pendingChunks.slice(batchStart, batchEnd);
      
      // 处理当前批次
      await Promise.all(batchChunks.map(async (chunkIndex) => {
        // 使用requestIdleCallback或setTimeout延迟处理，避免阻塞主线程
        await new Promise(resolve => {
          setTimeout(() => {
            const start = chunkIndex * chunkSize;
            const end = Math.min(start + chunkSize, file.size);
            const chunk = file.slice(start, end);
            resolve(chunk);
          }, CHUNK_DELAY);
        });
        
        const start = chunkIndex * chunkSize;
        const end = Math.min(start + chunkSize, file.size);
        const chunk = file.slice(start, end);
        
        await uploadChunkWithRetry(chunkIndex, chunk);
        completedCount++;
        
        // 更新进度
        const totalUploaded = uploadedCount + completedCount;
        const progress = Math.round((totalUploaded / totalChunks) * 100);
        uploadModalRef.value?.updateFileStatus(fileIndex, 'uploading', 
          `断点续传中 (${totalUploaded}/${totalChunks}) ${progress}%`,
          totalUploaded, totalChunks);
        
        // 每上传完一个分片，让出控制权给浏览器
        await new Promise(resolve => setTimeout(resolve, 1));
      }));
      
      // 智能批次间延迟：根据批次大小和文件大小动态调整
      if (batchEnd < pendingChunks.length) {
        const dynamicDelay = Math.max(30, Math.min(100, BATCH_SIZE * 10)); // 30-100ms动态延迟
        await new Promise(resolve => setTimeout(resolve, dynamicDelay));
      }
    }
    
    // 完成分片上传
    const completeResponse = await completeChunkUpload(uploadId);
    
    // 如果上传成功，记录文件ID
    if (completeResponse && completeResponse.id) {
      newUploadedFileIds.value.push(completeResponse.id.toString());
    }
    
    // 更新状态为成功
    uploadModalRef.value?.updateFileStatus(fileIndex, 'success');
  } catch (error) {
    console.error('分片上传失败:', error);
    // 上传失败，不自动取消，允许用户重试
    throw error;
  }
}

// 处理重试上传
async function handleRetryUpload(failedFiles: File[]) {
  if (!failedFiles.length) return;
  
  await handleUploadFiles(failedFiles);
}

// 修改closeUploadModal函数
function closeUploadModal() {
  // 始终允许关闭上传弹窗
  showUploadModal.value = false;
}

onMounted(() => {
  // 初始化时获取文件列表
  fetchFiles();
})

onUnmounted(() => {
  // 清理工作
})
</script>

<style lang="less">
.web-drive-container {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 40px);
  /* 减去padding的高度 */
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

    @media (max-width: 768px) {
      flex-direction: column;
      gap: 15px;
      margin-bottom: 15px;
    }

    .toolbar-left {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;

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

        @media (max-width: 768px) {
          padding: 10px 16px;
          font-size: 15px;
          min-height: 44px;
        }

        @media (max-width: 480px) {
          padding: 12px 16px;
          font-size: 16px;
          flex: 1;
          justify-content: center;
          min-height: 48px;
        }
      }

      @media (max-width: 480px) {
        gap: 8px;
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

      .mobile-preview-btn {
        display: none;

        @media (max-width: 768px) {
          display: flex;
          flex: 1;
          min-width: 100%;
          margin-top: 8px;
          justify-content: center;
        }
      }

      .enter-folder-btn {
        display: none;

        @media (max-width: 768px) {
          display: flex;
          flex: 1;
          min-width: 100%;
          margin-top: 8px;
          justify-content: center;
          background-color: #2a8aff;
          color: white;
          border: none;

          &:hover {
            background-color: #1a7aef;
          }
        }
      }

      @media (max-width: 768px) {
        .btn-primary,
        .btn-outline {
          &:nth-child(n+3):nth-last-child(-n+4) {
            flex: 1;
            min-width: 100%;
            margin: 4px 0;
          }
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

          @media (max-width: 768px) {
            width: 100%;
            padding: 10px 12px 10px 40px;
            font-size: 15px;
            min-height: 44px;
          }

          @media (max-width: 480px) {
            padding: 12px 12px 12px 40px;
            font-size: 16px;
            min-height: 48px;
          }
        }

        @media (max-width: 768px) {
          .search-icon {
            left: 12px;
            width: 18px;
            height: 18px;
          }
        }
      }

      @media (max-width: 768px) {
        width: 100%;
      }
    }
  }

  .file-container {
      flex: 1;
      overflow: visible;
      display: flex;
      flex-direction: column;
      min-height: 400px;
      background-color: #ffffff;
      padding: 10px;

      .file-container-inner {
        display: flex;
        flex-direction: column;
        overflow: visible;
      }

      .file-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
        gap: 16px;
        flex: 1;
        background-color: #ffffff;
        overflow: visible;
        padding: 10px;

        @media (max-width: 768px) {
          grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
          gap: 12px;
          padding: 8px;
        }

        @media (max-width: 480px) {
          grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
          gap: 8px;
          padding: 5px;
        }

      .file-item {
        cursor: pointer;
        border-radius: 8px;
        padding: 15px;
        transition: all 0.3s ease;
        position: relative;
        height: 140px;
        display: flex;
        background-color: #ffffff;
        border: 2px solid transparent;

        &:hover {
          background-color: #f5f5f5;
          transform: translateY(-3px);
          box-shadow: 0 5px 15px rgba(0, 0, 0, 0.05);
        }

        @media (max-width: 768px) {
          height: 120px;
          padding: 12px;
          border-radius: 6px;
        }

        @media (max-width: 480px) {
          height: 100px;
          padding: 10px;
          border-radius: 4px;
        }

        &.selected {
          background-color: #e6f0ff;
          border-color: #2a8aff;
          box-shadow: 0 5px 15px rgba(42, 138, 255, 0.2);
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

        &.new-uploaded-file {
          animation: highlight-pulse 2s ease-in-out infinite;
          background-color: rgba(59, 130, 246, 0.1);
          border: 1px solid rgba(59, 130, 246, 0.4);
          box-shadow: 0 0 10px rgba(59, 130, 246, 0.2);
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

            .folder-icon,
            .file-icon {
              width: 40px;
              height: 40px;
              transition: transform 0.3s ease;
            }

            @media (max-width: 768px) {
              width: 50px;
              height: 50px;
              
              .folder-icon,
              .file-icon {
                width: 35px;
                height: 35px;
              }
            }

            @media (max-width: 480px) {
              width: 40px;
              height: 40px;
              
              .folder-icon,
              .file-icon {
                width: 30px;
                height: 30px;
              }
            }

            .folder-icon {
              color: #2a8aff;
            }

            .file-icon {
              color: #2a8aff;

              &.image-icon {
                color: #4CAF50;
                /* 绿色 */
              }

              &.video-icon {
                color: #FF5722;
                /* 橙红色 */
              }

              &.audio-icon {
                color: #9C27B0;
                /* 紫色 */
              }

              &.code-icon {
                color: #607D8B;
                /* 蓝灰色 */
              }

              &.pdf-icon {
                color: #F44336;
                /* 红色 */
              }

              &.archive-icon {
                color: #795548;
                /* 棕色 */
              }

              &.spreadsheet-icon {
                color: #4CAF50;
                /* 绿色 */
              }

              &.presentation-icon {
                color: #FF9800;
                /* 橙色 */
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

              @media (max-width: 768px) {
                font-size: 13px;
              }

              @media (max-width: 480px) {
                font-size: 12px;
              }
            }

            .file-details {
              height: 30px;

              .file-size,
              .file-modified {
                font-size: 12px;
                color: #999;
                margin: 0;
                line-height: 1.2;

                @media (max-width: 480px) {
                  font-size: 11px;
                }
              }
            }

            @media (max-width: 768px) {
              height: 45px;
            }

            @media (max-width: 480px) {
              height: 40px;
            }
          }
        }
      }
    }
  }

  .context-menu {
    position: fixed;
    background: white;
    border-radius: 8px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
    z-index: 1000;
    min-width: 180px;
    max-width: 250px;

    @media (max-width: 768px) {
      border-radius: 12px;
      min-width: 200px;
      box-shadow: 0 8px 30px rgba(0, 0, 0, 0.2);
    }

    ul {
      list-style: none;
      padding: 0;
      margin: 0;

      li {
        padding: 12px 16px;
        display: flex;
        align-items: center;
        gap: 12px;
        cursor: pointer;
        font-size: 14px;

        &:hover {
          background-color: #f5f5f5;
        }

        &.danger {
          color: #ff4d4f;
        }

        .icon-xs {
          width: 16px;
          height: 16px;
        }

        @media (max-width: 768px) {
          padding: 16px 20px;
          font-size: 16px;
          min-height: 48px;

          .icon-xs {
            width: 18px;
            height: 18px;
          }
        }

        @media (max-width: 480px) {
          padding: 18px 22px;
          font-size: 17px;
          min-height: 52px;
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
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
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

  @media (max-width: 768px) {
    width: 95%;
    max-width: 95%;
    border-radius: 12px;
  }

  @media (max-width: 480px) {
    width: 100%;
    max-width: 100%;
    height: 100%;
    max-height: 100%;
    border-radius: 0;
    display: flex;
    flex-direction: column;
  }

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

      @media (max-width: 768px) {
        font-size: 20px;
      }

      @media (max-width: 480px) {
        font-size: 22px;
      }
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

      @media (max-width: 768px) {
        font-size: 24px;
        padding: 8px;
      }

      @media (max-width: 480px) {
        font-size: 28px;
        padding: 12px;
      }
    }

    @media (max-width: 480px) {
      padding: 20px;
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
            cursor: pointer;
            /* 改为指针，提示可以交互 */
            opacity: 0.8;
            pointer-events: none;
            /* 禁用事件，让父元素处理双击 */
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

      @media (max-width: 768px) {
        padding: 12px 14px;
        font-size: 16px;
        border-radius: 6px;
      }

      @media (max-width: 480px) {
        padding: 14px 16px;
        font-size: 17px;
        border-radius: 8px;
      }
    }

    @media (max-width: 768px) {
      padding: 24px;
    }

    @media (max-width: 480px) {
      padding: 28px;
      flex: 1;
    }
  }

  .dialog-footer {
    padding: 16px 20px;
    border-top: 1px solid #eee;
    display: flex;
    justify-content: flex-end;
    gap: 10px;

    @media (max-width: 480px) {
      padding: 20px;
      margin-top: auto;
    }

    button {
      padding: 8px 16px;
      border-radius: 4px;
      cursor: pointer;
      font-size: 14px;

      @media (max-width: 768px) {
        padding: 10px 20px;
        font-size: 16px;
        min-height: 44px;
      }

      @media (max-width: 480px) {
        padding: 12px 24px;
        font-size: 17px;
        min-height: 48px;
        flex: 1;
        max-width: 120px;
      }

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

// 修改高亮样式，使其更适合长时间显示
.file-item.new-uploaded-file {
  background-color: rgba(59, 130, 246, 0.05);
  border: 1px solid rgba(59, 130, 246, 0.3);
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.1);
  position: relative;
  
  &::after {
    content: "新";
    position: absolute;
    top: -10px; /* 调整标记位置，使其显示在元素外 */
    right: -10px;
    background-color: #3b82f6;
    color: white;
    font-size: 12px;
    font-weight: bold;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
    z-index: 10; /* 使用更高的z-index确保显示在最上层 */
  }
}

@keyframes highlight-pulse {
  0% {
    box-shadow: 0 0 5px rgba(59, 130, 246, 0.2);
  }
  50% {
    box-shadow: 0 0 15px rgba(59, 130, 246, 0.5);
  }
  100% {
    box-shadow: 0 0 5px rgba(59, 130, 246, 0.2);
  }
}

// 添加上传弹窗样式
.upload-modal-container {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none; /* 防止容器本身阻止点击事件 */
}

.upload-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(2px);
  z-index: 1101;
  pointer-events: auto; /* 允许蒙版接收点击事件 */
}
</style>
