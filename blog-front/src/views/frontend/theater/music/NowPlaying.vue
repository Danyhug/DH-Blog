<template>
  <!-- 全屏「正在播放」：封面取色般的模糊背景、大封面、逐行滚动的同步歌词。
       和 Apple Music 一样压暗的封面色背景在日间也保持深色，根节点挂 dark 让子组件与玻璃都走夜间样式。 -->
  <div data-nav-scope class="dark fixed inset-0 z-[70] overflow-hidden bg-black text-white">
    <div class="absolute inset-0">
      <img v-if="cover" :src="cover" alt="" class="absolute inset-0 size-full scale-125 object-cover opacity-80 blur-[80px] saturate-[1.8]" />
      <div v-else class="absolute inset-0 opacity-70" :style="{ background: gradientFor(track.album) }"></div>
      <div class="absolute inset-0 bg-black/45"></div>
    </div>

    <div class="relative flex h-full flex-col">
      <header class="flex shrink-0 items-center justify-between px-6 pt-5 max-md:px-4">
        <button :class="roundButton" title="收起（Esc）" @click="player.expanded = false">
          <ChevronDownIcon class="size-6" />
        </button>
        <div class="glass flex rounded-full p-1 text-sm">
          <button v-for="tab in tabs" :key="tab.value" :class="[tabButton, player.panel === tab.value ? 'glass-button text-white' : 'border border-transparent bg-transparent text-white/60 hover:text-white']" @click="player.panel = tab.value">
            {{ tab.label }}
          </button>
        </div>
        <span class="size-10"></span>
      </header>

      <!-- short（横屏矮屏，如电视）：保持左右并排、封面按高度缩小、音量条收起（电视用遥控器调音量），一屏放下全部控件 -->
      <div class="flex min-h-0 flex-1 items-center gap-16 px-[6vw] pb-8 max-lg:flex-col max-lg:gap-6 max-lg:overflow-y-auto max-lg:px-6 max-lg:pt-6 short:flex-row! short:gap-10! short:overflow-hidden! short:px-[5vw]! short:pt-0! short:pb-4!">
        <!-- 左：封面与控制 -->
        <section class="flex w-full max-w-[460px] shrink-0 flex-col max-lg:mx-auto short:mx-0! short:max-w-[320px]!">
          <Artwork
            :src="cover"
            :seed="track.album"
            class="aspect-square w-full rounded-2xl shadow-[0_30px_80px_rgba(0,0,0,0.55)] transition-transform duration-500 ease-out short:mx-auto short:w-[38vh]!"
            :class="player.playing ? 'scale-100' : 'scale-[0.86]'"
          />
          <div class="mt-8 flex items-start justify-between gap-4 short:mt-3!">
            <div class="min-w-0">
              <h2 class="m-0 truncate text-[22px] font-bold">{{ track.title }}</h2>
              <p class="m-0 mt-1 truncate text-lg text-white/65">
                <button :class="linkButton" @click="openArtist">{{ track.artist }}</button>
                <span> — </span>
                <button :class="linkButton" @click="openAlbum">{{ track.album }}</button>
              </p>
            </div>
            <el-dropdown trigger="click" placement="bottom-end" popper-class="glass-popup">
              <button :class="roundButton"><MoreIcon class="size-5" /></button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="openAlbum">前往专辑</el-dropdown-item>
                  <el-dropdown-item @click="openArtist">前往艺人</el-dropdown-item>
                  <el-dropdown-item @click="addOpen = true">添加到歌单…</el-dropdown-item>
                  <el-dropdown-item @click="download">下载</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <div class="mt-6 short:mt-3!">
            <ProgressBar
              label="播放进度"
              :value="scrubRatio ?? ratio"
              :buffered="player.duration ? player.buffered / player.duration : 0"
              fill-class="bg-white/90"
              track-class="bg-white/25"
              height-class="h-1.5 group-hover/bar:h-2"
              drag-height-class="h-2"
              :knob="false"
              @scrub="r => (scrubRatio = r)"
              @seek="commitSeek"
            />
            <div class="mt-2 flex justify-between text-xs tabular-nums text-white/55">
              <span>{{ formatTime(scrubRatio === null ? player.currentTime : scrubRatio * player.duration) }}</span>
              <span>-{{ formatTime(Math.max(0, player.duration - player.currentTime)) }}</span>
            </div>
          </div>

          <div class="glass mt-5 flex items-center justify-between rounded-full px-6 py-3 short:mt-3! short:py-1.5!">
            <button :class="[iconButton, player.shuffle ? 'text-[#fa2d48]' : 'text-white/60']" title="随机播放" @click="player.toggleShuffle()">
              <ShuffleIcon class="size-5" />
            </button>
            <button :class="[iconButton, 'text-white']" title="上一首" @click="player.prev()"><PrevIcon class="size-9" /></button>
            <button ref="playRef" :class="[iconButton, 'text-white']" :title="player.playing ? '暂停' : '播放'" @click="player.toggle()">
              <PauseIcon v-if="player.playing" class="size-12" />
              <PlayIcon v-else class="size-12" />
            </button>
            <button :class="[iconButton, 'text-white']" title="下一首" @click="player.next()"><NextIcon class="size-9" /></button>
            <button :class="[iconButton, player.repeat !== 'off' ? 'text-[#fa2d48]' : 'text-white/60']" title="循环" @click="player.cycleRepeat()">
              <RepeatOneIcon v-if="player.repeat === 'one'" class="size-5" />
              <RepeatIcon v-else class="size-5" />
            </button>
          </div>

          <div class="mt-5 flex items-center gap-3 px-2 text-white/60 short:hidden">
            <MuteIcon class="size-4 shrink-0" />
            <ProgressBar class="flex-1" label="音量" :value="player.muted ? 0 : player.volume" fill-class="bg-white/90" track-class="bg-white/25" :knob="false" @scrub="player.setVolume" @seek="player.setVolume" />
            <VolumeIcon class="size-4 shrink-0" />
          </div>
        </section>

        <!-- 右：歌词 / 待播清单 -->
        <section class="relative h-full min-h-[50vh] w-full min-w-0 flex-1 short:min-h-0! short:h-[calc(100vh-110px)]!">
          <div
            v-if="player.panel === 'lyrics'"
            ref="lyricsRef"
            class="absolute inset-0 overflow-y-auto [mask-image:linear-gradient(transparent,black_15%,black_85%,transparent)] max-lg:relative max-lg:max-h-[60vh] short:absolute! short:max-h-none!"
            @wheel.passive="userScrolledAt = Date.now()"
            @touchmove.passive="userScrolledAt = Date.now()"
          >
            <div v-if="lyricsLoading" class="flex h-full items-center justify-center text-white/50">正在加载歌词…</div>
            <div v-else-if="!lyrics.lines.length" class="flex h-full flex-col items-center justify-center gap-2 text-white/50">
              <LyricsIcon class="size-10" />
              <p class="m-0 text-lg">暂无歌词</p>
              <p class="m-0 text-sm text-white/35">把同名的 .lrc 文件放在歌曲旁边即可自动识别</p>
            </div>
            <div v-else class="py-[40vh] max-lg:py-12 short:py-[30vh]!">
              <p
                v-for="(line, i) in lyrics.lines"
                :key="i"
                :data-line="i"
                class="m-0 origin-left cursor-pointer rounded-xl px-4 py-3 text-[28px] font-bold leading-snug transition-all duration-500 hover:bg-white/5 max-md:text-xl"
                :class="!lyrics.synced ? 'cursor-default text-white/85' : i === activeLine ? 'scale-100 text-white' : 'scale-[0.96] text-white/30 blur-[0.6px]'"
                @click="lyrics.synced && player.seek(line.time)"
              >
                {{ line.text || '♪' }}
              </p>
            </div>
          </div>

          <div v-else class="glass-thick absolute inset-0 overflow-y-auto rounded-[28px] p-4 max-lg:relative short:absolute!">
            <h3 class="m-0 mb-2 text-sm font-semibold uppercase tracking-wider text-white/50">正在播放</h3>
            <QueueRow :track="track" :active="true" />
            <div class="mb-2 mt-6 flex items-center justify-between">
              <h3 class="m-0 text-sm font-semibold uppercase tracking-wider text-white/50">接下来</h3>
              <button v-if="player.upNext.length" class="cursor-pointer border-none bg-transparent text-sm text-[#fa2d48] hover:underline" @click="player.clearUpNext()">清空</button>
            </div>
            <p v-if="!player.upNext.length" class="m-0 py-6 text-center text-sm text-white/40">待播清单是空的</p>
            <QueueRow
              v-for="(item, i) in player.upNext"
              :key="`${item.id}-${i}`"
              :track="item"
              removable
              @play="player.jumpTo(player.index + 1 + i)"
              @remove="player.removeFromQueue(player.index + 1 + i)"
            />
          </div>
        </section>
      </div>
    </div>

    <AddToPlaylistDialog v-model="addOpen" :tracks="[track]" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMusicPlayerStore } from '@/store'
