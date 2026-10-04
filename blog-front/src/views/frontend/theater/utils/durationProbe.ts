import { mediaStreamUrl, reportDurations } from '@/api/media'

/**
 * 后端读标签拿不到可靠的时长，浏览器的媒体元素却总能拿到。
 * 这里在后台用 preload=metadata 逐个探测列表里缺时长的曲目（只拉文件头几个 KB），
 * 结果写回曲目对象并批量上报给后端缓存，下次打开就不用再探测。
 */
interface Probe {
  id: string
  duration?: number
}

const MAX_ACTIVE = 2
const PROBE_TIMEOUT = 15_000
const FLUSH_DELAY = 3_000

const queue: Probe[] = []
const pending = new Set<string>()
const reports = new Map<string, number>()
let active = 0
let flushTimer: ReturnType<typeof setTimeout> | null = null

export function probeDurations(items: Probe[]) {
  for (const item of items) {
    if (item.duration || pending.has(item.id)) continue
    pending.add(item.id)
    queue.push(item)
  }
  pump()
}

/** 播放器已经拿到时长时直接登记，免得再探测一次 */
export function rememberDuration(id: string, duration: number) {
  if (!Number.isFinite(duration) || duration <= 0) return
  reports.set(id, duration)
  scheduleFlush()
}

function pump() {
  while (active < MAX_ACTIVE && queue.length) {
    const item = queue.shift()!
    active++
    measure(item.id).then(duration => {
      if (duration) {
        item.duration = duration
        rememberDuration(item.id, duration)
      }
    }).finally(() => {
      active--
      pump()
    })
  }
}

function measure(id: string): Promise<number> {
  return new Promise(resolve => {
    const audio = new Audio()
    audio.preload = 'metadata'
    audio.muted = true
    let settled = false
    const done = (value: number) => {
      // 释放 src 本身会再触发一次 error，必须只结算一次
      if (settled) return
      settled = true
      clearTimeout(timer)
      audio.removeAttribute('src')
      audio.load()
      resolve(value)
    }
    const timer = setTimeout(() => done(0), PROBE_TIMEOUT)
    audio.addEventListener('loadedmetadata', () => done(Number.isFinite(audio.duration) ? audio.duration : 0), { once: true })
    audio.addEventListener('error', () => done(0), { once: true })
    audio.src = mediaStreamUrl(id)
  })
}

function scheduleFlush() {
  if (flushTimer) return
  flushTimer = setTimeout(() => {
    flushTimer = null
    const items = [...reports].slice(0, 100).map(([id, duration]) => ({ id, duration }))
    items.forEach(item => reports.delete(item.id))
    if (items.length) reportDurations(items).catch(() => {})
    if (reports.size) scheduleFlush()
  }, FLUSH_DELAY)
}
