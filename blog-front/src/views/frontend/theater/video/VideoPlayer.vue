<template>
  <!-- Netflix 风格的全屏播放器。整个容器（而不是 <video>）进入全屏，自绘的控件与字幕才能一起显示。
       画面背后永远是黑的，所以不随日间模式变化：根节点挂 dark，里面的 dark: 变体与玻璃材质一律走夜间样式。 -->
  <div
    ref="rootRef"
    data-nav-scope
    class="dark fixed inset-0 z-[90] select-none bg-black text-white"
    :class="!controlsVisible && 'cursor-none'"
    @mousemove="poke"
    @touchstart.passive="poke"
  >
    <video
      ref="videoRef"
      :key="video.id"
      :src="streamUrl"
      class="absolute inset-0 size-full bg-black object-contain"
      playsinline
      autoplay
      preload="auto"
      @click="togglePlay"
      @dblclick="toggleFullscreen"
      @loadedmetadata="onLoadedMetadata"
      @timeupdate="onTimeUpdate"
      @progress="onProgress"
      @play="playing = true"
      @pause="onPause"
      @waiting="waiting = true"
      @playing="onPlaying"
      @canplay="waiting = false"
      @ended="onEnded"
      @error="onError"
      @volumechange="onVolumeChange"
    ></video>

    <!-- 字幕：自己渲染，位置随控制栏上下浮动，避免被进度条挡住 -->
    <div
      v-if="currentCues.length"
      class="pointer-events-none absolute inset-x-0 flex flex-col items-center gap-1 px-[10%] text-center transition-[bottom] duration-300"
      :class="controlsVisible ? 'bottom-[18%]' : 'bottom-[7%]'"
    >
      <p
        v-for="cue in currentCues"
        :key="cue.start"
        class="m-0 whitespace-pre-line text-[clamp(18px,2.6vw,42px)] font-semibold leading-snug text-white [text-shadow:0_0_6px_rgba(0,0,0,0.9),0_2px_4px_rgba(0,0,0,0.9)]"
      >{{ cue.text }}</p>
    </div>

    <!-- 缓冲 -->
    <div v-if="waiting && !errorMessage" class="pointer-events-none absolute inset-0 flex items-center justify-center">
      <span class="size-16 animate-spin rounded-full border-4 border-white/15 border-t-[#e50914]"></span>
    </div>

    <!-- 暂停时的大标题（Netflix 暂停一会儿后会显示「你正在观看」） -->
    <transition enter-active-class="transition-opacity duration-700" enter-from-class="opacity-0" leave-active-class="transition-opacity duration-200" leave-to-class="opacity-0">
      <div v-if="showPausedInfo" class="pointer-events-none absolute inset-0 flex flex-col justify-center bg-gradient-to-r from-black/80 via-black/40 to-transparent px-[8%]">
        <p class="m-0 text-lg text-white/60">你正在观看</p>
        <h2 class="m-0 mt-2 max-w-[60%] text-[clamp(28px,4vw,60px)] font-black leading-tight">{{ title.name }}</h2>
        <p v-if="episodeText" class="m-0 mt-3 text-xl font-semibold text-white/90">{{ episodeText }}</p>
      </div>
    </transition>

    <!-- 续播提示 -->
    <transition enter-active-class="transition duration-300" enter-from-class="opacity-0 translate-y-2" leave-active-class="transition duration-300" leave-to-class="opacity-0">
      <div v-if="resumeNotice" class="glass absolute bottom-[22%] left-[4%] flex items-center gap-4 rounded-full py-2 pl-5 pr-2 text-sm">
        <span>已从 {{ formatTime(resumeNotice) }} 继续播放</span>
        <button class="cursor-pointer rounded-full border-none bg-white/95 px-4 py-1.5 text-sm font-semibold text-black hover:bg-white" @click="restart">从头播放</button>
      </div>
    </transition>

    <!-- 出错 -->
    <div v-if="errorMessage" class="absolute inset-0 flex items-center justify-center bg-black/50 px-6 text-center backdrop-blur-2xl tv:backdrop-blur-none">
      <div class="glass-thick flex max-w-xl flex-col items-center gap-4 rounded-[28px] px-8 py-8">
      <p class="m-0 text-2xl font-bold">无法播放此视频</p>
      <p class="m-0 max-w-lg text-sm leading-relaxed text-white/65">{{ errorMessage }}</p>
      <div class="mt-2 flex gap-3">
        <a :href="downloadUrl" target="_blank" class="rounded-full bg-white/95 px-5 py-2 text-sm font-semibold text-black! no-underline hover:bg-white">下载到本地播放</a>
        <button class="glass-button cursor-pointer rounded-full px-5 py-2 text-sm font-semibold text-white" @click="close">返回</button>
      </div>
      </div>
    </div>

    <!-- 播完：下一集倒计时 -->
    <div v-if="ended && !errorMessage" class="absolute inset-0 flex items-center justify-center bg-black/40 backdrop-blur-2xl tv:backdrop-blur-none">
      <div v-if="upcoming" class="glass-thick flex w-[min(520px,90vw)] flex-col gap-4 rounded-[28px] p-6">
        <p class="m-0 text-sm text-white/60">{{ countdown > 0 ? `${countdown} 秒后播放下一集` : '即将播放下一集' }}</p>
        <div class="flex gap-4">
          <VideoThumb :frame-id="upcoming.id" :seed="upcoming.name" class="aspect-video w-40 shrink-0 rounded-xl ring-1 ring-white/10" />
          <div class="min-w-0">
            <p class="m-0 font-semibold">{{ episodeLabel(upcoming) }}</p>
            <p class="m-0 mt-1 line-clamp-2 text-sm text-white/70">{{ upcoming.title }}</p>
          </div>
        </div>
        <div class="flex gap-3">
          <button class="flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-full border-none bg-white/95 py-2.5 font-semibold text-black hover:bg-white" @click="playUpcoming"><PlayIcon class="size-4" />立即播放</button>
          <button class="glass-button flex-1 cursor-pointer rounded-full py-2.5 font-semibold text-white" @click="cancelCountdown">取消</button>
        </div>
      </div>
      <div v-else class="glass-thick flex flex-col items-center gap-5 rounded-[28px] px-10 py-8">
        <p class="m-0 text-2xl font-bold">{{ title.name }}</p>
        <div class="flex gap-3">
          <button class="flex cursor-pointer items-center gap-2 rounded-full border-none bg-white/95 px-6 py-2.5 font-semibold text-black hover:bg-white" @click="restart"><RepeatIcon class="size-4" />重新播放</button>
          <button class="glass-button cursor-pointer rounded-full px-6 py-2.5 font-semibold text-white" @click="close">返回影院</button>
        </div>
      </div>
    </div>

    <!-- ===== 控制层 ===== -->
    <transition enter-active-class="transition-opacity duration-200" enter-from-class="opacity-0" leave-active-class="transition-opacity duration-500" leave-to-class="opacity-0">
      <div v-show="controlsVisible && !errorMessage" ref="controlsRef" class="pointer-events-none absolute inset-0">
        <!-- 顶栏 -->
        <div class="pointer-events-auto absolute inset-x-0 top-0 flex items-center gap-4 bg-gradient-to-b from-black/50 to-transparent px-[3%] pb-16 pt-6">
          <button class="glass-button flex size-12 cursor-pointer items-center justify-center rounded-full text-white" title="返回（Esc）" @click="close"><BackIcon class="size-6" /></button>
          <p class="glass m-0 max-w-[60%] truncate rounded-full px-5 py-2.5 text-sm font-semibold max-md:hidden">
            {{ title.name }}<span v-if="episodeText" class="ml-2 font-normal text-white/70">{{ episodeText }}</span>
          </p>
        </div>

        <!-- 底栏 -->
        <div class="pointer-events-auto absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/50 to-transparent px-[3%] pb-6 pt-24 max-md:px-2 max-md:pb-2" @click.stop>
          <div class="glass-thick rounded-[26px] px-6 pb-4 pt-4 max-md:rounded-2xl max-md:px-3">
          <div class="flex items-center gap-4">
            <ProgressBar
              class="flex-1"
              :value="scrubRatio ?? (duration ? currentTime / duration : 0)"
              :buffered="duration ? buffered / duration : 0"
              fill-class="bg-[#e50914]"
              track-class="bg-white/30"
              knob-class="bg-[#e50914] size-4"
              height-class="h-1 group-hover/bar:h-[7px]"
              drag-height-class="h-[7px]"
              hit-class="h-5"
              hover-preview
              label="播放进度"
              :step="duration ? 10 / duration : 0.01"
              @scrub="r => (scrubRatio = r)"
              @seek="commitSeek"
            >
              <template #tooltip="{ ratio }">
                <span class="glass rounded-full px-2.5 py-1 text-xs font-semibold tabular-nums">{{ formatTime(ratio * duration) }}</span>
              </template>
            </ProgressBar>
            <span class="w-20 text-right text-sm tabular-nums text-white/85">{{ formatTime(Math.max(0, duration - currentTime)) }}</span>
          </div>

          <div class="mt-3 flex items-center gap-6 max-md:gap-3">
            <button ref="playButtonRef" :class="iconButton" data-nav-autofocus :title="playing ? '暂停（空格）' : '播放（空格）'" @click="togglePlay">
              <PauseIcon v-if="playing" class="size-9 max-md:size-7" />
              <PlayIcon v-else class="size-9 max-md:size-7" />
            </button>
            <button :class="iconButton" title="后退 10 秒（←）" @click="skip(-10)"><Rewind10Icon class="size-8 max-md:size-6" /></button>
            <button :class="iconButton" title="前进 10 秒（→）" @click="skip(10)"><Forward10Icon class="size-8 max-md:size-6" /></button>
            <div class="group/vol flex items-center gap-2 max-md:hidden">
              <button :class="iconButton" :title="muted ? '取消静音（M）' : '静音（M）'" @click="toggleMute">
                <MuteIcon v-if="muted || volume === 0" class="size-8" />
                <VolumeIcon v-else class="size-8" />
              </button>
              <ProgressBar
                class="w-0 overflow-hidden opacity-0 transition-all duration-200 group-hover/vol:w-24 group-hover/vol:opacity-100"
                :value="muted ? 0 : volume"
                fill-class="bg-white"
                @scrub="setVolume"
                @seek="setVolume"
              />
            </div>

            <!-- 宽屏的片名已经在顶部玻璃胶囊里，这里只占位；窄屏顶栏不显示片名，才在这里露出来 -->
            <p class="m-0 min-w-0 flex-1 truncate text-center text-lg md:invisible max-md:text-sm">
              <span class="font-bold">{{ title.name }}</span>
              <span v-if="episodeText" class="ml-3 text-white/70">{{ episodeText }}</span>
            </p>

            <button v-if="next" :class="iconButton" title="下一集（N）" @click="switchTo(next)"><NextIcon class="size-8 max-md:size-6" /></button>

            <!-- 选集 -->
            <div v-if="title.kind === 'series'" class="relative">
              <button :class="iconButton" title="选集" @click="toggleMenu('episodes')"><EpisodesIcon class="size-8 max-md:size-6" /></button>
              <div v-if="menu === 'episodes'" data-menu-panel :class="[menuPanel, 'w-[min(420px,85vw)]']">
                <p class="m-0 border-b border-white/10 px-4 py-3 text-base font-bold">{{ title.name }}</p>
                <div class="max-h-[50vh] overflow-y-auto">
                  <button
                    v-for="episode in title.videos"
                    :key="episode.id"
                    class="flex w-full cursor-pointer items-center gap-3 rounded-xl border-none bg-transparent px-3 py-2.5 text-left text-sm text-white/80 hover:bg-white/10"
                    :class="episode.id === video.id && 'bg-white/10 text-white'"
                    @click="switchTo(episode)"
                  >
                    <span class="w-14 shrink-0 text-white/50">{{ episodeLabel(episode) || '—' }}</span>
                    <span class="min-w-0 flex-1 truncate">{{ episode.title }}</span>
                    <span v-if="episode.progress?.finished" class="text-xs text-white/40">已看完</span>
                  </button>
                </div>
              </div>
            </div>

            <!-- 字幕 -->
            <div class="relative">
              <button :class="[iconButton, !subtitles.length && 'opacity-40']" :title="subtitles.length ? '字幕（C）' : '没有找到同名字幕文件'" @click="toggleMenu('subtitles')"><SubtitleIcon class="size-8 max-md:size-6" /></button>
              <div v-if="menu === 'subtitles'" data-menu-panel :class="[menuPanel, 'w-60']">
                <p class="m-0 px-4 pb-2 pt-3 text-base font-bold">字幕</p>
                <button :class="menuItem" @click="selectSubtitle(-1)">
                  <CheckIcon class="size-4" :class="subtitleIndex === -1 ? 'opacity-100' : 'opacity-0'" />关闭
                </button>
                <button v-for="(subtitle, i) in subtitles" :key="subtitle.id" :class="menuItem" @click="selectSubtitle(i)">
                  <CheckIcon class="size-4" :class="subtitleIndex === i ? 'opacity-100' : 'opacity-0'" />{{ subtitle.label }}
                  <span class="text-xs uppercase text-white/40">{{ subtitle.format }}</span>
                </button>
                <p v-if="!subtitles.length" class="m-0 px-4 pb-3 text-xs leading-relaxed text-white/50">把与视频同名的 .srt / .vtt 放在同一目录即可自动加载，例如「电影.chs.srt」</p>
              </div>
            </div>

            <!-- 倍速 -->
            <div class="relative max-md:hidden">
              <button :class="iconButton" title="播放速度" @click="toggleMenu('speed')"><SpeedIcon class="size-8" /></button>
              <div v-if="menu === 'speed'" data-menu-panel :class="[menuPanel, 'w-44']">
                <p class="m-0 px-4 pb-2 pt-3 text-base font-bold">播放速度</p>
                <button v-for="speed in speeds" :key="speed" :class="menuItem" @click="setRate(speed)">
                  <CheckIcon class="size-4" :class="rate === speed ? 'opacity-100' : 'opacity-0'" />{{ speed === 1 ? '1x（正常）' : `${speed}x` }}
                </button>
              </div>
            </div>

            <button v-if="pipSupported" :class="[iconButton, 'max-md:hidden']" title="画中画" @click="togglePip"><PipIcon class="size-7" /></button>
            <button :class="iconButton" :title="fullscreen ? '退出全屏（F）' : '全屏（F）'" @click="toggleFullscreen">
              <ExitFullscreenIcon v-if="fullscreen" class="size-8 max-md:size-6" />
              <FullscreenIcon v-else class="size-8 max-md:size-6" />
            </button>
          </div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, onUnmounted, ref, watch } from 'vue'
