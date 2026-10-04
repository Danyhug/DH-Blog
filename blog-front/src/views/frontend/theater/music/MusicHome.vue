<template>
  <!-- 音乐（Apple Music 风格）：左侧资料库导航，右侧内容区。
       视图由路由 query 驱动（?view=album&id=…），浏览器后退可以逐级返回。 -->
  <div class="flex h-screen gap-3 p-3 text-zinc-900 dark:text-white max-md:gap-0 max-md:p-0" :class="player.current && 'pb-[100px] max-md:pb-[84px]'">
    <!-- ===== 侧边栏：悬浮的厚玻璃面板 ===== -->
    <aside class="glass-thick flex w-[250px] shrink-0 flex-col overflow-hidden rounded-[26px] max-md:hidden">
      <div class="flex items-center gap-2 px-5 pb-4 pt-6">
        <span class="flex size-8 items-center justify-center rounded-[10px] bg-gradient-to-br from-[#fb5c74] to-[#fa233b] shadow-[0_4px_16px_rgba(250,45,72,0.5),inset_0_1px_0_rgba(255,255,255,0.4)]">
          <MusicNoteIcon class="size-4 text-white" />
        </span>
        <span class="text-lg font-semibold tracking-tight">音乐</span>
      </div>

      <div class="relative mx-4 mb-4">
        <SearchIcon class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-black/40 dark:text-white/40" />
        <input
          v-model="searchInput"
          type="search"
          placeholder="搜索歌曲、专辑、艺人"
          class="glass-button w-full rounded-full py-1.5 pl-8 pr-3 text-sm text-zinc-900 dark:text-white outline-none placeholder:text-black/40 dark:placeholder:text-white/40 focus:border-[#fa2d48]/70"
        />
      </div>

      <nav class="flex-1 overflow-y-auto px-3 pb-4 text-sm">
        <button v-for="item in primaryNav" :key="item.view" :class="navClass(item.view)" @click="go(item.view)">
          <component :is="item.icon" class="size-[18px] text-[#fa2d48]" />{{ item.label }}
        </button>

        <p :class="sectionTitle">资料库</p>
        <button v-for="item in libraryNav" :key="item.view" :class="navClass(item.view)" @click="go(item.view)">
          <component :is="item.icon" class="size-[18px] text-[#fa2d48]" />{{ item.label }}
        </button>

        <div class="flex items-center justify-between pr-1">
          <p :class="sectionTitle">歌单</p>
          <button class="mt-5 flex size-6 cursor-pointer items-center justify-center rounded-full border-none bg-transparent text-black/50 dark:text-white/50 hover:bg-black/5 dark:hover:bg-white/10 hover:text-zinc-900 dark:hover:text-white" title="新建歌单" @click="newPlaylist">
            <PlusIcon class="size-4" />
          </button>
        </div>
        <p v-if="!music.playlists.length" class="m-0 px-3 py-1 text-xs text-black/35 dark:text-white/35">还没有歌单</p>
        <button
          v-for="playlist in music.playlists"
          :key="playlist.id"
          :class="navClass('playlist', String(playlist.id))"
          @click="go('playlist', playlist.id)"
        >
          <QueueIcon class="size-[18px] shrink-0 text-black/50 dark:text-white/50" /><span class="truncate">{{ playlist.name }}</span>
        </button>
      </nav>

      <div class="flex flex-col gap-0.5 border-t border-black/5 dark:border-white/[0.08] p-2 text-sm">
        <router-link :to="{ name: 'TheaterVideos' }" :class="footerLink"><FilmIcon class="size-4" />影视</router-link>
        <button :class="footerLink" @click="theater.openSettings()"><SettingsIcon class="size-4" />媒体库设置</button>
        <ThemeToggle v-slot="{ dark }" :class="footerLink" icon-class="size-4">{{ dark ? '日间模式' : '夜间模式' }}</ThemeToggle>
        <router-link to="/webdav" :class="footerLink"><FolderIcon class="size-4" />返回网盘</router-link>
      </div>
    </aside>

    <!-- ===== 内容区 ===== -->
    <!-- 内容区本身也是一块大玻璃，氛围背景（正在播放的封面）从后面透出来 -->
    <main ref="mainRef" class="glass min-w-0 flex-1 overflow-y-auto rounded-[26px] max-md:rounded-none max-md:border-0">
      <!-- 窄屏顶栏：侧边栏折叠成横向标签 -->
      <div class="glass-thick sticky top-0 z-10 flex flex-col gap-2 rounded-b-3xl border-t-0 px-4 pb-2 pt-3 md:hidden">
        <div class="flex items-center gap-2">
          <router-link :to="{ name: 'TheaterVideos' }" class="text-black/60! dark:text-white/60!"><FilmIcon class="size-5" /></router-link>
          <input v-model="searchInput" type="search" placeholder="搜索" class="glass-button min-w-0 flex-1 rounded-full px-3 py-1.5 text-sm text-zinc-900 dark:text-white outline-none" />
          <ThemeToggle class="flex cursor-pointer border-none bg-transparent p-0 text-black/60 dark:text-white/60" icon-class="size-5" />
          <button class="flex cursor-pointer border-none bg-transparent p-0 text-black/60 dark:text-white/60" @click="theater.openSettings()"><SettingsIcon class="size-5" /></button>
        </div>
        <div class="flex gap-2 overflow-x-auto pb-1">
          <button v-for="item in [...primaryNav, ...libraryNav]" :key="item.view" :class="['shrink-0 cursor-pointer rounded-full px-3 py-1 text-xs', view === item.view ? 'border-none bg-[#fa2d48] text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.35)]' : 'glass-button text-black/75 dark:text-white/75']" @click="go(item.view)">
            {{ item.label }}
          </button>
          <button v-for="playlist in music.playlists" :key="playlist.id" :class="['shrink-0 cursor-pointer rounded-full px-3 py-1 text-xs', view === 'playlist' && id === String(playlist.id) ? 'border-none bg-[#fa2d48] text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.35)]' : 'glass-button text-black/75 dark:text-white/75']" @click="go('playlist', playlist.id)">
            {{ playlist.name }}
          </button>
        </div>
      </div>

      <div class="mx-auto max-w-[1500px] px-10 pb-16 pt-8 max-md:px-4 max-md:pt-4">
        <!-- 加载 -->
        <div v-if="!music.loaded" class="flex h-[60vh] flex-col items-center justify-center gap-4 text-black/50 dark:text-white/50">
          <span class="glass flex size-16 items-center justify-center rounded-3xl"><span class="size-8 animate-spin rounded-full border-2 border-black/10 dark:border-white/15 border-t-[#fa2d48]"></span></span>
          <p class="m-0 text-sm">{{ music.loading ? '正在整理音乐库…首次打开需要读取每首歌的标签，曲目多时请稍候' : '音乐库加载失败' }}</p>
          <button v-if="!music.loading" :class="pillButton" @click="music.load(true)">重试</button>
        </div>

        <!-- 空库 -->
        <div v-else-if="!music.tracks.length" class="flex h-[60vh] flex-col items-center justify-center gap-3 text-center">
          <span class="flex size-20 items-center justify-center rounded-[24px] bg-gradient-to-br from-[#fb5c74] to-[#fa233b] shadow-[0_12px_40px_rgba(250,45,72,0.5),inset_0_1px_0_rgba(255,255,255,0.45)]"><MusicNoteIcon class="size-10 text-white" /></span>
          <h2 class="m-0 mt-3 text-2xl font-bold">音乐库还是空的</h2>
          <p class="m-0 max-w-md text-sm leading-relaxed text-black/55 dark:text-white/55">
            {{ music.scoped ? '指定的音乐文件夹里没有找到音频文件。' : '把 MP3、FLAC、M4A 等音频上传到网盘任意位置，这里会自动识别成专辑与艺人。' }}
          </p>
          <MissingNotice :missing="music.missing" kind="音频" class="mt-2 max-w-xl" />
          <div class="mt-3 flex gap-3">
            <router-link to="/webdav" :class="pillButton">去网盘上传</router-link>
            <button :class="pillButton" @click="theater.openSettings()">媒体库设置</button>
          </div>
        </div>

        <template v-else>
          <!-- ===== 立即收听 ===== -->
          <template v-if="view === 'home'">
            <h1 :class="pageTitle">立即收听</h1>
            <MissingNotice :missing="music.missing" kind="音频" class="-mt-2 mb-6" />
            <section class="mb-10 grid grid-cols-3 gap-4 max-lg:grid-cols-1">
              <button :class="heroCard" class="from-[#fa2d48] to-[#a3123a]" @click="player.playList(music.tracks, 0, { shuffle: true })">
                <ShuffleIcon class="size-7" />
                <span class="text-xl font-bold">随机播放全部</span>
                <span class="text-sm text-white/75">{{ music.tracks.length }} 首歌曲</span>
              </button>
              <button :class="heroCard" class="from-[#5e5ce6] to-[#2c2a8f]" @click="playTop">
                <SparkIcon class="size-7" />
                <span class="text-xl font-bold">常听歌曲</span>
                <span class="text-sm text-white/75">{{ topTracks.length ? `按播放次数排序的 ${topTracks.length} 首` : '多听几首就会出现在这里' }}</span>
              </button>
              <button :class="heroCard" class="from-[#30b0c7] to-[#0f5f73]" @click="playRecentlyAdded">
                <ClockIcon class="size-7" />
                <span class="text-xl font-bold">最近添加</span>
                <span class="text-sm text-white/75">最新放进网盘的 50 首</span>
              </button>
            </section>

            <ShelfSection v-if="recentlyPlayed.length" title="最近播放" @more="go('albums')">
              <AlbumTile v-for="album in recentlyPlayed" :key="album.key" :title="album.title" :subtitle="album.artist" :cover="album.cover" :seed="album.title" class="w-[180px] shrink-0 max-md:w-[140px]" @open="go('album', album.key)" @play="playAlbum(album)" />
            </ShelfSection>
            <ShelfSection v-if="topTracks.length" title="常听歌曲" @more="go('songs')">
              <div class="grid w-full grid-flow-col grid-rows-4 gap-x-6 [grid-auto-columns:minmax(280px,1fr)]">
                <QueueRow v-for="(track, i) in topTracks.slice(0, 12)" :key="track.id" :track="track" :active="player.current?.id === track.id" @play="player.playList(topTracks, i)" />
              </div>
            </ShelfSection>
            <ShelfSection title="最近添加" @more="go('recent')">
              <AlbumTile v-for="album in recentlyAddedAlbums.slice(0, 14)" :key="album.key" :title="album.title" :subtitle="album.artist" :cover="album.cover" :seed="album.title" class="w-[180px] shrink-0 max-md:w-[140px]" @open="go('album', album.key)" @play="playAlbum(album)" />
            </ShelfSection>
            <ShelfSection title="文件夹 · 自动识别" @more="go('folders')">
              <AlbumTile v-for="folder in folders.slice(0, 14)" :key="folder.id" :title="folder.name" :subtitle="`${folder.tracks.length} 首歌曲`" :cover="folder.cover" :seed="folder.path" :icon="FolderIcon" class="w-[180px] shrink-0 max-md:w-[140px]" @open="go('folder', folder.id || 'root')" @play="player.playList(folder.tracks)" />
            </ShelfSection>
          </template>

          <!-- ===== 最近添加 / 专辑 ===== -->
          <template v-else-if="view === 'recent' || view === 'albums'">
            <h1 :class="pageTitle">{{ view === 'recent' ? '最近添加' : '专辑' }}</h1>
            <div :class="tileGrid">
              <AlbumTile v-for="album in view === 'recent' ? recentlyAddedAlbums : albums" :key="album.key" :title="album.title" :subtitle="album.artist" :cover="album.cover" :seed="album.title" @open="go('album', album.key)" @play="playAlbum(album)" />
            </div>
          </template>

          <!-- ===== 艺人 ===== -->
          <template v-else-if="view === 'artists'">
            <h1 :class="pageTitle">艺人</h1>
            <div :class="tileGrid">
              <AlbumTile v-for="artist in artists" :key="artist.key" round :title="artist.name" :subtitle="`${artist.tracks.length} 首歌曲`" :cover="artist.cover" :seed="artist.name" :icon="MicIcon" @open="go('artist', artist.key)" />
            </div>
          </template>

          <!-- ===== 歌曲 ===== -->
          <template v-else-if="view === 'songs'">
            <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
              <h1 :class="[pageTitle, 'mb-0']">歌曲</h1>
              <div class="flex items-center gap-3">
                <el-select v-model="songSort" size="small" class="w-32" popper-class="glass-popup">
                  <el-option label="按标题" value="title" />
                  <el-option label="按艺人" value="artist" />
                  <el-option label="按专辑" value="album" />
                  <el-option label="最近添加" value="added" />
                  <el-option label="播放次数" value="plays" />
                </el-select>
                <button :class="pillButton" @click="player.playList(sortedSongs, 0, { shuffle: true })"><ShuffleIcon class="size-4" />随机播放</button>
              </div>
            </div>
            <TrackList :tracks="sortedSongs" @play="i => player.playList(sortedSongs, i)" />
          </template>

          <!-- ===== 文件夹 ===== -->
          <template v-else-if="view === 'folders'">
            <h1 :class="pageTitle">文件夹</h1>
            <p class="-mt-4 mb-6 text-sm text-black/50 dark:text-white/50">网盘里每个含有音频的文件夹都会自动识别成一个列表，可以一键存为歌单。</p>
            <div :class="tileGrid">
              <AlbumTile v-for="folder in folders" :key="folder.id" :title="folder.name" :subtitle="`${folder.path} · ${folder.tracks.length} 首`" :cover="folder.cover" :seed="folder.path" :icon="FolderIcon" @open="go('folder', folder.id || 'root')" @play="player.playList(folder.tracks)" />
            </div>
          </template>

          <!-- ===== 专辑详情 ===== -->
          <template v-else-if="view === 'album'">
            <template v-if="currentAlbum">
              <CollectionHeader
                kind="专辑"
                :title="currentAlbum.title"
                :subtitle="currentAlbum.artist"
                subtitle-clickable
                :meta="albumMeta(currentAlbum)"
                :covers="currentAlbum.cover ? [currentAlbum.cover] : []"
                @subtitle="go('artist', currentAlbum.artist.toLowerCase())"
                @play="playAlbum(currentAlbum)"
                @shuffle="player.playList(currentAlbum.tracks, 0, { shuffle: true })"
              >
                <template #actions>
                  <button :class="iconPill" title="添加到歌单" @click="openAdd(currentAlbum.tracks)"><PlusIcon class="size-4" /></button>
                </template>
              </CollectionHeader>
              <div class="mt-8">
                <TrackList variant="album" :tracks="currentAlbum.tracks" :album-artist="currentAlbum.artist" @play="i => player.playList(currentAlbum!.tracks, i)" />
              </div>
              <ShelfSection v-if="moreFromArtist.length" :title="`更多 ${currentAlbum.artist} 的专辑`" class="mt-12" @more="go('artist', currentAlbum.artist.toLowerCase())">
                <AlbumTile v-for="album in moreFromArtist" :key="album.key" :title="album.title" :subtitle="album.year ? String(album.year) : ''" :cover="album.cover" :seed="album.title" class="w-[180px] shrink-0 max-md:w-[140px]" @open="go('album', album.key)" @play="playAlbum(album)" />
              </ShelfSection>
            </template>
            <NotFound v-else />
          </template>

          <!-- ===== 艺人详情 ===== -->
          <template v-else-if="view === 'artist'">
            <template v-if="currentArtist">
              <CollectionHeader
                kind="艺人"
                round
                :icon="MicIcon"
                :title="currentArtist.name"
                :meta="`${currentArtist.albums.length} 张专辑 · ${currentArtist.tracks.length} 首歌曲`"
                :covers="currentArtist.cover ? [currentArtist.cover] : []"
                @play="player.playList(artistTopTracks)"
                @shuffle="player.playList(currentArtist.tracks, 0, { shuffle: true })"
              />
              <h2 :class="sectionHeading" class="mt-10">热门歌曲</h2>
              <TrackList :tracks="artistTopTracks" @play="i => player.playList(artistTopTracks, i)" />
              <template v-if="currentArtist.albums.length">
                <h2 :class="sectionHeading" class="mt-10">专辑</h2>
                <div :class="tileGrid">
                  <AlbumTile v-for="album in currentArtist.albums" :key="album.key" :title="album.title" :subtitle="album.year ? String(album.year) : `${album.tracks.length} 首`" :cover="album.cover" :seed="album.title" @open="go('album', album.key)" @play="playAlbum(album)" />
                </div>
              </template>
            </template>
            <NotFound v-else />
          </template>

          <!-- ===== 文件夹详情 ===== -->
          <template v-else-if="view === 'folder'">
            <template v-if="currentFolder">
              <CollectionHeader
                kind="文件夹 · 自动识别"
                :icon="FolderIcon"
                :title="currentFolder.name"
                :subtitle="currentFolder.path"
                :meta="listMeta(currentFolder.tracks)"
                :covers="currentFolder.cover ? [currentFolder.cover] : []"
                @play="player.playList(currentFolder.tracks)"
                @shuffle="player.playList(currentFolder.tracks, 0, { shuffle: true })"
              >
                <template #actions>
                  <button :class="pillButton" @click="saveFolderAsPlaylist(currentFolder)"><PlusIcon class="size-4" />存为歌单</button>
                </template>
              </CollectionHeader>
              <div class="mt-8">
                <TrackList :tracks="currentFolder.tracks" @play="i => player.playList(currentFolder!.tracks, i)" />
              </div>
            </template>
            <NotFound v-else />
          </template>

          <!-- ===== 歌单详情 ===== -->
          <template v-else-if="view === 'playlist'">
            <template v-if="currentPlaylist">
              <CollectionHeader
                kind="歌单"
                :icon="QueueIcon"
                :title="currentPlaylist.name"
                :subtitle="currentPlaylist.description"
                :meta="listMeta(currentPlaylistTracks)"
                :covers="playlistCovers"
                :playable="currentPlaylistTracks.length > 0"
                @play="player.playList(currentPlaylistTracks)"
                @shuffle="player.playList(currentPlaylistTracks, 0, { shuffle: true })"
              >
                <template #actions>
                  <el-dropdown trigger="click" popper-class="glass-popup">
                    <button :class="iconPill" title="更多"><MoreIcon class="size-4" /></button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item @click="editPlaylist(currentPlaylist)">编辑名称与描述</el-dropdown-item>
                        <el-dropdown-item @click="player.addToQueue(currentPlaylistTracks)">添加到待播清单</el-dropdown-item>
                        <el-dropdown-item divided @click="removePlaylist(currentPlaylist)"><span class="text-[#ff453a]">删除歌单</span></el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </template>
              </CollectionHeader>
              <div class="mt-8">
                <p v-if="!currentPlaylistTracks.length" class="py-10 text-center text-sm text-black/45 dark:text-white/45">
                  歌单还是空的。在任意歌曲的「…」菜单里选择「添加到歌单」即可加入。
                </p>
                <template v-else>
                  <p class="mb-2 text-xs text-black/35 dark:text-white/35">拖动歌曲可以调整顺序</p>
                  <TrackList :tracks="currentPlaylistTracks" :playlist="currentPlaylist" @play="i => player.playList(currentPlaylistTracks, i)" />
                </template>
              </div>
            </template>
            <NotFound v-else />
          </template>

          <!-- ===== 搜索 ===== -->
          <template v-else-if="view === 'search'">
            <h1 :class="pageTitle">“{{ searchKeyword }}”的搜索结果</h1>
            <p v-if="!searchTracks.length && !searchAlbums.length && !searchArtists.length" class="text-black/50 dark:text-white/50">没有找到相关的歌曲、专辑或艺人</p>
            <template v-if="searchArtists.length">
              <h2 :class="sectionHeading">艺人</h2>
              <div :class="tileGrid" class="mb-10">
                <AlbumTile v-for="artist in searchArtists" :key="artist.key" round :title="artist.name" :cover="artist.cover" :seed="artist.name" :icon="MicIcon" @open="go('artist', artist.key)" />
              </div>
            </template>
            <template v-if="searchAlbums.length">
              <h2 :class="sectionHeading">专辑</h2>
              <div :class="tileGrid" class="mb-10">
                <AlbumTile v-for="album in searchAlbums" :key="album.key" :title="album.title" :subtitle="album.artist" :cover="album.cover" :seed="album.title" @open="go('album', album.key)" @play="playAlbum(album)" />
              </div>
            </template>
            <template v-if="searchTracks.length">
              <h2 :class="sectionHeading">歌曲</h2>
              <TrackList :tracks="searchTracks" @play="i => player.playList(searchTracks, i)" />
            </template>
          </template>
        </template>
      </div>
    </main>

    <AddToPlaylistDialog v-model="addOpen" :tracks="addTracks" />
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onActivated, onDeactivated, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMusicPlayerStore, useMusicStore } from '@/store'
import { trackCoverUrl, type Playlist, type Track } from '@/api/media'
import { notify } from '@/utils/notification'
import { useTheater } from '../context'
import AlbumTile from './AlbumTile.vue'
import CollectionHeader from './CollectionHeader.vue'
import TrackList from './TrackList.vue'
import QueueRow from './QueueRow.vue'
import ShelfSection from './ShelfSection.vue'
import AddToPlaylistDialog from './AddToPlaylistDialog.vue'
import MissingNotice from '../components/MissingNotice.vue'
import ThemeToggle from '@/components/Child/ThemeToggle.vue'
import { buildAlbums, buildArtists, buildFolders, matches, totalDuration, type Album, type FolderList } from './library'
import { musicLink, type MusicView } from './links'
import { formatTotal, timeValue } from '../utils/format'
import {
  AlbumIcon, ClockIcon, FilmIcon, FolderIcon, HomeIcon, MicIcon, MoreIcon, MusicNoteIcon,
  PlusIcon, QueueIcon, SearchIcon, SettingsIcon, ShuffleIcon, SparkIcon
} from '../components/icons'

