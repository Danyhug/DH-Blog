/** 秒 → 3:07 / 1:02:09 */
export function formatTime(seconds?: number): string {
  if (!seconds || !Number.isFinite(seconds) || seconds < 0) return '0:00'
  const total = Math.floor(seconds)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const ss = String(s).padStart(2, '0')
  return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${ss}` : `${m}:${ss}`
}

/** 列表合计时长：「1 小时 12 分钟」/「38 分钟」 */
export function formatTotal(seconds: number): string {
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes} 分钟`
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return m ? `${h} 小时 ${m} 分钟` : `${h} 小时`
}

/** 影片时长的 Netflix 写法：「2小时 8分」/「45分钟」 */
export function formatRuntime(seconds?: number): string {
  if (!seconds) return ''
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}分钟`
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return m ? `${h}小时 ${m}分` : `${h}小时`
}

/** 后端时间串（2006-01-02 15:04:05）→ 毫秒，用于排序 */
export function timeValue(value?: string): number {
  if (!value) return 0
  const parsed = Date.parse(value.replace(' ', 'T'))
  return Number.isNaN(parsed) ? 0 : parsed
}

/** 由字符串稳定地生成一组渐变色，没有封面时用作占位 */
export function gradientFor(seed: string): string {
  let hash = 0
  for (let i = 0; i < seed.length; i++) hash = (hash * 31 + seed.charCodeAt(i)) | 0
  const hue = Math.abs(hash) % 360
  return `linear-gradient(135deg, hsl(${hue} 65% 45%), hsl(${(hue + 50) % 360} 70% 25%))`
}

export function shuffled<T>(items: T[]): T[] {
  const copy = items.slice()
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[copy[i], copy[j]] = [copy[j], copy[i]]
  }
  return copy
}
