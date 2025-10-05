import request from '@/api/axios'

export interface SystemConfig {
  webdav_chunk_size?: number
  [key: string]: any
}

// 获取系统配置
export async function getSystemConfig(): Promise<SystemConfig> {
  const response = await request({
    url: '/admin/config',
    method: 'get'
  })
  return response
}

// 更新系统配置
export async function updateSystemConfig(data: Partial<SystemConfig>) {
  return request({
    url: '/admin/config',
    method: 'put',
    data
  })
}