import { useMusicPlayerStore } from '@/store'
import { mediaStreamUrl, saveProgress, type Video, type VideoProgress } from '@/api/media'
import { getDownloadUrl } from '@/api/file'
import ProgressBar from '../components/ProgressBar.vue'
import VideoThumb from './VideoThumb.vue'
import { episodeLabel, nextEpisode, type Title } from './catalog'
import { formatTime } from '../utils/format'
import { activeCues, parseSubtitles, type Cue } from '../utils/subtitle'
import { rememberDuration } from '../utils/durationProbe'
import { tvMode, useFocusRestore } from '../utils/tv'
import {
  BackIcon, CheckIcon, EpisodesIcon, ExitFullscreenIcon, Forward10Icon, FullscreenIcon, MuteIcon, NextIcon,
  PauseIcon, PipIcon, PlayIcon, RepeatIcon, Rewind10Icon, SpeedIcon, SubtitleIcon, VolumeIcon
} from '../components/icons'

const props = defineProps<{
  video: Video
  title: Title
  /** 指定起播秒数；不传则按观看记录续播 */
  startAt?: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'play', video: Video): void
  (e: 'progress', video: Video, progress: VideoProgress): void
}>()

const iconButton = 'flex cursor-pointer items-center justify-center rounded-full border-none bg-transparent p-1 text-white transition-all hover:scale-110 hover:bg-white/10'
const menuPanel = 'glass-thick absolute bottom-full right-0 mb-7 overflow-hidden rounded-2xl p-1.5 pb-2'
const menuItem = 'flex w-full cursor-pointer items-center gap-3 rounded-xl border-none bg-transparent px-3 py-2 text-left text-sm text-white/85 hover:bg-white/10'
const speeds = [0.5, 0.75, 1, 1.25, 1.5, 2]

