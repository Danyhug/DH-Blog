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
 * 全盘搜索命中：文件本身 + 定位所需的所在目录信息
 */
interface SearchHit extends FileInfo {
  /** 所在目录的展示路径，以 / 分隔；位于根目录时为空 */
  parent_path: string;
  /** 从根到所在目录的完整链路，用于还原面包屑 */
  parent_segments: { id: string; name: string }[];
}

export interface SearchResult {
  files: SearchHit[];
  /** 命中数超过 limit，返回的只是前 limit 条 */
  truncated: boolean;
  limit: number;
}

/**
 * 跨目录搜索文件（全盘搜索，含子目录）
 * @param keyword 搜索关键词
 * @returns 命中列表及是否被截断
 */
export const searchFiles = (keyword: string): Promise<SearchResult> => {
  return request.get('/files/search', {
    params: { keyword }
  })
}

/**
 * 初始化分片上传
 * @param parentId 父目录ID
 * @param fileName 文件名
 * @param fileSize 文件大小
 * @param chunkSize 分片大小（字节）。传 0 表示由服务端按系统配置决定，响应里会回传实际使用的值
 * @param uploadId 指定上传会话ID（用于断点续传）
 * @returns 上传会话ID
 */
export const initChunkUpload = (parentId: string | undefined, fileName: string, fileSize: number, chunkSize: number = 0, uploadId?: string): Promise<any> => {
  return request.post('/files/upload/chunk/init', {
    parentId: parentId || '',
    fileName,
    fileSize,
    chunkSize,
    uploadId
  })
}

/**
 * 上传分片
 * @param uploadId 上传会话ID
 * @param chunkIndex 分片索引
 * @param chunkData 分片数据
 * @returns 上传结果
 */
export const uploadChunk = (uploadId: string, chunkIndex: number, chunkData: Blob): Promise<any> => {
  const formData = new FormData()
  formData.append('uploadId', uploadId)
  formData.append('chunkIndex', chunkIndex.toString())
  formData.append('chunk', chunkData)
  
  return request.post('/files/upload/chunk/chunk', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

/**
 * 完成分片上传
 * @param uploadId 上传会话ID
 * @returns 上传结果
 */
export const completeChunkUpload = (uploadId: string): Promise<any> => {
  return request.post('/files/upload/chunk/complete', {
    uploadId
  })
}

/**
 * 获取已上传分片列表
 * @param uploadId 上传会话ID
 * @returns 已上传分片列表、总分片数和会话创建时使用的分片大小
 */
export const getUploadedChunks = (uploadId: string): Promise<any> => {
  // uploadId 内含原始文件名，其中的 #、?、% 会截断或改写 URL 路径，
  // 必须编码，后端 gin 会先解码再取 Param。
  return request.get(`/files/upload/chunk/${encodeURIComponent(uploadId)}/chunks`)
}

/**
 * 取消分片上传
 * @param uploadId 上传会话ID
 * @returns 取消结果
 */
export const cancelChunkUpload = (uploadId: string): Promise<any> => {
  return request.delete(`/files/upload/chunk/${encodeURIComponent(uploadId)}`)
}

/**
 * 创建文件夹
 * @param parentId 父目录ID，为空则创建在根目录
 * @param folderName 文件夹名称
 * @returns 创建结果
 */
export const createFolder = (parentId: string, folderName: string): Promise<any> => {
  return request.post('/files/folder', {
    parentId: parentId || '',
    folderName
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
 * 获取文件下载链接
 * @param fileId 文件ID
 * @param preview 是否为预览模式（用于音视频流式传输）
 */
export const getDownloadUrl = (fileId: string, preview: boolean = false): string => {
  const token = localStorage.getItem("token") || "";
  // 检查token是否已经包含Bearer前缀，如果包含则直接使用，否则不添加前缀
  const tokenParam = token.startsWith("Bearer ") ? token.substring(7) : token;
  let url = `${SERVER_URL}/files/download/${fileId}?token=${tokenParam}`;
  if (preview) {
    url += '&preview=true';
  }
  return url;
};

/**
 * 获取批量打包（zip）下载链接
 * @param fileIds 文件ID列表
 * @param archiveName 压缩包文件名，不带后缀
 */
export const getBatchDownloadUrl = (fileIds: (string | number)[], archiveName = "dhblog-download"): string => {
  const token = localStorage.getItem("token") || "";
  const tokenParam = token.startsWith("Bearer ") ? token.substring(7) : token;
  const ids = fileIds.join(",");
  return `${SERVER_URL}/files/download-batch?ids=${encodeURIComponent(ids)}&name=${encodeURIComponent(archiveName)}&token=${tokenParam}`;
};