import { ref, computed, type Ref } from 'vue';
import { ElMessage } from 'element-plus';
import type { FileItem } from '../utils/types/file';

export function useSelection(
  filteredFiles: Ref<Readonly<FileItem[]>>,
  handleFileClick: (file: FileItem, event: MouseEvent) => void,
  downloadFile: (file: FileItem) => boolean,
  deleteFiles: (files: FileItem[]) => Promise<void>
) {
  const isSelecting = ref(false);
  const selectionStart = ref({ x: 0, y: 0 });
  const selectionEnd = ref({ x: 0, y: 0 });
  const selectedFileIds = ref<string[]>([]);
  const isDraggingWithModifier = ref(false);
  const isProcessingBatchAction = ref(false);

  const selectionBoxStyle = computed(() => {
    const left = Math.min(selectionStart.value.x, selectionEnd.value.x);
    const top = Math.min(selectionStart.value.y, selectionEnd.value.y);
    const width = Math.abs(selectionEnd.value.x - selectionStart.value.x);
    const height = Math.abs(selectionEnd.value.y - selectionStart.value.y);
    return {
      left: `${left}px`, top: `${top}px`, width: `${width}px`, height: `${height}px`,
      display: isSelecting.value ? 'block' : 'none'
    };
  });

  const updateSelectedFiles = () => {
    if (!isSelecting.value) return;
    const selBox = {
      left: Math.min(selectionStart.value.x, selectionEnd.value.x),
      top: Math.min(selectionStart.value.y, selectionEnd.value.y),
      right: Math.max(selectionStart.value.x, selectionEnd.value.x),
      bottom: Math.max(selectionStart.value.y, selectionEnd.value.y)
    };
    const newSelectedIds: string[] = [];
    const fileItems = document.querySelectorAll('.file-item');
    const containerRect = document.querySelector('.file-container')?.getBoundingClientRect();
    if (!containerRect) return;

    fileItems.forEach((item, index) => {
      const file = filteredFiles.value[index];
      if (!file || file.type === 'folder') return;
      const itemRect = item.getBoundingClientRect();
      const itemBox = {
        left: itemRect.left - containerRect.left,
        top: itemRect.top - containerRect.top,
        right: itemRect.left - containerRect.left + itemRect.width,
        bottom: itemRect.top - containerRect.top + itemRect.height,
      };
      if (itemBox.right > selBox.left && itemBox.left < selBox.right &&
          itemBox.bottom > selBox.top && itemBox.top < selBox.bottom) {
        if (file.id && !selectedFileIds.value.includes(file.id)) {
          newSelectedIds.push(file.id);
        }
      }
    });

    if (isDraggingWithModifier.value) {
      const currentSelection = new Set([...selectedFileIds.value, ...newSelectedIds]);
      selectedFileIds.value = Array.from(currentSelection);
    } else {
      selectedFileIds.value = newSelectedIds;
    }
  };

  const handleMouseDown = (e: MouseEvent) => {
    if (e.button !== 0 || (e.target as HTMLElement).closest('.file-item')) return;
    e.preventDefault();
    isDraggingWithModifier.value = e.ctrlKey || e.metaKey;
    if (!isDraggingWithModifier.value) {
      selectedFileIds.value = [];
    }
    isSelecting.value = true;
    const container = (e.currentTarget as HTMLElement)?.getBoundingClientRect();
    if (!container) return;
    selectionStart.value = { x: e.clientX - container.left, y: e.clientY - container.top };
    selectionEnd.value = { ...selectionStart.value };
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  };

  const handleMouseMove = (e: MouseEvent) => {
    if (!isSelecting.value) return;
    const container = document.querySelector('.file-container')?.getBoundingClientRect();
    if (!container) return;
    selectionEnd.value = {
      x: Math.max(0, Math.min(e.clientX - container.left, container.width)),
      y: Math.max(0, Math.min(e.clientY - container.top, container.height))
    };
    updateSelectedFiles();
  };

  const handleMouseUp = () => {
    isSelecting.value = false;
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  };
  
  const handleFileItemClick = (file: FileItem, event: MouseEvent) => {
    if (event.button === 2) return;
    if (file.type === 'folder') {
      handleFileClick(file, event);
      return;
    }
    if (event.ctrlKey || event.metaKey) {
      event.preventDefault();
      if (file.id) {
        const index = selectedFileIds.value.indexOf(file.id);
        if (index > -1) {
          selectedFileIds.value.splice(index, 1);
        } else {
          selectedFileIds.value.push(file.id);
        }
      }
    } else if (selectedFileIds.value.length > 0 && file.id && selectedFileIds.value.includes(file.id)) {
      event.preventDefault();
    } else {
      selectedFileIds.value = [];
      handleFileClick(file, event);
    }
  };
  
  // 批量下载
  const handleBatchDownload = async (event: MouseEvent) => {
    event.stopPropagation();
    event.preventDefault();
    if (isProcessingBatchAction.value) return;
    isProcessingBatchAction.value = true;
    
    if (selectedFileIds.value.length === 0) {
        ElMessage.warning('请先选择要下载的文件');
        isProcessingBatchAction.value = false;
        return;
    }
    let downloadCount = 0;
    for (const fileId of selectedFileIds.value) {
        const file = filteredFiles.value.find((f: FileItem) => f.id === fileId);
        if (file && downloadFile(file)) {
            downloadCount++;
            await new Promise(resolve => setTimeout(resolve, 300));
        }
    }
    if (downloadCount > 0) {
        ElMessage.success(`已开始下载 ${downloadCount} 个文件`);
    }

    isProcessingBatchAction.value = false;
  };

  // 批量删除
  const handleBatchDelete = async (event: MouseEvent) => {
    event.stopPropagation();
    event.preventDefault();
    if (isProcessingBatchAction.value) return;
    
    const filesToDelete = filteredFiles.value.filter(f => selectedFileIds.value.includes(f.id!));
    if (filesToDelete.length === 0) {
      return ElMessage.warning('请先选择要删除的文件');
    }

    if (!confirm(`确定要删除选中的 ${filesToDelete.length} 个文件吗？`)) return;

    isProcessingBatchAction.value = true;
    await deleteFiles(filesToDelete);
    selectedFileIds.value = [];
    isProcessingBatchAction.value = false;
  };

  // 取消选择
  const cancelSelection = (event: MouseEvent) => {
    event.stopPropagation();
    event.preventDefault();
    selectedFileIds.value = [];
  };

  return {
    isSelecting,
    selectionStart,
    selectionEnd,
    selectedFileIds,
    isDraggingWithModifier,
    isProcessingBatchAction,
    selectionBoxStyle,
    handleMouseDown,
    handleMouseMove,
    handleMouseUp,
    handleFileItemClick,
    handleBatchDownload,
    handleBatchDelete,
    cancelSelection,
  };
} 