const rootRef = ref<HTMLElement | null>(null)
const videoRef = ref<HTMLVideoElement | null>(null)
const controlsRef = ref<HTMLElement | null>(null)
const playButtonRef = ref<HTMLButtonElement | null>(null)
useFocusRestore()

const streamUrl = computed(() => mediaStreamUrl(props.video.id))
const downloadUrl = computed(() => getDownloadUrl(props.video.id))
const next = computed(() => (props.title.kind === 'series' ? nextEpisode(props.title, props.video) : undefined))
const episodeText = computed(() => {
  if (props.title.kind !== 'series') return ''
  return [episodeLabel(props.video), props.video.title].filter(Boolean).join(' ')
})

// ---- 播放状态 ----
const playing = ref(false)
const waiting = ref(true)
const currentTime = ref(0)
const duration = ref(props.video.duration || 0)
const buffered = ref(0)
const scrubRatio = ref<number | null>(null)
const rate = ref(1)
const errorMessage = ref('')
const ended = ref(false)
const resumeNotice = ref(0)

const volumeKey = 'dh-blog:video-volume'
const volume = ref(readVolume())
const muted = ref(false)

function readVolume() {
  try {
    const raw = localStorage.getItem(volumeKey)
    const value = Number(raw)
    return raw !== null && value >= 0 && value <= 1 ? value : 1
  } catch {
    return 1
  }
}

