import request from '@/api/axios'

/** 一次 AI 修改授权的摘要视图（列表接口，不含明文 token） */
export interface AgentGrant {
  id: number
  tokenPrefix: string
  expireAt: string
  articleId: number
  revoked: boolean
  usedCount: number
  lastUsedAt: string | null
  note: string
  createTime: string
}

/** 创建接口的返回值，是唯一能拿到明文 token 的地方 */
export interface CreatedAgentGrant {
  id: number
  token: string
  expireAt: string
  articleId: number
  note: string
}

export function getAgentGrants(): Promise<AgentGrant[]> {
  return request({ url: '/admin/agent/grants', method: 'get' })
}

export function createAgentGrant(data: { note?: string; articleId?: number }): Promise<CreatedAgentGrant> {
  return request({ url: '/admin/agent/grants', method: 'post', data })
}

/** 过期后再看一次明文 token */
export function revealAgentGrant(id: number): Promise<{ id: number; token: string }> {
  return request({ url: `/admin/agent/grants/${id}/reveal`, method: 'get' })
}

export function deleteAgentGrant(id: number): Promise<void> {
  return request({ url: `/admin/agent/grants/${id}`, method: 'delete' })
}