import { getLyrics, trackCoverUrl } from '@/api/media'
import { getDownloadUrl } from '@/api/file'
import Artwork from '../components/Artwork.vue'
import ProgressBar from '../components/ProgressBar.vue'
import QueueRow from './QueueRow.vue'
import AddToPlaylistDialog from './AddToPlaylistDialog.vue'
import { formatTime, gradientFor } from '../utils/format'
import { activeLyricIndex, parseLyrics, type LyricLine } from '../utils/lrc'
import { goToAlbum, goToArtist } from './links'
import { useFocusRestore } from '../utils/tv'
import {
  ChevronDownIcon, LyricsIcon, MoreIcon, MuteIcon, NextIcon, PauseIcon, PlayIcon, PrevIcon,
  RepeatIcon, RepeatOneIcon, ShuffleIcon, VolumeIcon
} from '../components/icons'

const player = useMusicPlayerStore()
const router = useRouter()
const track = computed(() => player.current!)
const cover = computed(() => trackCoverUrl(track.value))
const ratio = computed(() => (player.duration ? player.currentTime / player.duration : 0))
const scrubRatio = ref<number | null>(null)
const addOpen = ref(false)

const tabs = [
  { value: 'lyrics' as const, label: '歌词' },
  { value: 'queue' as const, label: '待播清单' }
]
const roundButton = 'glass-button flex size-10 cursor-pointer items-center justify-center rounded-full text-white'
const tabButton = 'cursor-pointer rounded-full px-4 py-1.5 font-medium transition-all'
const iconButton = 'flex cursor-pointer items-center justify-center rounded-full border-none bg-transparent p-0 transition-transform hover:scale-110 focus-visible:scale-110 active:scale-95'
const linkButton = 'cursor-pointer border-none bg-transparent p-0 text-inherit hover:text-white hover:underline'