function el() {
  return videoRef.value!
}

function togglePlay() {
  if (errorMessage.value) return
  const video = el()
  if (video.paused) video.play().catch(() => {})
  else video.pause()
}

function skip(seconds: number) {
  const video = el()
  video.currentTime = Math.max(0, Math.min(video.currentTime + seconds, duration.value || video.currentTime + seconds))
  poke()
}

function commitSeek(ratio: number) {
  scrubRatio.value = null
  if (duration.value) el().currentTime = ratio * duration.value
  ended.value = false
}

function setVolume(value: number) {
  const video = el()
  video.volume = Math.max(0, Math.min(1, value))
  video.muted = video.volume === 0
}

function toggleMute() {
  const video = el()
  video.muted = !video.muted
  if (!video.muted && video.volume === 0) video.volume = 0.5
}

function onVolumeChange() {
  const video = el()
  volume.value = video.volume
  muted.value = video.muted
  try {
    localStorage.setItem(volumeKey, String(video.volume))
  } catch {
    // 记不住音量也不影响播放
  }
}

function setRate(value: number) {
  rate.value = value
  el().playbackRate = value
  menu.value = null
}

function restart() {
  resumeNotice.value = 0
  ended.value = false
  el().currentTime = 0
  el().play().catch(() => {})
}

