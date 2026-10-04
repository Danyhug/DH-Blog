export interface LyricLine {
  /** 秒；无时间轴的纯文本歌词为 -1 */
  time: number
  text: string
}

const TIME_TAG = /\[(\d{1,3}):(\d{1,2})(?:[.:](\d{1,3}))?\]/g
const META_TAG = /^\[(ar|ti|al|by|offset|re|ve|length|au|la|id):(.*)\]$/i

/**
 * 解析 LRC。支持一行多个时间标签（副歌复用）与 [offset:±ms]；
 * 一个时间标签都没有时按纯文本歌词返回，time 统一为 -1。
 */
export function parseLyrics(text: string): { lines: LyricLine[]; synced: boolean } {
  const lines: LyricLine[] = []
  const plain: LyricLine[] = []
  let offset = 0

  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line) continue
    const meta = line.match(META_TAG)
    if (meta) {
      if (meta[1].toLowerCase() === 'offset') offset = Number(meta[2]) / 1000 || 0
      continue
    }
    const stamps = [...line.matchAll(TIME_TAG)]
    if (stamps.length === 0) {
      plain.push({ time: -1, text: line })
      continue
    }
    const content = line.replace(TIME_TAG, '').trim()
    for (const stamp of stamps) {
      const fraction = stamp[3] ? Number(stamp[3].padEnd(3, '0')) / 1000 : 0
      lines.push({ time: Number(stamp[1]) * 60 + Number(stamp[2]) + fraction, text: content })
    }
  }

  if (lines.length === 0) return { lines: plain, synced: false }
  // offset 为正表示歌词整体提前
  for (const line of lines) line.time = Math.max(0, line.time - offset)
  lines.sort((a, b) => a.time - b.time)
  return { lines, synced: true }
}

/** 当前应高亮的行：最后一个 time <= now 的行 */
export function activeLyricIndex(lines: LyricLine[], now: number): number {
  let lo = 0
  let hi = lines.length - 1
  let found = -1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    if (lines[mid].time <= now + 0.15) {
      found = mid
      lo = mid + 1
    } else {
      hi = mid - 1
    }
  }
  return found
}
