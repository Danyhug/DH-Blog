import { mediaStreamUrl } from '@/api/media'

/**
 * 没有海报图的视频，从视频本身截一帧当缩略图。
 * 只用 preload=metadata 加 seek，浏览器按 Range 只取那一小段；全局限流 2 路，
 * 结果缓存在内存里（一次会话内反复滚动不重复截）。
 * 浏览器解不了的格式（部分 MKV/AVI）会失败，调用方回落到渐变占位。
 */
const MAX_ACTIVE = 2
const TIMEOUT = 20_000

const cache = new Map<string, string>()
const inflight = new Map<string, Promise<string>>()
const waiting: (() => void)[] = []
let active = 0

export function captureFrame(fileId: string): Promise<string> {
  const cached = cache.get(fileId)
  if (cached !== undefined) return Promise.resolve(cached)
  const running = inflight.get(fileId)
  if (running) return running
  const task = acquire().then(() => grab(fileId)).then(result => {
    cache.set(fileId, result)
    return result
  }).finally(() => {
    inflight.delete(fileId)
    active--
    waiting.shift()?.()
  })
  inflight.set(fileId, task)
  return task
}

function acquire(): Promise<void> {
  if (active < MAX_ACTIVE) {
    active++
    return Promise.resolve()
  }
  return new Promise(resolve => waiting.push(() => {
    active++
    resolve()
  }))
}

function grab(fileId: string): Promise<string> {
  return new Promise(resolve => {
    const video = document.createElement('video')
    video.crossOrigin = 'anonymous'
    video.muted = true
    video.playsInline = true
    video.preload = 'metadata'
    let settled = false
    const finish = (value: string) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      video.removeAttribute('src')
      video.load()
      resolve(value)
    }
    const timer = setTimeout(() => finish(''), TIMEOUT)
    video.addEventListener('loadedmetadata', () => {
      const duration = Number.isFinite(video.duration) ? video.duration : 0
      // 片头常是黑屏或片商 Logo，取 10% 处（3 秒到 5 分钟之间）更像海报
      video.currentTime = duration ? Math.min(Math.max(duration * 0.1, 3), 300, duration / 2) : 3
    }, { once: true })
    video.addEventListener('seeked', () => {
      try {
        const width = 480
        const height = Math.round((video.videoHeight / video.videoWidth) * width) || 270
        const canvas = document.createElement('canvas')
        canvas.width = width
        canvas.height = height
        canvas.getContext('2d')!.drawImage(video, 0, 0, width, height)
        finish(canvas.toDataURL('image/jpeg', 0.72))
      } catch {
        finish('')
      }
    }, { once: true })
    video.addEventListener('error', () => finish(''), { once: true })
    video.src = mediaStreamUrl(fileId)
  })
}