const route = useRoute()
const router = useRouter()
const music = useMusicStore()
const player = useMusicPlayerStore()
const theater = useTheater()
const mainRef = ref<HTMLElement | null>(null)

// ---- 样式常量 ----
const pageTitle = 'mb-6 mt-0 text-[32px] font-bold tracking-tight max-md:text-2xl'
const sectionHeading = 'mb-4 mt-0 text-xl font-bold'
const sectionTitle = 'mb-1 mt-5 px-3 text-[11px] font-semibold uppercase tracking-wider text-black/40 dark:text-white/40'
const tileGrid = 'grid grid-cols-[repeat(auto-fill,minmax(170px,1fr))] gap-x-6 gap-y-8 max-md:grid-cols-2 max-md:gap-4'
const pillButton = 'glass-button flex cursor-pointer items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold text-zinc-900! dark:text-white! no-underline'
const iconPill = 'glass-button flex size-9 cursor-pointer items-center justify-center rounded-full text-[#fa2d48]'
// 渐变卡片叠一层玻璃高光：顶部亮边 + 左上反光，像一块有颜色的玻璃
const heroCard = 'relative flex h-36 cursor-pointer flex-col items-start justify-end gap-1 overflow-hidden rounded-[24px] border border-white/20 bg-gradient-to-br p-5 text-left text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.45),0_16px_40px_-12px_rgba(0,0,0,0.6)] transition-transform hover:scale-[1.02] active:scale-[0.99] before:pointer-events-none before:absolute before:inset-0 before:bg-[radial-gradient(120%_80%_at_0%_0%,rgba(255,255,255,0.35),transparent_55%)]'
const footerLink = 'flex cursor-pointer items-center gap-3 rounded-xl border-none bg-transparent px-3 py-1.5 text-left text-black/60! dark:text-white/60! no-underline transition-colors hover:bg-black/5 dark:hover:bg-white/[0.08] hover:text-zinc-900! dark:hover:text-white!'