// ---- 媒体事件 ----
let started = false

function onLoadedMetadata() {
  const video = el()
  video.volume = volume.value
  video.playbackRate = rate.value
  if (Number.isFinite(video.duration)) {
    duration.value = video.duration
    if (!props.video.duration) props.video.duration = video.duration
    rememberDuration(props.video.id, video.duration)
  }
  const progress = props.video.progress
  if (props.startAt !== undefined) {
    video.currentTime = props.startAt
  } else if (progress && !progress.finished && progress.position > 10 && progress.position < duration.value - 10) {
    video.currentTime = progress.position
    resumeNotice.value = progress.position
    setTimeout(() => (resumeNotice.value = 0), 6000)
  }
}

// 关闭播放器时 <video> 被移出文档会补发一次 pause / timeupdate，那时 ref 已经清空
function onTimeUpdate() {
  if (videoRef.value) currentTime.value = videoRef.value.currentTime
}

function onProgress() {
  const ranges = videoRef.value?.buffered
  buffered.value = ranges?.length ? ranges.end(ranges.length - 1) : 0
}

function onPlaying() {
  waiting.value = false
  playing.value = true
  ended.value = false
  if (!started) {
    started = true
    persist(true)
  }
}

function onPause() {
  playing.value = false
  persist()
}

