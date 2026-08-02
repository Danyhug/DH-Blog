import request from '@/api/axios'
import { SERVER_URL } from '@/types/Constant'

/**
 * gatewayBaseUrl 拼出 agent 侧要填的完整地址。
 * 生产环境的 SERVER_URL 是相对路径（/api），直接显示出来无法复制使用。
 */
export function gatewayBaseUrl(): string {
  const configured = (window as any).__SERVER_CONFIG__?.SERVER_URL || SERVER_URL || '/api'
  const absolute = /^https?:\/\//.test(configured)
    ? configured
    : `${window.location.origin}${configured.startsWith('/') ? '' : '/'}${configured}`
  return `${absolute.replace(/\/+$/, '')}/gateway/v1`
}

/** 上游凭据。一个供应商可以配多把，网关轮换使用，被上游拒掉的会自动停止调度 */
export interface GatewayProviderKey {
  id: number
  label: string
  masked: string
  enabled: boolean
  /** active / auth_failed / quota_exceeded */
  status: string
  lastError: string
  lastUsedAt: string | null
  disabledAt: string | null
  /** 是否正在参与调度，已折算了启用状态与跨月自愈 */
  inRotation: boolean
  /**
   * 上游自己报的用量，每 60 分钟同步一次。
   * 与 monthlyUsed 是两套账：本地只数经过网关的请求，上游数的是这把密钥的全部消耗。
   */
  upstreamUsed: number
  /** 0 表示上游没有给出上限，不能理解为已用尽 */
  upstreamLimit: number
  /** credit / request，两家计量单位不同，不能互相换算 */
  upstreamUnit: string
  /** key = 这把密钥自己的额度；account = 与同账户其它密钥共享 */
  upstreamScope: string
  /** 额度对应的周期，Tavily 是账单周期、Brave 是滚动 30 天，都不是自然月 */
  upstreamWindow: string
  upstreamSyncedAt: string | null
  /** 最近一次同步失败的原因，成功后清空 */
  upstreamError: string
}

/** 搜索供应商配置（后台视图，密钥始终脱敏） */
export interface GatewayProvider {
  name: string
  displayName: string
  /** 供应商官网 / 文档 / 控制台，来自后端的静态元信息 */
  homeUrl: string
  docsUrl: string
  consoleUrl: string
  logoUrl: string
  /** 计费方式说明，三家口径不同，用于避免看板数字被误读 */
  billing: string
  enabled: boolean
  keys: GatewayProviderKey[]
  /** 当前仍在轮换中的密钥数 */
  activeKeys: number
  baseUrl: string
  priority: number
  weight: number
  rps: number
  monthlyQuota: number
  monthlyUsed: number
  /** 本月花费，单位为微美元（1e-6 USD），仅 Exa 这类按额计费的供应商非零 */
  monthlyCostMicroUsd: number
  /**
   * 月费用上限，同样是微美元，0 表示不限。
   * Exa 按金额计费（免费版每月 $10），单次价格不固定，次数上限说明不了预算。
   */
  monthlyCostLimitMicroUsd: number
  /** 该供应商是否提供用量接口；为 false 时数字空着是正常的，不是同步坏了 */
  supportsUsageSync: boolean
  extra: string
  health: 'closed' | 'open' | 'half_open'
}

/** 供应商配置补丁，字段留空表示不修改。凭据不在其中，走 keys 接口管理 */
export interface GatewayProviderPatch {
  displayName?: string
  enabled?: boolean
  baseUrl?: string
  priority?: number
  weight?: number
  rps?: number
  monthlyQuota?: number
  monthlyCostLimitMicroUsd?: number
  extra?: string
}

/** 连通性测试的入参，全部可选：带 apiKey 就是"保存前先测" */
export interface GatewayProviderProbe {
  keyId?: number
  apiKey?: string
  baseUrl?: string
  extra?: string
}

export interface GatewayProviderTestResult {
  ok: boolean
  latencyMs: number
  resultCount?: number
  error?: string
  keyLabel?: string
}

/** 网关签发给 agent 的 API Key（不含明文） */
export interface GatewayApiKey {
  id: number
  name: string
  keyPrefix: string
  enabled: boolean
  allowedProviders: string
  rateLimitPerMin: number
  monthlyQuota: number
  monthlyUsed: number
  expireAt: string | null
  lastUsedAt: string | null
  note: string
}

export interface CreateGatewayApiKeyPayload {
  name: string
  allowedProviders?: string
  rateLimitPerMin?: number
  monthlyQuota?: number
  expireDays?: number
  note?: string
}

/** 创建接口是唯一能拿到明文 Key 的地方 */
export interface CreatedGatewayApiKey {
  id: number
  name: string
  apiKey: string
}

