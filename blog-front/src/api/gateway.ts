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
   * 这个账号自己的月额度，0 表示没单独配。
   * 额度是跟着账号走的：两个各 1000 的账号合起来是 2000，供应商级的一个数字说不出这件事。
   */
  monthlyQuota: number
  /** 同上的费用上限，单位微美元，0 表示不限 */
  monthlyCostLimitMicroUsd: number
  /** 这把密钥本月经过网关的用量，密钥级额度就是拿它来衡量的 */
  monthlyUsed: number
  monthlyCostMicroUsd: number
  /**
   * 上游用量接口里这把密钥的标识（Exa 是搜索密钥的 UUID）。
   * 必须逐把填：整条轮换链共用一个 id，会让每把密钥都上报同一把的花费。
   */
  usageKeyId: string
  /**
   * 上游自己报的用量，每 60 分钟同步一次。
   * 与 monthlyUsed 是两套账：本地只数经过网关的请求，上游数的是这把密钥的全部消耗。
   */
  upstreamUsed: number
  /** 0 表示上游没有给出上限，不能理解为已用尽 */
  upstreamLimit: number
  /** credit / request，各家计量单位不同，不能互相换算 */
  upstreamUnit: string
  /** key = 这把密钥自己的额度；account = 与同账户其它密钥共享 */
  upstreamScope: string
  /** 额度对应的周期，Tavily / Firecrawl 是账单周期、Brave 是滚动 30 天，都不是自然月 */
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
  /** 计费方式说明，各家口径不同，用于避免看板数字被误读 */
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
  /**
   * 选路真正衡量的额度：任何一把密钥配了自己的额度时，它就是这些密钥的汇总，
   * 否则等于上面的供应商级数字。两者不同是正常的 —— 前者才是"总额度"。
   */
  effectiveMonthlyQuota: number
  effectiveMonthlyUsed: number
  effectiveMonthlyCostMicroUsd: number
  effectiveMonthlyCostLimitMicroUsd: number
  /** 该供应商是否提供用量接口；为 false 时数字空着是正常的，不是同步坏了 */
  supportsUsageSync: boolean
  /**
   * 上游自己报的花费（微美元）。非 null 时选路用的是这个数而不是本地累加值；
   * null 表示没有新鲜的上游数据，当前按本地统计走。
   */
  upstreamCostMicroUsd: number | null
  /** Exa 团队管理接口所需的 service key，只会返回掩码，真实值不出后端。
   *  对应的密钥 UUID 是逐把配的，见 GatewayProviderKey.usageKeyId */
  usageServiceKeyMasked: string
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
  /** Exa 团队用量接口的 service key：传空串表示清除，不传表示不改 */
  usageServiceKey?: string
}

/** 上游凭据的补丁，字段留空表示不修改 */
export interface GatewayProviderKeyPatch {
  label?: string
  apiKey?: string
  enabled?: boolean
  revive?: boolean
  /** 这个账号自己的月额度，0 表示不单独限制 */
  monthlyQuota?: number
  monthlyCostLimitMicroUsd?: number
  /** 上游用量接口里这把密钥的标识，传空串表示清除 */
  usageKeyId?: string
}

/** 本地用量校准补丁，字段留空表示不修改；值是覆盖而不是累加 */
export interface GatewayUsagePatch {
  count?: number
  credits?: number
  costMicroUsd?: number
  /** 只校准某一把上游密钥的计数；不传则改供应商级的总数 */
  keyId?: number
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
  /** 能力范围，逗号分隔；空串表示仅搜索 */
  scopes?: string
  /** 用这把 Key 写文章时显示的署名 */
  authorName?: string
}

export interface CreateGatewayApiKeyPayload {
  name: string
  allowedProviders?: string
  rateLimitPerMin?: number
  monthlyQuota?: number
  expireDays?: number
  note?: string
  scopes?: string
  authorName?: string
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

interface GatewayProviderStats {
  provider: string
  total: number
  succeeded: number
  cached: number
  credits: number
  costMicroUsd: number
  avgLatency: number
}

interface GatewayQuota {
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

/** 按供应商官网的真实账单覆盖本月的本地统计 */
export function updateGatewayProviderUsage(name: string, data: GatewayUsagePatch) {
  return request.put(`/admin/gateway/providers/${name}/usage`, data)
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

export function createGatewayProviderKey(name: string, data: GatewayProviderKeyPatch & { apiKey: string }) {
  return request({ url: `/admin/gateway/providers/${name}/keys`, method: 'post', data })
}

export function updateGatewayProviderKey(name: string, id: number, data: GatewayProviderKeyPatch) {
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

/** MCP 工具的一个入参，由后端从工具的 JSON Schema 摊平而来 */
interface GatewayMcpParam {
  name: string
  type: string
  description: string
  required: boolean
}

export interface GatewayMcpTool {
  name: string
  title: string
  /** 原样是模型看到的工具说明，不是给管理员写的摘要 */
  description: string
  /** 门槛 scope；等于基线 scope 时表示每把 Key 都能用 */
  scope: string
  params: GatewayMcpParam[]
}

export interface GatewayMcpScope {
  value: string
  label: string
  description: string
  /** 基线能力，每把 Key 自带，不作为可勾选项 */
  baseline: boolean
}

export interface GatewayMcpCatalog {
  serverName: string
  version: string
  instructions: string
  endpoint: string
  scopes: GatewayMcpScope[]
  tools: GatewayMcpTool[]
}

/**
 * 工具目录来自后端实际挂载的工具表。前端不再自己维护工具名与 scope 清单，
 * 后端新注册一个 MCP 工具，这里就自动多一项。
 */
export function getGatewayMcpCatalog(): Promise<GatewayMcpCatalog> {
  return request({ url: '/admin/gateway/mcp/tools', method: 'get' })
}