function onError() {
  waiting.value = false
  const code = el().error?.code
  errorMessage.value = code === MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED || code === MediaError.MEDIA_ERR_DECODE
    ? `浏览器无法解码「${props.video.name}」。MKV、AVI、WMV 或 HEVC/H.265 编码的视频在部分浏览器里不受支持，可以下载后用本地播放器观看，或转码为 H.264 的 MP4。`
    : '视频加载失败，请检查网络或文件是否仍在网盘中。'
}

// ---- 观看进度 ----
const SAVE_INTERVAL = 10_000
let saveTimer: ReturnType<typeof setInterval> | null = null

function persist(startedNow = false) {
  const video = videoRef.value
  if (!video || !duration.value || errorMessage.value) return
  const position = video.ended ? duration.value : video.currentTime
  const progress: VideoProgress = {
    position,
    duration: duration.value,
    finished: position >= duration.value * 0.95,
    updated_at: nowString()
  }
  emit('progress', props.video, progress)
  saveProgress(props.video.id, position, duration.value, startedNow).catch(() => {})
}

function nowString() {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

// ---- 播完与下一集 ----
const countdown = ref(0)
const upcoming = ref<Video | undefined>()
let countdownTimer: ReturnType<typeof setInterval> | null = null

function onEnded() {
  playing.value = false
  persist()
  ended.value = true
  upcoming.value = next.value
  if (!upcoming.value) return
  countdown.value = 8
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) playUpcoming()
  }, 1000)
}

function playUpcoming() {
  cancelCountdown()
  if (upcoming.value) switchTo(upcoming.value)
}

function cancelCountdown() {
  if (countdownTimer) clearInterval(countdownTimer)
  countdownTimer = null
  countdown.value = 0
}

// ---- 字幕 ----
const subtitles = computed(() => props.video.subtitles)
const subtitleIndex = ref(-1)
const cues = ref<Cue[]>([])
const currentCues = computed(() => (cues.value.length ? activeCues(cues.value, currentTime.value) : []))
const subtitleCache = new Map<string, Cue[]>()

async function loadSubtitleText(fileId: string): Promise<string> {
  const response = await fetch(mediaStreamUrl(fileId))
  const buffer = await response.arrayBuffer()
  // 中文字幕常见 GBK 编码，UTF-8 严格解码失败就换 GB18030
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(buffer)
  } catch {
    return new TextDecoder('gb18030').decode(buffer)
  }
}

async function selectSubtitle(index: number) {
  subtitleIndex.value = index
  menu.value = null
  cues.value = []
  const subtitle = subtitles.value[index]
  if (!subtitle) return
  const cached = subtitleCache.get(subtitle.id)
  if (cached) {
    cues.value = cached
    return
  }
  try {
    const parsed = parseSubtitles(await loadSubtitleText(subtitle.id))
    subtitleCache.set(subtitle.id, parsed)
    if (subtitleIndex.value === index) cues.value = parsed
  } catch {
    cues.value = []
  }
}

function defaultSubtitle(): number {
  if (!subtitles.value.length) return -1
  const chinese = subtitles.value.findIndex(item => /中|简|繁|双语/.test(item.label) || /^(zh|chs|cht|sc|tc)/.test(item.lang))
  return chinese >= 0 ? chinese : 0
}

