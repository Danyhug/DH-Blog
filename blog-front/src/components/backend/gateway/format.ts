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