const primaryNav = [{ view: 'home' as const, label: '立即收听', icon: HomeIcon }]
const libraryNav = [
  { view: 'recent' as const, label: '最近添加', icon: ClockIcon },
  { view: 'artists' as const, label: '艺人', icon: MicIcon },
  { view: 'albums' as const, label: '专辑', icon: AlbumIcon },
  { view: 'songs' as const, label: '歌曲', icon: MusicNoteIcon },
  { view: 'folders' as const, label: '文件夹', icon: FolderIcon }
]

const NotFound = defineComponent({
  render: () => h('p', { class: 'py-20 text-center text-black/50 dark:text-white/50' }, '内容不存在，可能已被移动或删除')
})

// ---- 路由状态 ----
const view = computed<MusicView>(() => (route.query.view as MusicView) || 'home')
const id = computed(() => String(route.query.id ?? ''))

function navClass(target: MusicView, targetId?: string) {
  const active = view.value === target && (targetId === undefined || id.value === targetId)
  return [
    'flex w-full cursor-pointer items-center gap-3 rounded-xl px-3 py-1.5 text-left text-[13px] transition-all',
    active ? 'glass-button text-zinc-900 dark:text-white' : 'border border-transparent bg-transparent text-black/80 dark:text-white/80 hover:bg-black/5 dark:hover:bg-white/[0.07]'
  ]
}

