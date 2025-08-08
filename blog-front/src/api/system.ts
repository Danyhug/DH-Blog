import request from '@/api/axios'

export interface SystemConfig {
  webdav_chunk_size?: number
  [key: string]: any
}

// 获取系统配置
export async function getSystemConfig(): Promise<SystemConfig> {
  const response = await request({
    url: '/api/system/config',
    method: 'get'
  })
  return response.data
}

// 更新系统配置
export async function updateSystemConfig(data: Partial<SystemConfig>) {
  return request({
    url: '/api/system/config',
    method: 'put',
    data
  })
}