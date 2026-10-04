import request from '@/api/axios'
import { SERVER_URL } from '@/types/Constant'
import { getDownloadUrl } from '@/api/file'

// 首次打开音乐库时后端要逐个读取音频标签，大曲库可能远超默认的 10 秒超时
const LIBRARY_TIMEOUT = 180_000

export interface Track {
  id: string
  name: string
  title: string
  artist: string
  album: string
  album_artist?: string
  genre?: string
  year?: number
  track_no?: number
  disc_no?: number
  duration?: number
  size: number
  mime_type?: string
  folder_id: string
  folder_name: string
  folder_path: string
  cover_file_id?: string
  has_embedded_cover: boolean
  has_lyrics: boolean
  play_count?: number
  last_played_at?: string
  added_at?: string
}

interface Subtitle {
  id: string
  label: string
  lang: string
  format: 'srt' | 'vtt'
}

export interface VideoProgress {
  position: number
  duration: number
  finished: boolean
  updated_at?: string
}

export interface Video {
  id: string
  name: string
  title: string
  year?: number
  season?: number
  episode?: number
  size: number
  mime_type?: string
  duration?: number
  collection_id: string
  collection_name: string
  folder_path: string
  poster_file_id?: string
  backdrop_file_id?: string
  subtitles: Subtitle[]
  progress?: VideoProgress
  added_at?: string
}

/** 网盘索引里有、磁盘上却找不到的媒体文件（被移出存储目录或换了存储路径） */
export interface MissingMedia {
  count: number
  samples: string[]
}

export interface Playlist {
  id: number
  name: string
  description: string
  track_ids: string[]
  createTime: string
  updateTime: string
}

export interface FolderRef {
  id: string
  name: string
  path: string
}

export interface LibrarySettings {
  music_folders: FolderRef[]
  video_folders: FolderRef[]
}

export const getMusicLibrary = (): Promise<{ tracks: Track[]; scoped: boolean; missing: MissingMedia }> =>
  request.get('/media/music', { timeout: LIBRARY_TIMEOUT })

export const getVideoLibrary = (): Promise<{ videos: Video[]; scoped: boolean; missing: MissingMedia }> =>
  request.get('/media/videos', { timeout: LIBRARY_TIMEOUT })

export const getLyrics = (id: string): Promise<{ text: string }> => request.get(`/media/lyrics/${id}`)

export const saveProgress = (id: string, position: number, duration: number, started = false): Promise<void> =>
  request.put(`/media/progress/${id}`, { position, duration, started })

export const removeProgress = (id: string): Promise<void> => request.delete(`/media/progress/${id}`)

export const reportDurations = (items: { id: string; duration: number }[]): Promise<void> =>
  request.post('/media/durations', { items })

export const listPlaylists = (): Promise<Playlist[]> => request.get('/media/playlists')

export const createPlaylist = (name: string, trackIds: string[] = [], description = ''): Promise<Playlist> =>
  request.post('/media/playlists', { name, description, track_ids: trackIds })

export const updatePlaylist = (
  id: number,
  patch: { name?: string; description?: string; track_ids?: string[] }
): Promise<Playlist> => request.put(`/media/playlists/${id}`, patch)

export const deletePlaylist = (id: number): Promise<void> => request.delete(`/media/playlists/${id}`)

export const getLibrarySettings = (): Promise<LibrarySettings> => request.get('/media/settings')

export const saveLibrarySettings = (musicFolderIds: string[], videoFolderIds: string[]): Promise<LibrarySettings> =>
  request.put('/media/settings', { music_folder_ids: musicFolderIds, video_folder_ids: videoFolderIds })

function tokenParam(): string {
  const token = localStorage.getItem('token') || ''
  return token.startsWith('Bearer ') ? token.substring(7) : token
}

/** 媒体本体走网盘的下载接口：preview=true 返回 inline，且原生支持 Range 拖动 */
export const mediaStreamUrl = (fileId: string): string => getDownloadUrl(fileId, true)

/** 曲目封面：同目录的封面图优先（同一专辑共享缓存），否则取内嵌封面，都没有返回空串 */
export const trackCoverUrl = (track: Pick<Track, 'id' | 'cover_file_id' | 'has_embedded_cover'>): string => {
  if (track.cover_file_id) return getDownloadUrl(track.cover_file_id, true)
  if (track.has_embedded_cover) return `${SERVER_URL}/media/cover/${track.id}?token=${tokenParam()}`
  return ''
}