function commitSeek(r: number) {
  scrubRatio.value = null
  player.seek(r * player.duration)
}

// ---- 歌词 ----
const lyricsCache = new Map<string, { lines: LyricLine[]; synced: boolean }>()
const lyrics = ref<{ lines: LyricLine[]; synced: boolean }>({ lines: [], synced: false })
const lyricsLoading = ref(false)
const lyricsRef = ref<HTMLElement | null>(null)
// 用户手动滚动歌词后，暂停自动居中一会儿，免得刚滚上去就被拽回来
const userScrolledAt = ref(0)

watch(() => track.value?.id, async id => {
  lyrics.value = { lines: [], synced: false }
  const current = track.value
  if (!id || !current.has_lyrics) return
  const cached = lyricsCache.get(id)
  if (cached) {
    lyrics.value = cached
    return
  }
  lyricsLoading.value = true
  try {
    const { text } = await getLyrics(id)
    const parsed = parseLyrics(text || '')
    lyricsCache.set(id, parsed)
    if (track.value?.id === id) lyrics.value = parsed
  } catch {
    // 歌词只是锦上添花，失败时显示「暂无歌词」
  } finally {
    lyricsLoading.value = false
  }
}, { immediate: true })

const activeLine = computed(() => (lyrics.value.synced ? activeLyricIndex(lyrics.value.lines, player.currentTime) : -1))

watch([activeLine, () => player.panel], async () => {
  if (activeLine.value < 0 || Date.now() - userScrolledAt.value < 3000) return
  await nextTick()
  const line = lyricsRef.value?.querySelector<HTMLElement>(`[data-line="${activeLine.value}"]`)
  line?.scrollIntoView({ block: 'center', behavior: 'smooth' })
})

// ---- 跳转 ----
function openAlbum() {
  player.expanded = false
  goToAlbum(router, track.value)
}

function openArtist() {
  player.expanded = false
  goToArtist(router, track.value)
}

function download() {
  window.open(getDownloadUrl(track.value.id), '_blank')
}

function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') player.expanded = false
  if (event.key === ' ' && !(event.target instanceof HTMLInputElement)) {
    event.preventDefault()
    player.toggle()
  }
}

// 遥控器打开「正在播放」后焦点直接在播放键上，关闭后回到打开它的按钮
const playRef = ref<HTMLButtonElement | null>(null)
useFocusRestore()
onMounted(() => {
  window.addEventListener('keydown', onKey)
  nextTick(() => playRef.value?.focus({ preventScroll: true }))
})
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>
