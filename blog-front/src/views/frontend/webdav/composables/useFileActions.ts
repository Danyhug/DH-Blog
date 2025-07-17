import { ref, type Ref } from 'vue';
import { ElMessage } from 'element-plus';
import {
  createFolder as apiCreateFolder,
  renameFile as apiRenameFile,
  deleteFile as apiDeleteFile,
  uploadFile as apiUploadFile,
  getDownloadUrl,
  type FileInfo
} from '@/api/file';
import type { FileItem } from '../utils/types/file';
import type { ComponentPublicInstance } from 'vue';

type UploadModalInstance = ComponentPublicInstance & {
  updateFileStatus: (fileIndex: number, status: 'success' | 'error', error?: string) => void;
};

// useFileActions接收一个包含响应式引用和回调函数的对象
export function useFileActions({
  currentParentId,
  fetchFiles
}: {
  currentParentId: Ref<string>;
  fetchFiles: (parentId: string) => Promise<void>;
}) {
  const uploadProgress = ref(0);
  const newUploadedFileIds = ref<string[]>([]);
  const uploadModalRef = ref<UploadModalInstance | null>(null);

  // 创建新文件夹
  const createFolder = async (folderName: string) => {
    if (!folderName.trim()) {
      ElMessage.warning('文件夹名称不能为空');
      return false;
    }
    try {
      await apiCreateFolder(currentParentId.value, folderName);
      ElMessage.success('文件夹创建成功');
      await fetchFiles(currentParentId.value);
      return true;
    } catch (error) {
      console.error('创建文件夹失败:', error);
      ElMessage.error('创建文件夹失败');
      return false;
    }
  };

  // 重命名文件
  const renameFile = async (file: FileItem, newName: string) => {
    if (!file.id || !newName.trim() || newName === file.name) {
      return false;
    }
    try {
      await apiRenameFile(file.id, newName);
      ElMessage.success('重命名成功');
      await fetchFiles(currentParentId.value);
      return true;
    } catch (error) {
      console.error('重命名失败:', error);
      ElMessage.error('重命名失败');
      return false;
    }
  };
  
  // 删除单个文件
  const deleteFile = async (file: FileItem) => {
    if (!file.id) return false;
    try {
      await apiDeleteFile(file.id);
      ElMessage.success('删除成功');
      await fetchFiles(currentParentId.value);
      return true;
    } catch (error) {
      console.error('删除失败:', error);
      ElMessage.error('删除失败');
      return false;
    }
  };

  // 批量删除文件
  const deleteFiles = async (files: FileItem[]) => {
    if (files.length === 0) {
      ElMessage.warning('没有选择要删除的文件');
      return;
    }
    
    let successCount = 0;
    for (const file of files) {
      if (file.id) {
        try {
          await apiDeleteFile(file.id);
          successCount++;
        } catch (error) {
          console.error(`删除文件 ${file.name} (ID: ${file.id}) 失败:`, error);
        }
      }
    }

    if (successCount > 0) {
      ElMessage.success(`成功删除 ${successCount} 个文件`);
    }
    if (successCount < files.length) {
      ElMessage.warning(`有 ${files.length - successCount} 个文件删除失败`);
    }

    await fetchFiles(currentParentId.value);
  };

  // 下载文件
  const downloadFile = (file: FileItem) => {
    if (file.type !== 'folder' && file.id) {
      const downloadUrl = getDownloadUrl(file.id);
      window.open(downloadUrl, '_blank');
      return true;
    }
    ElMessage.warning('文件夹无法下载');
    return false;
  };
  
  // 处理文件上传
  const handleUploadFiles = async (files: File[]) => {
    if (!files.length) return;

    let successCount = 0;
    let failCount = 0;
    const totalFiles = files.length;
    newUploadedFileIds.value = [];
    uploadProgress.value = 0;

    for (let i = 0; i < totalFiles; i++) {
      const file = files[i];
      try {
        const response = await apiUploadFile(currentParentId.value, file);
        uploadModalRef.value?.updateFileStatus(i, 'success');
        successCount++;
        if (response && response.id) {
          newUploadedFileIds.value.push(response.id.toString());
        }
      } catch (error) {
        console.error(`文件 "${file.name}" 上传失败:`, error);
        const errorMessage = error instanceof Error ? error.message : '上传失败';
        uploadModalRef.value?.updateFileStatus(i, 'error', errorMessage);
        failCount++;
      }
      uploadProgress.value = Math.round(((i + 1) / totalFiles) * 100);
    }

    if (failCount === 0) {
      ElMessage.success(`全部 ${successCount} 个文件上传成功`);
    } else if (successCount === 0) {
      ElMessage.error(`全部 ${failCount} 个文件上传失败`);
    } else {
      ElMessage.warning(`上传完成: ${successCount} 个成功, ${failCount} 个失败`);
    }

    await fetchFiles(currentParentId.value);
  };

  // 重试上传
  const handleRetryUpload = async (failedFiles: File[]) => {
    if (!failedFiles.length) return;
    await handleUploadFiles(failedFiles);
  };

  return {
    uploadProgress,
    newUploadedFileIds,
    uploadModalRef,
    createFolder,
    renameFile,
    deleteFile,
    deleteFiles,
    downloadFile,
    handleUploadFiles,
    handleRetryUpload
  };
} 