function go(target: MusicView, targetId?: string | number) {
  router.push(musicLink(target, targetId))
}

watch(() => route.fullPath, () => mainRef.value?.scrollTo({ top: 0 }))

// ---- 数据 ----
music.load()
watch(theater.libraryVersion, () => music.load(true))

const albums = computed(() => buildAlbums(music.tracks))
const artists = computed(() => buildArtists(albums.value, music.tracks))
const folders = computed(() => buildFolders(music.tracks))

const recentlyPlayed = computed(() => albums.value.filter(album => album.playedAt > 0).sort((a, b) => b.playedAt - a.playedAt).slice(0, 14))
const recentlyAddedAlbums = computed(() => albums.value.slice().sort((a, b) => b.addedAt - a.addedAt))
const topTracks = computed(() => music.tracks.filter(track => (track.play_count || 0) > 0).sort((a, b) => (b.play_count || 0) - (a.play_count || 0)).slice(0, 50))

const songSort = ref<'title' | 'artist' | 'album' | 'added' | 'plays'>('title')
const sortedSongs = computed(() => {
  const list = music.tracks.slice()
  const zh = (a: string, b: string) => a.localeCompare(b, 'zh-CN')
  switch (songSort.value) {
    case 'artist': return list.sort((a, b) => zh(a.artist, b.artist) || zh(a.title, b.title))
    case 'album': return list.sort((a, b) => zh(a.album, b.album) || (a.track_no || 0) - (b.track_no || 0))
    case 'added': return list.sort((a, b) => timeValue(b.added_at) - timeValue(a.added_at))
    case 'plays': return list.sort((a, b) => (b.play_count || 0) - (a.play_count || 0))
    default: return list.sort((a, b) => zh(a.title, b.title))
  }
})

