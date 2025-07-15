import request from '@/api/axios'
import { SERVER_URL } from "@/types/Constant";

/**
 * 文件类型定义
 */
export interface FileInfo {
  id: number;
  name: string;
  is_folder: boolean; // 后端使用is_folder而不是type
  size: number;
  mimeType?: string;
  user_id: number;
  parent_id?: string;
  createTime: string; // 后端使用createTime
  updateTime: string; // 后端使用updateTime
  deletedAt?: string | null;
}

/**
 * 列出文件
 * @param parentId 父目录ID，空表示根目录
 */
export const listFiles = (parentId?: string): Promise<FileInfo[]> => {
  return request.get(`/files/list${parentId ? `?parentId=${parentId}` : ''}`);
};

/**
 * 创建文件夹
 * @param parentId 父目录ID，空表示根目录
 * @param folderName 文件夹名称
 */
export const createFolder = (parentId: string, folderName: string): Promise<FileInfo> => {
  return request.post("/files/folder", {
    parentId,
    folderName
  });
};

/**
 * 上传文件
 * @param parentId 父目录ID，空表示根目录
 * @param file 文件对象
 */
export const uploadFile = (parentId: string, file: File): Promise<FileInfo> => {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("parentId", parentId);
  
  return request.post("/files/upload", formData, {
    headers: {
      "Content-Type": "multipart/form-data"
    }
  });
};

/**
 * 获取文件下载链接
 * @param fileId 文件ID
 */
export const getDownloadUrl = (fileId: string): string => {
  const token = localStorage.getItem("token") || "";
  // 检查token是否已经包含Bearer前缀，如果包含则直接使用，否则不添加前缀
  const tokenParam = token.startsWith("Bearer ") ? token.substring(7) : token;
  const url = `${SERVER_URL}/files/download/${fileId}?token=${tokenParam}`;
  return url;
};

/**
 * 重命名文件或文件夹
 * @param fileId 文件ID
 * @param newName 新名称
 */
export const renameFile = (fileId: string, newName: string): Promise<void> => {
  return request.put(`/files/rename/${fileId}`, { newName });
};

/**
 * 删除文件或文件夹
 * @param fileId 文件ID
 */
export const deleteFile = (fileId: string): Promise<void> => {
  return request.delete(`/files/${fileId}`);
}; 