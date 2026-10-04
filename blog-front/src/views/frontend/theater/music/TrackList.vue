<template>
  <div class="text-sm">
    <div v-if="variant === 'list'" :class="[gridClass, 'border-b border-black/5 dark:border-white/10 px-3 pb-2 text-xs font-medium text-black/45 dark:text-white/45']">
      <span>歌曲</span>
      <span class="max-md:hidden">专辑</span>
      <span class="text-right">时长</span>
      <span></span>
    </div>

    <div
      v-for="(track, i) in tracks"
      :key="`${track.id}-${i}`"
      :class="[
        gridClass,
        'group cursor-default rounded-xl px-3 py-2 transition-colors',
        i % 2 === 1 && variant === 'album' ? 'bg-black/[0.02] dark:bg-white/[0.03]' : '',
        isCurrent(track) ? 'glass' : 'border border-transparent hover:bg-black/5 focus-within:bg-black/5 dark:hover:bg-white/[0.07] dark:focus-within:bg-white/[0.07]',
        dragOver === i ? 'ring-1 ring-[#fa2d48]' : ''
      ]"
      :draggable="!!playlist"
      @dblclick="emit('play', i)"
      @dragstart="dragFrom = i"
      @dragover.prevent="dragOver = i"
      @dragleave="dragOver = dragOver === i ? null : dragOver"
      @drop.prevent="drop(i)"
      @dragend="endDrag"
    >
      <!-- 序号 / 封面列：悬停时换成播放按钮，正在播放时显示跳动的音柱。
           遥控器没有悬停：焦点落在这一行的按钮上时同样露出播放键（group-focus-within / focus-visible） -->
      <div class="flex min-w-0 items-center gap-3">
        <div class="relative flex shrink-0 items-center justify-center" :class="variant === 'album' ? 'w-6' : 'size-10'">
          <Artwork v-if="variant === 'list'" :src="trackCoverUrl(track)" :seed="track.album" class="size-10 rounded" />
          <span v-else class="tabular-nums text-black/45 dark:text-white/45 group-hover:invisible group-focus-within:invisible" :class="isCurrent(track) && 'invisible'">{{ track.track_no || i + 1 }}</span>
          <button
            class="absolute inset-0 flex cursor-pointer items-center justify-center rounded border-none p-0 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            :class="variant === 'list' ? 'bg-black/50 text-white' : 'bg-transparent text-zinc-900 dark:text-white'"
            title="播放"
            data-nav-reveal
            @click="isCurrent(track) ? player.toggle() : emit('play', i)"
          >
            <PauseIcon v-if="isCurrent(track) && player.playing" class="size-4" />
            <PlayIcon v-else class="size-4" />
          </button>
          <span v-if="isCurrent(track) && player.playing" class="pointer-events-none absolute inset-0 flex items-end justify-center gap-[2px] pb-[30%] group-hover:hidden group-focus-within:hidden" :class="variant === 'list' && 'bg-black/40 rounded'">
            <span class="h-3.5 w-[3px] origin-bottom animate-equalizer rounded-sm bg-[#fa2d48]"></span>
            <span class="h-3.5 w-[3px] origin-bottom animate-equalizer rounded-sm bg-[#fa2d48] [animation-delay:-0.3s] [animation-duration:0.7s]"></span>
            <span class="h-3.5 w-[3px] origin-bottom animate-equalizer rounded-sm bg-[#fa2d48] [animation-delay:-0.6s] [animation-duration:1.1s]"></span>
          </span>
        </div>
        <div class="min-w-0">
          <p class="m-0 truncate font-medium" :class="isCurrent(track) ? 'text-[#fa2d48]' : 'text-zinc-900 dark:text-white'">{{ track.title }}</p>
          <p v-if="variant === 'list' || showArtist(track)" class="m-0 truncate text-xs text-black/50 dark:text-white/50">
            <button class="cursor-pointer border-none bg-transparent p-0 text-inherit hover:text-zinc-900 dark:hover:text-white hover:underline" @click="goToArtist(router, track)">{{ track.artist }}</button>
          </p>
        </div>
      </div>

      <button v-if="variant === 'list'" class="min-w-0 cursor-pointer truncate border-none bg-transparent p-0 text-left text-black/55 dark:text-white/55 hover:text-zinc-900 dark:hover:text-white hover:underline max-md:hidden" @click="goToAlbum(router, track)">
        {{ track.album }}
      </button>

      <span class="text-right text-xs tabular-nums text-black/45 dark:text-white/45">{{ track.duration ? formatTime(track.duration) : '—' }}</span>

      <el-dropdown trigger="click" placement="bottom-end" popper-class="glass-popup">
        <button class="flex size-7 cursor-pointer items-center justify-center rounded-full border-none bg-transparent text-black/60 dark:text-white/60 opacity-0 transition hover:bg-black/5 dark:hover:bg-white/10 hover:text-zinc-900 dark:hover:text-white group-hover:opacity-100 focus-visible:opacity-100 max-md:opacity-100" title="更多" data-nav-reveal>
          <MoreIcon class="size-4" />
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item @click="player.playNext([track])">下一首播放</el-dropdown-item>
            <el-dropdown-item @click="player.addToQueue([track])">添加到待播清单</el-dropdown-item>
            <el-dropdown-item @click="openAdd(track)">添加到歌单…</el-dropdown-item>
            <el-dropdown-item v-if="playlist" @click="removeFromPlaylist(i)">从歌单中移除</el-dropdown-item>
            <el-dropdown-item divided @click="goToAlbum(router, track)">前往专辑</el-dropdown-item>
            <el-dropdown-item @click="goToArtist(router, track)">前往艺人</el-dropdown-item>
            <el-dropdown-item @click="download(track)">下载</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <AddToPlaylistDialog v-model="addOpen" :tracks="addTracks" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMusicPlayerStore, useMusicStore } from '@/store'