// ---- 控制层显隐 ----
const controlsVisible = ref(true)
const menu = ref<'subtitles' | 'speed' | 'episodes' | null>(null)
let hideTimer: ReturnType<typeof setTimeout> | null = null
const pausedFor = ref(0)
let pausedTimer: ReturnType<typeof setInterval> | null = null
const showPausedInfo = computed(() => !playing.value && !ended.value && !errorMessage.value && pausedFor.value >= 6 && !controlsVisible.value)

function poke() {
  controlsVisible.value = true
  pausedFor.value = 0
  if (hideTimer) clearTimeout(hideTimer)
  hideTimer = setTimeout(() => {
    if (!menu.value) controlsVisible.value = false
  }, 3000)
}

function toggleMenu(name: 'subtitles' | 'speed' | 'episodes') {
  menu.value = menu.value === name ? null : name
  poke()
  // 用遥控器打开的菜单，焦点直接进到第一项（当前选中项优先），不必再按方向键找
  if (menu.value && controlFocused()) {
    nextTick(() => {
      const panel = rootRef.value?.querySelector('[data-menu-panel]')
      const items = [...(panel?.querySelectorAll<HTMLElement>('button') ?? [])]
      ;(items.find(item => item.classList.contains('bg-white/10') || item.querySelector('.opacity-100')) ?? items[0])?.focus()
    })
  }
}

// ---- Media Session ----
// 遥控器上的播放/暂停、快进快退键在 Android 上走系统的媒体会话，而不是 keydown；
// 把它们接到当前视频上，关闭时交还给音乐播放器。
const musicPlayer = useMusicPlayerStore()

function updateMediaSession() {
  if (!('mediaSession' in navigator)) return
  const session = navigator.mediaSession
  session.metadata = new MediaMetadata({ title: props.title.name, artist: episodeText.value || props.video.folder_path || '' })
  const handlers: [MediaSessionAction, MediaSessionActionHandler | null][] = [
    ['play', () => { lastMediaAction = Date.now(); el().play().catch(() => {}) }],
    ['pause', () => { lastMediaAction = Date.now(); el().pause() }],
    ['seekbackward', () => skip(-10)],
    ['seekforward', () => skip(10)],
    ['seekto', details => { if (details.seekTime !== undefined) el().currentTime = details.seekTime }],
    ['nexttrack', next.value ? () => switchTo(next.value!) : null],
    ['previoustrack', null],
  ]
  for (const [action, handler] of handlers) {
    try {
      session.setActionHandler(action, handler)
    } catch {
      // 老版本浏览器不认识的动作直接跳过
    }
  }
}

function releaseMediaSession() {
  if (!('mediaSession' in navigator)) return
  for (const action of ['play', 'pause', 'seekbackward', 'seekforward', 'seekto', 'nexttrack', 'previoustrack'] as MediaSessionAction[]) {
    try {
      navigator.mediaSession.setActionHandler(action, null)
    } catch {
      // 同上
    }
  }
  navigator.mediaSession.metadata = null
  musicPlayer.updateMediaSession()
}

// ---- 全屏 / 画中画 ----
const fullscreen = ref(false)
const pipSupported = typeof document !== 'undefined' && 'pictureInPictureEnabled' in document && document.pictureInPictureEnabled

function toggleFullscreen() {
  if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
  else rootRef.value?.requestFullscreen().catch(() => {})
}

function onFullscreenChange() {
  fullscreen.value = !!document.fullscreenElement
}

async function togglePip() {
  try {
    if (document.pictureInPictureElement) await document.exitPictureInPicture()
    else await el().requestPictureInPicture()
  } catch {
    // 画中画被浏览器拒绝时静默忽略
  }
}

// ---- 键盘与遥控器 ----
// 焦点在某个控件上（键盘/遥控器移过去的，即 :focus-visible）时，方向键交给影院的空间导航在控件间移动、
// 确认键和空格交给按钮自己；否则方向键直接快退快进。鼠标点过的按钮虽有焦点但不是 focus-visible，不受影响。
function controlFocused() {
  const active = document.activeElement
  return !!active && controlsVisible.value && !!controlsRef.value?.contains(active) && active.matches(':focus-visible')
}

