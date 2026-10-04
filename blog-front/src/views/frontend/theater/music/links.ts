import type { Router } from 'vue-router'
import type { Track } from '@/api/media'
import { buildAlbums } from './library'

// 音乐页用 query 表示当前视图（?view=album&id=…），这样浏览器后退能回到上一个视图。
export type MusicView =
  | 'home' | 'recent' | 'artists' | 'albums' | 'songs' | 'folders'
  | 'album' | 'artist' | 'folder' | 'playlist' | 'search'

export function musicLink(view: MusicView, id?: string | number) {
  return { name: 'TheaterMusic', query: id === undefined ? { view } : { view, id: String(id) } }
}

export function goToAlbum(router: Router, track: Track) {
  const [album] = buildAlbums([track])
  return router.push(musicLink('album', album.key))
}

export function goToArtist(router: Router, track: Track) {
  return router.push(musicLink('artist', track.artist.toLowerCase()))
}