import { trackCoverUrl, type Playlist, type Track } from '@/api/media'
import { getDownloadUrl } from '@/api/file'
import Artwork from '../components/Artwork.vue'
import AddToPlaylistDialog from './AddToPlaylistDialog.vue'
import { MoreIcon, PauseIcon, PlayIcon } from '../components/icons'
import { formatTime } from '../utils/format'
import { probeDurations } from '../utils/durationProbe'
import { goToAlbum, goToArtist } from './links'

const props = withDefaults(defineProps<{
  tracks: Track[]
  /** album：专辑内页，显示音轨号；list：通用列表，显示封面、艺人与专辑列 */
  variant?: 'album' | 'list'
  /** 专辑艺人，曲目艺人与之不同时（合辑）才在标题下显示艺人 */
  albumArtist?: string
  /** 传入时可拖动排序、可从歌单移除 */
  playlist?: Playlist
}>(), { variant: 'list', albumArtist: '', playlist: undefined })

const emit = defineEmits<{ (e: 'play', index: number): void }>()

const router = useRouter()
const player = useMusicPlayerStore()
const music = useMusicStore()

const gridClass = computed(() =>
  props.variant === 'list'
    ? 'grid items-center gap-4 grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)_56px_28px] max-md:grid-cols-[minmax(0,1fr)_48px_28px]'
    : 'grid items-center gap-4 grid-cols-[minmax(0,1fr)_56px_28px]'
)

// 列表出现在屏幕上时，顺带把缺失的时长探测出来
watch(() => props.tracks, tracks => probeDurations(tracks.slice(0, 300)), { immediate: true })

function isCurrent(track: Track) {
  return player.current?.id === track.id
}

function showArtist(track: Track) {
  return !!props.albumArtist && track.artist !== props.albumArtist
}

const addOpen = ref(false)
const addTracks = ref<Track[]>([])
function openAdd(track: Track) {
  addTracks.value = [track]
  addOpen.value = true
}

function download(track: Track) {
  window.open(getDownloadUrl(track.id), '_blank')
}

async function removeFromPlaylist(index: number) {
  if (!props.playlist) return
  const ids = props.tracks.map(track => track.id)
  ids.splice(index, 1)
  await music.setPlaylistTracks(props.playlist, ids)
}

// ---- 歌单内拖动排序 ----
const dragFrom = ref<number | null>(null)
const dragOver = ref<number | null>(null)

function endDrag() {
  dragFrom.value = null
  dragOver.value = null
}

async function drop(target: number) {
  const from = dragFrom.value
  dragFrom.value = dragOver.value = null
  if (!props.playlist || from === null || from === target) return
  const ids = props.tracks.map(track => track.id)
  const [moved] = ids.splice(from, 1)
  ids.splice(target, 0, moved)
  await music.setPlaylistTracks(props.playlist, ids)
}
</script>
