import type { Video } from '@/api/media'
import { timeValue } from '../utils/format'

// 把扁平的视频列表整理成 Netflix 式的「作品」：一部电影，或一部由同一文件夹里
// 带集号的视频组成的剧集。后端只负责从文件名里解析季/集，归组在这里做。

export interface Title {
  key: string
  kind: 'movie' | 'series'
  name: string
  year?: number
  /** 电影只有一个元素；剧集按季、集排好序 */
  videos: Video[]
  imageId?: string
  posterId?: string
  addedAt: number
  watchedAt: number
  /** 所在文件夹（电影按它分行），根目录为空 */
  collectionId: string
  collectionName: string
}

function compareEpisodes(a: Video, b: Video): number {
  return (a.season || 1) - (b.season || 1) || (a.episode || 0) - (b.episode || 0) || a.name.localeCompare(b.name, 'zh-CN', { numeric: true })
}

function watchedAt(video: Video) {
  return timeValue(video.progress?.updated_at)
}

export function buildTitles(videos: Video[]): Title[] {
  const byCollection = new Map<string, Video[]>()
  for (const video of videos) {
    const group = byCollection.get(video.collection_id)
    if (group) group.push(video)
    else byCollection.set(video.collection_id, [video])
  }

  const titles: Title[] = []
  for (const [collectionId, items] of byCollection) {
    const episodic = items.filter(video => video.episode).length
    // 根目录里散落的视频不是一部剧；文件夹里至少两集才算剧集
    if (collectionId && episodic >= 2) {
      const sorted = items.slice().sort(compareEpisodes)
      titles.push({
        key: `s:${collectionId}`,
        kind: 'series',
        name: items[0].collection_name,
        year: sorted.find(video => video.year)?.year,
        videos: sorted,
        imageId: sorted.find(video => video.backdrop_file_id)?.backdrop_file_id || sorted.find(video => video.poster_file_id)?.poster_file_id,
        posterId: sorted.find(video => video.poster_file_id)?.poster_file_id,
        addedAt: Math.max(...items.map(video => timeValue(video.added_at))),
        watchedAt: Math.max(...items.map(watchedAt)),
        collectionId,
        collectionName: items[0].collection_name
      })
      continue
    }
    for (const video of items) {
      titles.push({
        key: `v:${video.id}`,
        kind: 'movie',
        name: video.title,
        year: video.year,
        videos: [video],
        imageId: video.backdrop_file_id || video.poster_file_id,
        posterId: video.poster_file_id,
        addedAt: timeValue(video.added_at),
        watchedAt: watchedAt(video),
        collectionId,
        collectionName: video.collection_name
      })
    }
  }
  return titles.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN', { numeric: true }))
}

function isInProgress(video: Video): boolean {
  return !!video.progress && !video.progress.finished && video.progress.position > 5
}

/**
 * 继续观看：每部作品只出一个入口。看到一半的那集优先；
 * 最近看完的是剧集中的某一集时，给出它的下一集（Netflix 的「下一集」）。
 */
export function continueWatching(titles: Title[]): { title: Title; video: Video }[] {
  const entries: { title: Title; video: Video; at: number }[] = []
  for (const title of titles) {
    const watched = title.videos.filter(video => video.progress).sort((a, b) => watchedAt(b) - watchedAt(a))
    const latest = watched[0]
    if (!latest) continue
    if (isInProgress(latest)) {
      entries.push({ title, video: latest, at: watchedAt(latest) })
    } else if (latest.progress?.finished && title.kind === 'series') {
      const next = nextEpisode(title, latest)
      if (next && !next.progress?.finished) entries.push({ title, video: next, at: watchedAt(latest) })
    }
  }
  return entries.sort((a, b) => b.at - a.at).map(({ title, video }) => ({ title, video }))
}

export function nextEpisode(title: Title, video: Video): Video | undefined {
  const index = title.videos.findIndex(item => item.id === video.id)
  return index >= 0 ? title.videos[index + 1] : undefined
}

/** 点作品的「播放」时该放哪一集 */
export function resumeTarget(title: Title): Video {
  return continueWatching([title])[0]?.video ?? title.videos[0]
}

export function episodeLabel(video: Video): string {
  if (!video.episode) return ''
  return video.season ? `S${video.season}:E${video.episode}` : `第 ${video.episode} 集`
}

export function seasonsOf(title: Title): number[] {
  return [...new Set(title.videos.map(video => video.season || 1))].sort((a, b) => a - b)
}

export function matchesTitle(title: Title, keyword: string): boolean {
  const k = keyword.toLowerCase()
  return title.name.toLowerCase().includes(k) || title.videos.some(video => video.title.toLowerCase().includes(k) || video.name.toLowerCase().includes(k))
}