const currentAlbum = computed(() => (view.value === 'album' ? albums.value.find(album => album.key === id.value) : undefined))
const moreFromArtist = computed(() => {
  const album = currentAlbum.value
  if (!album) return []
  return albums.value.filter(other => other.key !== album.key && other.artist === album.artist)
})

const currentArtist = computed(() => (view.value === 'artist' ? artists.value.find(artist => artist.key === id.value) : undefined))
const artistTopTracks = computed(() => {
  const artist = currentArtist.value
  if (!artist) return []
  return artist.tracks.slice().sort((a, b) => (b.play_count || 0) - (a.play_count || 0)).slice(0, 10)
})

const currentFolder = computed(() => {
  if (view.value !== 'folder') return undefined
  const folderId = id.value === 'root' ? '' : id.value
  return folders.value.find(folder => folder.id === folderId)
})

const currentPlaylist = computed(() => (view.value === 'playlist' ? music.playlists.find(playlist => String(playlist.id) === id.value) : undefined))
const currentPlaylistTracks = computed(() => (currentPlaylist.value ? music.playlistTracks(currentPlaylist.value) : []))
const playlistCovers = computed(() => {
  const covers: string[] = []
  for (const track of currentPlaylistTracks.value) {
    const cover = trackCoverUrl(track)
    if (cover && !covers.includes(cover)) covers.push(cover)
    if (covers.length === 4) break
  }
  return covers.length >= 4 ? covers : covers.slice(0, 1)
})

