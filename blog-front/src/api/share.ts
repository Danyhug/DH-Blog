import request from '@/api/axios'
import { SERVER_URL } from "@/types/Constant"

/**
 * 分享信息类型
 */
export interface ShareInfo {
  id: number
  share_id: string
  file_key: string
  password?: string
  expire_at?: string
  max_download_count?: number
  view_count: number
  download_count: number
  create_time: string
  update_time: string
}

/**
 * 公开分享信息类型（用于访问页）
 */
export interface PublicShareInfo {
  share_id: string
  file_name: string
  file_size: number
  has_password: boolean
  expire_at?: string
  is_expired: boolean
  view_count: number
  download_count: number
  create_time: string
}

/**
 * 创建分享请求类型
 */
export interface CreateShareRequest {
  file_key: string           // 文件ID（必填）
  password?: string          // 访问密码（可选）
  expire_days?: number       // 过期天数（可选）
  max_download_count?: number // 最大下载次数（可选）
}

/**
 * 分享访问日志类型
 */
export interface ShareAccessLog {
  id: number
  share_id: string
  action_type: 'view' | 'download'
  ip: string
  user_agent?: string
  referer?: string
  create_time: string
}

/**
 * 分页结果类型
 */
export interface PageResult<T> {
  total: number
  list: T[]
}

// ========== 管理接口（需要登录） ==========

/**
 * 创建分享
 * @param data 创建分享请求
 * @returns 分享信息
 */
export const createShare = (data: CreateShareRequest): Promise<ShareInfo> => {
  return request.post('/files/share', data)
}

/**
 * 获取分享列表
 * @param page 页码
 * @param pageSize 每页数量
 * @returns 分享列表
 */
export const listShares = (page: number = 1, pageSize: number = 10): Promise<PageResult<ShareInfo>> => {
  return request.get('/files/share', {
    params: { page, pageSize }
  })
}

/**
 * 获取分享详情（管理）
 * @param id 分享ID
 * @returns 分享详情
 */
export const getShareDetail = (id: number): Promise<ShareInfo> => {
  return request.get(`/files/share/${id}`)
}

/**
 * 删除分享
 * @param id 分享ID
 */
export const deleteShare = (id: number): Promise<void> => {
  return request.delete(`/files/share/${id}`)
}

/**
 * 获取分享访问日志
 * @param id 分享ID
 * @param page 页码
 * @param pageSize 每页数量
 * @returns 访问日志列表
 */
export const getShareAccessLogs = (id: number, page: number = 1, pageSize: number = 10): Promise<PageResult<ShareAccessLog>> => {
  return request.get(`/files/share/${id}/logs`, {
    params: { page, pageSize }
  })
}

// ========== 公开接口（无需登录） ==========

/**
 * 获取分享信息（公开）
 * @param shareId 分享短链ID
 * @returns 分享信息
 */
export const getShareInfo = (shareId: string): Promise<PublicShareInfo> => {
  return request.get(`/share/${shareId}`)
}

/**
 * 密码验证响应类型
 */
export interface VerifyPasswordResponse {
  valid: boolean
  download_token?: string
  expires_in?: number  // 令牌有效期（秒）
}

/**
 * 验证分享密码
 * @param shareId 分享短链ID
 * @param password 密码
 * @returns 验证结果和下载令牌
 */
export const verifySharePassword = (shareId: string, password: string): Promise<VerifyPasswordResponse> => {
  return request.post(`/share/${shareId}/verify`, { password })
}

/**
 * 获取分享下载链接
 * @param shareId 分享短链ID
 * @param token 下载令牌
 * @returns 下载链接
 */
export const getShareDownloadUrl = (shareId: string, token: string): string => {
  return `${SERVER_URL}/share/${shareId}/download?token=${encodeURIComponent(token)}`
}

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @returns 格式化后的字符串
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

/**
 * 生成分享链接
 * @param shareId 分享短链ID
 * @returns 完整分享链接
 */
export const generateShareLink = (shareId: string): string => {
  // 使用hash路由模式
  return `${window.location.origin}/#/share/${shareId}`
}
