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
 * 获取系统目录树
 * @param rootPath 根目录路径，为空则使用系统根目录
 * @param maxDepth 最大深度，默认为2
 * @returns 目录树结构
 */
export const getDirectoryTree = (rootPath?: string, maxDepth?: number): Promise<any> => {
  let url = '/files/directory-tree'
  const params: Record<string, string> = {}
  
  if (rootPath) {
    params.rootPath = rootPath
  }
  
  if (maxDepth !== undefined) {
    params.maxDepth = maxDepth.toString()
  }
  
  return request.get(url, { params })
}

/**
 * 列出文件
 * @param parentId 父目录ID，为空则获取根目录
 * @returns 文件列表
 */
export const listFiles = (parentId?: string): Promise<any> => {
  return request.get('/files/list', {
    params: { parentId }
  })
}

/**
 * 上传文件
 * @param file 文件对象
 * @param parentId 父目录ID，为空则上传到根目录
 * @returns 上传结果
 */
export const uploadFile = (file: File, parentId?: string): Promise<any> => {
  const formData = new FormData()
  formData.append('file', file)
  
  if (parentId) {
    formData.append('parentId', parentId)
  }
  
  return request.post('/files/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

/**
 * 创建文件夹
 * @param folderName 文件夹名称
 * @param parentId 父目录ID，为空则创建在根目录
 * @returns 创建结果
 */
export const createFolder = (folderName: string, parentId?: string): Promise<any> => {
  return request.post('/files/folder', {
    folderName,
    parentId: parentId || ''
  })
}

/**
 * 重命名文件或文件夹
 * @param fileId 文件ID
 * @param newName 新名称
 * @returns 重命名结果
 */
export const renameFile = (fileId: string, newName: string): Promise<any> => {
  return request.put(`/files/rename/${fileId}`, {
    newName
  })
}

/**
 * 删除文件或文件夹
 * @param fileId 文件ID
 * @returns 删除结果
 */
export const deleteFile = (fileId: string): Promise<any> => {
  return request.delete(`/files/${fileId}`)
}

/**
 * 更新存储路径
 * @param path 新的存储路径
 * @returns 更新结果
 */
export const updateStoragePath = (path: string): Promise<any> => {
  return request.put('/files/storage-path', { path })
}

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