// ---- 氛围背景：专辑页用该专辑封面，其余用正在播放的封面，再不行取曲库里第一张 ----
const active = ref(false)
const ambientCover = computed(() => {
  if (currentAlbum.value?.cover) return currentAlbum.value.cover
  if (player.current) {
    const cover = trackCoverUrl(player.current)
    if (cover) return cover
  }
  return albums.value.find(album => album.cover)?.cover ?? ''
})
watch(ambientCover, cover => {
  if (active.value) theater.ambient.value = cover
})
onActivated(() => {
  active.value = true
  theater.ambient.value = ambientCover.value
  music.load()
})
onDeactivated(() => (active.value = false))

function albumMeta(album: Album) {
  return [album.genre, album.year, listMeta(album.tracks)].filter(Boolean).join(' · ')
}

function listMeta(tracks: Track[]) {
  const total = totalDuration(tracks)
  return `${tracks.length} 首歌曲${total ? ` · ${formatTotal(total)}` : ''}`
}

// ---- 搜索 ----
const searchInput = ref(view.value === 'search' ? String(route.query.q ?? '') : '')
const searchKeyword = computed(() => String(route.query.q ?? '').trim())
let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(searchInput, value => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    const keyword = value.trim()
    if (keyword) router.replace({ name: 'TheaterMusic', query: { view: 'search', q: keyword } })
    else if (view.value === 'search') router.replace(musicLink('home'))
  }, 250)
})
const searchTracks = computed(() => (searchKeyword.value ? music.tracks.filter(track => matches(track, searchKeyword.value)).slice(0, 100) : []))
const searchAlbums = computed(() => {
  const k = searchKeyword.value.toLowerCase()
  return k ? albums.value.filter(album => album.title.toLowerCase().includes(k) || album.artist.toLowerCase().includes(k)).slice(0, 24) : []
})
const searchArtists = computed(() => {
  const k = searchKeyword.value.toLowerCase()
  return k ? artists.value.filter(artist => artist.name.toLowerCase().includes(k)).slice(0, 12) : []
})

