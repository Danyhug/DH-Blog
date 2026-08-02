/** AI 网关各标签页共用的展示口径，集中在这里避免各页各写一份、数字含义对不上 */

/** formatCost 把微美元转成可读金额；小额也保留有效数字，否则一律显示为 $0.00 */
export function formatCost(microUsd: number) {
  if (!microUsd) return '$0'
  const usd = microUsd / 1_000_000
  if (usd < 0.01) return `$${usd.toFixed(4)}`
  return `$${usd.toFixed(2)}`
}

export function formatTime(value: string | null) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString('zh-CN')
}

/** fallbackColor 由名称派生出稳定的色相，保证同一供应商每次渲染颜色一致 */
export function fallbackColor(name: string) {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) % 360
  return `hsl(${hash}, 55%, 48%)`
}

/** 配额为 0 表示不限量，此时进度条恒为 0 而不是满格 */
export function quotaPercentage(used: number, quota: number) {
  if (!quota) return 0
  return Math.min(100, Math.round((used / quota) * 100))
}

/** formatSince 把同步时间说成「多久以前」，比一串时间戳更能看出数字新不新 */
export function formatSince(value: string | null) {
  if (!value) return '从未同步'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  const minutes = Math.floor((Date.now() - parsed.getTime()) / 60000)
  if (minutes < 1) return '刚刚同步'
  if (minutes < 60) return `${minutes} 分钟前同步`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前同步`
  return `${Math.floor(hours / 24)} 天前同步`
}

/** 上游的计量单位不能混着显示：Tavily 扣 credit，Brave 扣请求数 */
export function usageUnitLabel(unit: string) {
  if (unit === 'credit') return 'credit'
  if (unit === 'request') return '次请求'
  return unit
}

/** account 表示这条额度是同账户多把密钥共用的，换一把密钥并不会多出余量 */
export function usageScopeLabel(scope: string) {
  return scope === 'account' ? '账户共享额度' : '该密钥独立额度'
}
