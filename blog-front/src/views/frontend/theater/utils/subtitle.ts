export interface Cue {
  start: number
  end: number
  text: string
}

const TIMESTAMP = /(?:(\d{1,2}):)?(\d{1,2}):(\d{1,2})[.,](\d{1,3})/

function toSeconds(stamp: string): number {
  const match = stamp.match(TIMESTAMP)
  if (!match) return NaN
  const [, h, m, s, ms] = match
  return Number(h || 0) * 3600 + Number(m) * 60 + Number(s) + Number(ms.padEnd(3, '0')) / 1000
}

/**
 * 解析 SRT / WebVTT 字幕。两者都是「时间行 + 若干文本行」的块结构，
 * 区别只在毫秒分隔符与文件头，统一按块处理即可。
 * 字幕自己渲染（而不是交给 <track>），是为了能像 Netflix 那样控制字号与位置。
 */
export function parseSubtitles(text: string): Cue[] {
  const cues: Cue[] = []
  const blocks = text.replace(/\r/g, '').replace(/^﻿/, '').split(/\n{2,}/)
  for (const block of blocks) {
    const lines = block.split('\n')
    const timingIndex = lines.findIndex(line => line.includes('-->'))
    if (timingIndex < 0) continue
    const [rawStart, rawEnd] = lines[timingIndex].split('-->')
    const start = toSeconds(rawStart)
    const end = toSeconds(rawEnd)
    if (Number.isNaN(start) || Number.isNaN(end)) continue
    const body = lines
      .slice(timingIndex + 1)
      .join('\n')
      // 去掉 <i>/<b>/<font> 与 ASS 风格的 {\an8} 标签，只保留文字
      .replace(/<[^>]+>/g, '')
      .replace(/\{\\[^}]*\}/g, '')
      .trim()
    if (body) cues.push({ start, end, text: body })
  }
  return cues.sort((a, b) => a.start - b.start)
}

export function activeCues(cues: Cue[], now: number): Cue[] {
  return cues.filter(cue => cue.start <= now && cue.end >= now)
}
