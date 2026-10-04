import { trackCoverUrl, type Track } from '@/api/media'
import { timeValue } from '../utils/format'

// 「自动识别」的歌单都在这里由曲目派生：专辑、艺人、文件夹。
// 后端只给扁平的曲目表，分组规则放在前端，改起来不用动接口。

export interface Album {
  key: string
  title: string
  artist: string
  year?: number
  genre?: string
  cover: string
  tracks: Track[]
  addedAt: number
  playedAt: number
}

export interface Artist {
  key: string
  name: string
  cover: string
  albums: Album[]
  tracks: Track[]
}

export interface FolderList {
  id: string
  name: string
  path: string
  cover: string
  tracks: Track[]
}

const VARIOUS_ARTISTS = '群星'

/**
 * 有专辑艺人标签时按「专辑 + 专辑艺人」归组；没有时退回「专辑 + 所在文件夹」，
 * 否则合辑里每首歌的艺人不同，会被拆成一堆只有一首歌的专辑。
 */
function albumKey(track: Track): string {
  const album = track.album.toLowerCase()
  return track.album_artist ? `${album}::a:${track.album_artist.toLowerCase()}` : `${album}::f:${track.folder_id}`
}

function compareAlbumOrder(a: Track, b: Track): number {
  return (a.disc_no || 1) - (b.disc_no || 1) || (a.track_no || 0) - (b.track_no || 0) || a.title.localeCompare(b.title, 'zh-CN')
}

export function buildAlbums(tracks: Track[]): Album[] {
  const groups = new Map<string, Track[]>()
  for (const track of tracks) {
    const key = albumKey(track)
    const group = groups.get(key)
    if (group) group.push(track)
    else groups.set(key, [track])
  }
  const albums: Album[] = []
  for (const [key, items] of groups) {
    items.sort(compareAlbumOrder)
    const artists = new Set(items.map(track => track.artist))
    const tagged = items.find(track => track.album_artist)?.album_artist
    albums.push({
      key,
      title: items[0].album,
      artist: tagged || (artists.size === 1 ? items[0].artist : VARIOUS_ARTISTS),
      year: items.find(track => track.year)?.year,
      genre: items.find(track => track.genre)?.genre,
      cover: trackCoverUrl(items.find(track => track.cover_file_id || track.has_embedded_cover) || items[0]),
      tracks: items,
      addedAt: Math.max(...items.map(track => timeValue(track.added_at))),
      playedAt: Math.max(...items.map(track => timeValue(track.last_played_at)))
    })
  }
  return albums.sort((a, b) => a.title.localeCompare(b.title, 'zh-CN'))
}

export function buildArtists(albums: Album[], tracks: Track[]): Artist[] {
  const byName = new Map<string, Artist>()
  const ensure = (name: string) => {
    const key = name.toLowerCase()
    let artist = byName.get(key)
    if (!artist) {
      artist = { key, name, cover: '', albums: [], tracks: [] }
      byName.set(key, artist)
    }
    return artist
  }
  for (const track of tracks) ensure(track.artist).tracks.push(track)
  for (const album of albums) {
    if (album.artist === VARIOUS_ARTISTS) continue
    const artist = ensure(album.artist)
    artist.albums.push(album)
    if (!artist.cover && album.cover) artist.cover = album.cover
  }
  for (const artist of byName.values()) {
    if (!artist.cover) {
      const withCover = artist.tracks.find(track => track.cover_file_id || track.has_embedded_cover)
      if (withCover) artist.cover = trackCoverUrl(withCover)
    }
  }
  return [...byName.values()]
    .filter(artist => artist.tracks.length > 0)
    .sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
}

export function buildFolders(tracks: Track[]): FolderList[] {
  const byId = new Map<string, FolderList>()
  for (const track of tracks) {
    let folder = byId.get(track.folder_id)
    if (!folder) {
      folder = { id: track.folder_id, name: track.folder_name, path: track.folder_path || '我的网盘', cover: '', tracks: [] }
      byId.set(track.folder_id, folder)
    }
    folder.tracks.push(track)
    if (!folder.cover && (track.cover_file_id || track.has_embedded_cover)) folder.cover = trackCoverUrl(track)
  }
  for (const folder of byId.values()) folder.tracks.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN', { numeric: true }))
  return [...byId.values()].sort((a, b) => a.path.localeCompare(b.path, 'zh-CN'))
}

export function totalDuration(tracks: Track[]): number {
  return tracks.reduce((sum, track) => sum + (track.duration || 0), 0)
}

export function matches(track: Track, keyword: string): boolean {
  const k = keyword.toLowerCase()
  return [track.title, track.artist, track.album, track.name].some(value => value.toLowerCase().includes(k))
}