export interface GatewayRequestLog {
  id: number
  createdAt: string
  apiKeyId: number
  provider: string
  endpoint: string
  query: string
  resultCount: number
  cached: boolean
  fallbackFrom: string
  status: string
  httpStatus: number
  latencyMs: number
  credits: number
  costMicroUsd: number
  error: string
  clientIp: string
}

export interface GatewayProviderStats {
  provider: string
  total: number
  succeeded: number
  cached: number
  credits: number
  costMicroUsd: number
  avgLatency: number
}

export interface GatewayQuota {
  provider: string
  monthlyQuota: number
  monthlyUsed: number
  monthlyCostMicroUsd: number
  monthlyCostLimitMicroUsd: number
}

export interface GatewayStats {
  days: number
  total: number
  succeeded: number
  cached: number
  credits: number
  costMicroUsd: number
  providers: GatewayProviderStats[]
  quotas: GatewayQuota[]
}

/** 调度方式；model 尚未接入，选中后后端仍按 balanced 执行 */
export type RoutingStrategy = 'balanced' | 'priority' | 'model'

export interface StrategyOption {
  value: RoutingStrategy
  label: string
  description: string
  /** 为 false 表示该调度方式还没接入，界面需要如实说明 */
  implemented: boolean
}

export interface GatewaySettings {
  routingStrategy: RoutingStrategy
  strategies: StrategyOption[]
}

export function getGatewaySettings(): Promise<GatewaySettings> {
  return request({ url: '/admin/gateway/settings', method: 'get' })
}

export function updateGatewaySettings(data: { routingStrategy: RoutingStrategy }) {
  return request({ url: '/admin/gateway/settings', method: 'put', data })
}

export function getGatewayProviders(): Promise<GatewayProvider[]> {
  return request({ url: '/admin/gateway/providers', method: 'get' })
}

export function updateGatewayProvider(name: string, data: GatewayProviderPatch) {
  return request({ url: `/admin/gateway/providers/${name}`, method: 'put', data })
}

export function testGatewayProvider(name: string, probe: GatewayProviderProbe = {}): Promise<GatewayProviderTestResult> {
  return request({ url: `/admin/gateway/providers/${name}/test`, method: 'post', data: probe })
}

/** 一次用量同步的结果。skipped 指没有用量接口或还没数据可报，不算失败 */
export interface GatewayUsageSyncResult {
  synced: number
  skipped: number
  failed: number
  parked: string[]
  revived: string[]
}

/** 手动触发一次上游用量同步；后台每 60 分钟也会自己跑一次 */
export function syncGatewayUsage(): Promise<GatewayUsageSyncResult> {
  return request({ url: '/admin/gateway/usage/sync', method: 'post' })
}

export function createGatewayProviderKey(name: string, data: { label?: string; apiKey: string }) {
  return request({ url: `/admin/gateway/providers/${name}/keys`, method: 'post', data })
}

export function updateGatewayProviderKey(
  name: string,
  id: number,
  data: { label?: string; apiKey?: string; enabled?: boolean; revive?: boolean }
) {
  return request({ url: `/admin/gateway/providers/${name}/keys/${id}`, method: 'put', data })
}

export function deleteGatewayProviderKey(name: string, id: number) {
  return request({ url: `/admin/gateway/providers/${name}/keys/${id}`, method: 'delete' })
}

export function getGatewayApiKeys(): Promise<GatewayApiKey[]> {
  return request({ url: '/admin/gateway/keys', method: 'get' })
}

export function createGatewayApiKey(data: CreateGatewayApiKeyPayload): Promise<CreatedGatewayApiKey> {
  return request({ url: '/admin/gateway/keys', method: 'post', data })
}

/** 再次取出明文，用于重复复制；后端只在这个接口返回它 */
export function revealGatewayApiKey(id: number): Promise<{ id: number; name: string; apiKey: string }> {
  return request({ url: `/admin/gateway/keys/${id}/reveal`, method: 'get' })
}

export function updateGatewayApiKey(id: number, data: Partial<GatewayApiKey>) {
  return request({ url: `/admin/gateway/keys/${id}`, method: 'put', data })
}

export function deleteGatewayApiKey(id: number) {
  return request({ url: `/admin/gateway/keys/${id}`, method: 'delete' })
}

export function getGatewayLogs(params: {
  page?: number
  pageSize?: number
  provider?: string
  status?: string
}): Promise<{ total: number; list: GatewayRequestLog[] }> {
  return request({ url: '/admin/gateway/logs', method: 'get', params })
}

export function getGatewayStats(days = 1): Promise<GatewayStats> {
  return request({ url: '/admin/gateway/stats', method: 'get', params: { days } })
}