// 遥控器的上下键没有音量可调（电视有自己的音量键），改为唤出控制栏并聚焦播放键
function focusControls() {
  poke()
  nextTick(() => playButtonRef.value?.focus())
}

// 同一次按键可能既作为 keydown 又作为 Media Session 动作送达，切换类操作只认先到的那个
let lastMediaAction = 0
function mediaToggle() {
  if (Date.now() - lastMediaAction < 400) return
  lastMediaAction = Date.now()
  togglePlay()
}

function onKey(event: KeyboardEvent) {
  if (event.defaultPrevented) return
  if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return
  poke()
  if (controlFocused() && ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Enter', ' '].includes(event.key)) return
  switch (event.key) {
    case ' ':
    case 'k':
    case 'Enter':
      togglePlay()
      break
    case 'ArrowLeft':
    case 'j':
    case 'MediaRewind':
      skip(-10)
      break
    case 'ArrowRight':
    case 'l':
    case 'MediaFastForward':
      skip(10)
      break
    case 'ArrowUp':
      if (tvMode.value) focusControls()
      else setVolume(el().volume + 0.1)
      break
    case 'ArrowDown':
      if (tvMode.value) focusControls()
      else setVolume(el().volume - 0.1)
      break
    case 'MediaPlayPause':
      mediaToggle()
      break
    case 'MediaPlay':
      el().play().catch(() => {})
      break
    case 'MediaPause':
      el().pause()
      break
    case 'MediaTrackNext':
      if (next.value) switchTo(next.value)
      break
    case 'MediaStop':
      close()
      break
    case 'f':
      toggleFullscreen()
      break
    case 'm':
      toggleMute()
      break
    case 'c':
      selectSubtitle(subtitleIndex.value === -1 ? defaultSubtitle() : -1)
      break
    case 'n':
      if (next.value) switchTo(next.value)
      break
    case 'Escape':
      if (menu.value) menu.value = null
      else if (!document.fullscreenElement) close()
      break
    default:
      return
  }
  event.preventDefault()
  poke()
}

// 换集前先记下这一集的进度：换集后 props.video 已是新的一集
function switchTo(video: Video) {
  persist()
  menu.value = null
  emit('play', video)
}

// 进度在卸载前保存一次（关闭、切路由都会走到这里）
function close() {
  if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
  emit('close')
}

// 切换到另一集时（同一个组件实例）重置一切
watch(() => props.video.id, () => {
  started = false
  ended.value = false
  errorMessage.value = ''
  waiting.value = true
  currentTime.value = 0
  buffered.value = 0
  duration.value = props.video.duration || 0
  resumeNotice.value = 0
  cancelCountdown()
  upcoming.value = undefined
  selectSubtitle(defaultSubtitle())
  updateMediaSession()
})

onMounted(() => {
  poke()
  selectSubtitle(defaultSubtitle())
  // 挂在 document 而不是 window：先于影院的空间导航与详情页的 Esc 处理（都在 window 上），
  // 播放器处理过的按键 preventDefault 后它们就不再响应
  document.addEventListener('keydown', onKey)
  updateMediaSession()
  // 电视上直接全屏，藏掉浏览器的地址栏；打开播放器的那次按键/点击就是全屏需要的用户手势
  if (tvMode.value) rootRef.value?.requestFullscreen().catch(() => {})
  document.addEventListener('fullscreenchange', onFullscreenChange)
  saveTimer = setInterval(() => {
    if (playing.value) persist()
  }, SAVE_INTERVAL)
  pausedTimer = setInterval(() => {
    if (!playing.value) pausedFor.value++
  }, 1000)
})

onBeforeUnmount(persist)

onUnmounted(() => {
  document.removeEventListener('keydown', onKey)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  releaseMediaSession()
  if (saveTimer) clearInterval(saveTimer)
  if (pausedTimer) clearInterval(pausedTimer)
  if (hideTimer) clearTimeout(hideTimer)
  cancelCountdown()
})
</script>