// ---- 播放 ----
function playAlbum(album: Album) {
  player.playList(album.tracks)
}

function playTop() {
  if (topTracks.value.length) player.playList(topTracks.value)
  else player.playList(music.tracks, 0, { shuffle: true })
}

function playRecentlyAdded() {
  const recent = music.tracks.slice().sort((a, b) => timeValue(b.added_at) - timeValue(a.added_at)).slice(0, 50)
  player.playList(recent)
}

// 从网盘预览页「在影院中播放」跳过来：?play=<文件ID>，以所在文件夹为上下文播放
watch([() => music.loaded, () => route.query.play], ([loaded, play]) => {
  if (!loaded || !play) return
  const track = music.trackById.get(String(play))
  if (track) {
    const folder = folders.value.find(item => item.id === track.folder_id)
    const list = folder?.tracks ?? [track]
    player.playList(list, Math.max(0, list.indexOf(track)))
    player.expanded = true
  } else {
    notify.warning('该音频不在音乐库中，可能被媒体库设置排除了')
  }
  router.replace({ name: 'TheaterMusic', query: { ...route.query, play: undefined } })
}, { immediate: true })

// ---- 歌单 ----
const addOpen = ref(false)
const addTracks = ref<Track[]>([])

function openAdd(tracks: Track[]) {
  addTracks.value = tracks
  addOpen.value = true
}

