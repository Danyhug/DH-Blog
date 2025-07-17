import { ref, nextTick, type Ref } from 'vue';
import type { FileItem } from '../utils/types/file';

export function useDialogs(
  renameFileAction: (file: FileItem, newName: string) => Promise<boolean>,
  createFolderAction: (folderName: string) => Promise<boolean>
) {
  // 对话框状态
  const showNewFolderDialog = ref(false);
  const showRenameDialog = ref(false);

  // 新建文件夹相关
  const newFolderName = ref('新建文件夹');
  const folderNameInput = ref<HTMLInputElement | null>(null);

  // 重命名相关
  const newFileName = ref('');
  const fileToRename = ref<FileItem | null>(null);
  const fileNameWithoutExt = ref('');
  const fileExtension = ref('');
  const editingExtension = ref(false);
  const fileNameInput = ref<HTMLInputElement | null>(null);
  const extensionInput = ref<HTMLInputElement | null>(null);

  // 打开新建文件夹对话框
  const openNewFolderDialog = () => {
    showNewFolderDialog.value = true;
    newFolderName.value = '新建文件夹';
    nextTick(() => {
      folderNameInput.value?.focus();
      folderNameInput.value?.select();
    });
  };

  // 确认新建文件夹
  const confirmNewFolder = async () => {
    if (await createFolderAction(newFolderName.value)) {
      showNewFolderDialog.value = false;
    }
  };

  // 打开重命名对话框
  const openRenameDialog = (file: FileItem) => {
    if (!file.id) return;
    fileToRename.value = file;
    if (file.type !== 'folder') {
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
    nextTick(() => {
      if (file.type !== 'folder') {
        fileNameInput.value?.focus();
        fileNameInput.value?.select();
      } else {
        folderNameInput.value?.focus();
        folderNameInput.value?.select();
      }
    });
  };

  // 确认重命名
  const confirmRename = async () => {
    if (fileToRename.value?.type !== 'folder') {
      newFileName.value = fileNameWithoutExt.value + fileExtension.value;
    }
    if (fileToRename.value) {
      if (await renameFileAction(fileToRename.value, newFileName.value)) {
        showRenameDialog.value = false;
      }
    } else {
      showRenameDialog.value = false;
    }
  };

  // 取消对话框
  const cancelDialog = () => {
    showNewFolderDialog.value = false;
    showRenameDialog.value = false;
  };

  // 切换扩展名编辑状态
  const toggleExtensionEdit = () => {
    editingExtension.value = !editingExtension.value;
    if (editingExtension.value) {
      nextTick(() => {
        const extInput = document.querySelector('.extension-input') as HTMLInputElement;
        extInput?.focus();
        extInput?.select();
      });
    }
  };

  // 启用扩展名编辑
  const enableExtensionEdit = (event: MouseEvent) => {
    event.stopPropagation();
    if (!editingExtension.value) {
      editingExtension.value = true;
      nextTick(() => {
        if (extensionInput.value) {
          extensionInput.value.disabled = false;
          extensionInput.value.focus();
          extensionInput.value.select();
        }
      });
    }
  };

  return {
    showNewFolderDialog,
    showRenameDialog,
    newFolderName,
    newFileName,
    fileToRename,
    fileNameWithoutExt,
    fileExtension,
    editingExtension,
    folderNameInput,
    fileNameInput,
    extensionInput,
    openNewFolderDialog,
    confirmNewFolder,
    openRenameDialog,
    confirmRename,
    cancelDialog,
    toggleExtensionEdit,
    enableExtensionEdit,
  };
} 