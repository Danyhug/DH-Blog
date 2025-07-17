import { ref, computed } from 'vue';
import { ElMessage } from 'element-plus';
import type { FileItem } from '../utils/types/file';
import { listFiles, FileInfo } from '@/api/file';
import {
  FolderIcon, FileIcon, FileTextIcon, ImageIcon, VideoIcon, MusicIcon,
  FileCodeIcon, FileZipIcon, FilePdfIcon, FileSpreadsheetIcon, FilePresentationIcon
} from '../utils/icons';

// 路径导航历史记录的接口定义
export interface PathSegment {
  id: string;
  name: string;
}

export function useFileManagement() {
  const isLoading = ref(false);
  const apiFiles = ref<FileInfo[]>([]);
  const currentPath = ref('');
  const currentParentId = ref('');
  const pathSegments = ref<PathSegment[]>([]);
  const searchQuery = ref('');

  // 格式化文件大小
  const formatSize = (size: number, isFolder: boolean): string => {
    if (isFolder) return '文件夹';
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

  // 将API返回的文件数据转换为组件使用的格式
  const convertedFiles = computed<FileItem[]>(() => {
    return apiFiles.value.map(file => {
      let icon;
      let fileType: FileItem['type'] = file.is_folder ? 'folder' : 'file';

      if (file.is_folder) {
        icon = FolderIcon;
      } else if (file.mimeType) {
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
        } else if (['zip', 'compressed', 'archive', 'x-tar', 'x-rar'].some(t => file.mimeType!.includes(t))) {
          icon = FileZipIcon;
          fileType = 'archive';
        } else if (['excel', 'spreadsheet', 'csv'].some(t => file.mimeType!.includes(t))) {
          icon = FileSpreadsheetIcon;
          fileType = 'spreadsheet';
        } else if (['powerpoint', 'presentation'].some(t => file.mimeType!.includes(t))) {
          icon = FilePresentationIcon;
          fileType = 'presentation';
        } else if (['javascript', 'json', 'html', 'css', 'xml', 'text/plain', 'text/markdown'].some(t => file.mimeType!.includes(t)) || file.mimeType.startsWith('text/')) {
          icon = FileCodeIcon;
          fileType = 'text';
        } else {
          icon = FileTextIcon;
        }
      } else {
        const extension = file.name.split('.').pop()?.toLowerCase();
        if (extension) {
          if (['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'].includes(extension)) {
            icon = ImageIcon; fileType = 'image';
          } else if (['mp4', 'webm', 'avi', 'mov', 'wmv', 'flv', 'mkv'].includes(extension)) {
            icon = VideoIcon; fileType = 'video';
          } else if (['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a'].includes(extension)) {
            icon = MusicIcon; fileType = 'audio';
          } else if (extension === 'pdf') {
            icon = FilePdfIcon; fileType = 'pdf';
          } else if (['zip', 'rar', '7z', 'tar', 'gz', 'bz2'].includes(extension)) {
            icon = FileZipIcon; fileType = 'archive';
          } else if (['xls', 'xlsx', 'csv', 'ods'].includes(extension)) {
            icon = FileSpreadsheetIcon; fileType = 'spreadsheet';
          } else if (['ppt', 'pptx', 'odp'].includes(extension)) {
            icon = FilePresentationIcon; fileType = 'presentation';
          } else if (['txt', 'md', 'markdown', 'text', 'log', 'rtf', 'js', 'ts', 'html', 'css', 'xml', 'json', 'py', 'java', 'c', 'cpp', 'go', 'php', 'rb', 'sh', 'bat', 'ps1', 'sql', 'yaml', 'yml', 'toml', 'ini', 'conf', 'config'].includes(extension)) {
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

      return {
        id: file.id ? file.id.toString() : '',
        name: file.name,
        type: fileType,
        size: formatSize(file.size, file.is_folder),
        modified: formatDate(file.updateTime),
        icon,
        originalFile: file
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
  });

  // 获取文件列表
  const fetchFiles = async (parentId: string = '') => {
    try {
      isLoading.value = true;
      apiFiles.value = await listFiles(parentId);
    } catch (error) {
      console.error('获取文件列表失败:', error);
      ElMessage.error('获取文件列表失败');
    } finally {
      isLoading.value = false;
    }
  };

  // 导航到根目录
  const navigateToRoot = () => {
    currentPath.value = '';
    currentParentId.value = '';
    pathSegments.value = [];
    fetchFiles();
  };

  // 导航到特定路径段
  const navigateToPathSegment = (index: number) => {
    if (index < 0 || index >= pathSegments.value.length) return;
    const targetSegment = pathSegments.value[index];
    currentParentId.value = targetSegment.id;
    pathSegments.value = pathSegments.value.slice(0, index + 1);
    currentPath.value = pathSegments.value.map(segment => segment.name).join('/');
    fetchFiles(targetSegment.id);
  };

  // 导航到上一级目录
  const navigateToParent = () => {
    if (pathSegments.value.length <= 1) {
      navigateToRoot();
    } else {
      navigateToPathSegment(pathSegments.value.length - 2);
    }
  };
  
  // 处理文件点击（进入文件夹）
  const handleFolderClick = (file: FileItem) => {
    if (file.type === 'folder' && file.id) {
        currentParentId.value = file.id;
        pathSegments.value.push({ id: file.id, name: file.name });
        currentPath.value = pathSegments.value.map(s => s.name).join('/');
        fetchFiles(file.id);
    }
  }

  return {
    isLoading,
    apiFiles,
    currentPath,
    currentParentId,
    pathSegments,
    searchQuery,
    convertedFiles,
    filteredFiles,
    fetchFiles,
    navigateToRoot,
    navigateToPathSegment,
    navigateToParent,
    handleFolderClick,
  };
} 