async function promptName(title: string, initial: string, confirmText: string) {
  const result = await ElMessageBox.prompt('歌单名称', title, {
    confirmButtonText: confirmText,
    cancelButtonText: '取消',
    customClass: 'glass-popup',
    inputValue: initial,
    inputValidator: value => (value.trim() ? true : '名称不能为空')
  }).catch(() => null)
  return result?.value.trim() || ''
}

async function newPlaylist() {
  const name = await promptName('新建歌单', '新建歌单', '创建')
  if (!name) return
  const playlist = await music.addPlaylist(name)
  go('playlist', playlist.id)
}

async function saveFolderAsPlaylist(folder: FolderList) {
  const name = await promptName('存为歌单', folder.name, '保存')
  if (!name) return
  const playlist = await music.addPlaylist(name, folder.tracks.map(track => track.id))
  notify.success(`已创建歌单「${playlist.name}」`)
  go('playlist', playlist.id)
}

async function editPlaylist(playlist: Playlist) {
  const name = await promptName('编辑歌单', playlist.name, '保存')
  if (!name) return
  const description = await ElMessageBox.prompt('一句话描述（可留空）', '歌单描述', {
    confirmButtonText: '保存',
    cancelButtonText: '跳过',
    customClass: 'glass-popup',
    inputValue: playlist.description
  }).then(result => result.value).catch(() => playlist.description)
  await music.renamePlaylist(playlist, name, description)
}

async function removePlaylist(playlist: Playlist) {
  const confirmed = await ElMessageBox.confirm(`确定删除歌单「${playlist.name}」吗？歌曲文件不会被删除。`, '删除歌单', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    customClass: 'glass-popup',
    type: 'warning'
  }).then(() => true).catch(() => false)
  if (!confirmed) return
  await music.removePlaylist(playlist)
  go('home')
}
</script>
