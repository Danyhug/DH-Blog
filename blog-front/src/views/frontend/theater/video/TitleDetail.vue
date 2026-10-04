<template>
  <!-- 作品详情（Netflix 的「更多信息」弹层）：大剧照 + 播放按钮 + 选集列表 -->
  <!-- data-nav-scope：打开时遥控器的焦点只在详情里移动，打开即落在播放键上 -->
  <div data-nav-scope class="fixed inset-0 z-[80] overflow-y-auto bg-black/25 px-4 dark:bg-black/45 py-8 backdrop-blur-xl tv:backdrop-blur-none" @click.self="emit('close')">
    <div class="glass-thick relative mx-auto w-full max-w-[880px] overflow-hidden rounded-[30px] animate-dialog-appear">
      <button class="glass-button absolute right-4 top-4 z-10 flex size-10 cursor-pointer items-center justify-center rounded-full text-zinc-900 dark:text-white" title="关闭" @click="emit('close')">
        <CloseIcon class="size-5" />
      </button>

      <!-- 剧照上的标题与按钮日夜都是白字深玻璃：这一块挂 dark -->
      <VideoThumb :image-id="title.imageId" :frame-id="target.id" :seed="title.name" class="dark aspect-video w-full">
        <div class="absolute inset-0 bg-gradient-to-t from-[rgb(20_20_26/0.95)] via-[rgb(20_20_26/0.25)] to-transparent"></div>
        <div class="absolute inset-x-0 bottom-0 px-12 pb-10 max-md:px-5 max-md:pb-5">
          <h2 class="m-0 mb-5 max-w-[80%] text-[42px] font-black leading-none tracking-tight text-zinc-900 dark:text-white drop-shadow-xl max-md:mb-3 max-md:text-2xl">{{ title.name }}</h2>
          <div class="flex items-center gap-3">
            <button ref="playRef" :class="playButton" data-nav-autofocus @click="emit('play', target)">
              <PlayIcon class="size-6" />
              {{ resumeLabel }}
            </button>
            <button v-if="target.progress && !target.progress.finished" :class="ghostButton" title="从头播放" @click="emit('play', target, true)">
              <RepeatIcon class="size-5" />从头播放
            </button>
          </div>
          <div v-if="targetProgress !== null" class="mt-4 flex max-w-[360px] items-center gap-3 text-xs text-black/70 dark:text-white/70">
            <div class="h-1 flex-1 overflow-hidden rounded-full bg-black/15 dark:bg-white/25"><div class="h-full bg-[#e50914]" :style="{ width: `${targetProgress * 100}%` }"></div></div>
            <span>剩余 {{ formatRuntime(remaining) || '不到 1 分钟' }}</span>
          </div>
        </div>
      </VideoThumb>

      <div class="grid grid-cols-[2fr_1fr] gap-8 px-12 pb-8 pt-2 max-md:grid-cols-1 max-md:gap-4 max-md:px-5">
        <div>
          <div class="flex flex-wrap items-center gap-2 text-sm text-zinc-500 dark:text-[#bcbcbc]">
            <span v-if="title.year" class="text-green-600 dark:text-[#46d369] font-semibold">{{ title.year }}</span>
            <span v-if="title.kind === 'series'">{{ seasons.length > 1 ? `${seasons.length} 季` : `${title.videos.length} 集` }}</span>
            <span v-else-if="target.duration">{{ formatRuntime(target.duration) }}</span>
            <span v-if="qualityBadge" class="rounded-[3px] border border-black/25 px-1 dark:border-white/40 text-[10px] font-semibold leading-4">{{ qualityBadge }}</span>
            <span v-if="target.subtitles.length" class="rounded-[3px] border border-black/25 px-1 dark:border-white/40 text-[10px] font-semibold leading-4">字幕</span>
          </div>
          <p v-if="title.kind === 'series' && target.episode" class="m-0 mt-4 text-lg font-semibold text-zinc-900 dark:text-white">
            {{ episodeLabel(target) }}「{{ target.title }}」
          </p>
          <p class="m-0 mt-3 text-sm leading-relaxed text-black/80 dark:text-white/80">{{ target.name }}</p>
        </div>
        <div class="flex flex-col gap-2 text-sm text-[#777]">
          <p class="m-0"><span>位置：</span><span class="text-zinc-700 dark:text-[#ddd]">{{ target.folder_path || '我的网盘' }}</span></p>
          <p class="m-0"><span>大小：</span><span class="text-zinc-700 dark:text-[#ddd]">{{ formatSize(target.size) }}</span></p>
          <p v-if="target.subtitles.length" class="m-0"><span>字幕：</span><span class="text-zinc-700 dark:text-[#ddd]">{{ target.subtitles.map(s => s.label).join('、') }}</span></p>
          <button v-if="hasProgress" class="mt-1 w-fit cursor-pointer border-none bg-transparent p-0 text-left text-zinc-500 dark:text-[#bcbcbc] underline-offset-2 hover:text-zinc-900 dark:hover:text-white hover:underline" @click="emit('remove-progress', title)">
            清除观看记录
          </button>
        </div>
      </div>

      <!-- 选集 -->
      <section v-if="title.kind === 'series'" class="px-12 pb-12 max-md:px-5">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="m-0 text-2xl font-bold text-zinc-900 dark:text-white">选集</h3>
          <el-select v-if="seasons.length > 1" v-model="season" size="large" class="w-36" popper-class="glass-popup">
            <el-option v-for="s in seasons" :key="s" :label="`第 ${s} 季`" :value="s" />
          </el-select>
        </div>
        <button
          v-for="(video, i) in episodes"
          :key="video.id"
          class="group mb-1 flex w-full cursor-pointer items-center gap-5 rounded-2xl border border-transparent bg-transparent px-4 py-4 text-left transition-all hover:bg-black/5 dark:hover:bg-white/[0.07] max-md:gap-3 max-md:px-2"
          :class="video.id === target.id && 'glass'"
          @click="emit('play', video)"
        >
          <span class="w-6 shrink-0 text-center text-2xl text-zinc-600 dark:text-[#d2d2d2] max-md:hidden">{{ video.episode || i + 1 }}</span>
          <VideoThumb :image-id="''" :frame-id="video.id" :seed="video.name" class="aspect-video w-[140px] shrink-0 rounded-xl ring-1 ring-black/5 dark:ring-white/10 max-md:w-[110px]">
            <div class="absolute inset-0 flex items-center justify-center bg-black/0 opacity-0 transition group-hover:bg-black/40 group-hover:opacity-100">
              <span class="glass-button flex size-10 items-center justify-center rounded-full"><PlayIcon class="ml-0.5 size-4 text-zinc-900 dark:text-white" /></span>
            </div>
            <div v-if="video.progress && video.progress.duration" class="absolute inset-x-0 bottom-0 h-1 bg-white/30">
              <div class="h-full bg-[#e50914]" :style="{ width: `${Math.min(1, video.progress.position / video.progress.duration) * 100}%` }"></div>
            </div>
          </VideoThumb>
          <span class="min-w-0 flex-1">
            <span class="flex items-baseline justify-between gap-3">
              <span class="truncate text-base font-semibold text-zinc-900 dark:text-white">{{ video.title }}</span>
              <span class="shrink-0 text-sm text-zinc-600 dark:text-[#d2d2d2]">{{ formatRuntime(video.duration) }}</span>
            </span>
            <span class="mt-1 line-clamp-2 block text-sm text-zinc-500 dark:text-[#a3a3a3]">{{ video.name }}</span>
          </span>
        </button>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import type { Video } from '@/api/media'
import VideoThumb from './VideoThumb.vue'
import { episodeLabel, resumeTarget, seasonsOf, type Title } from './catalog'
import { formatRuntime } from '../utils/format'
import { CloseIcon, PlayIcon, RepeatIcon } from '../components/icons'
import { focusElement, useFocusRestore } from '../utils/tv'

const props = defineProps<{ title: Title }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'play', video: Video, fromStart?: boolean): void
  (e: 'remove-progress', title: Title): void
}>()

const playButton = 'flex cursor-pointer items-center gap-2 rounded-full border-none bg-white/95 px-7 py-2.5 text-base font-bold text-black shadow-[0_8px_30px_rgba(255,255,255,0.25),inset_0_1px_0_white] transition-all hover:bg-white active:scale-95 max-md:px-5'
const ghostButton = 'glass-button flex cursor-pointer items-center gap-2 rounded-full px-5 py-2.5 text-base font-semibold text-zinc-900 dark:text-white'

const target = computed(() => resumeTarget(props.title))
const seasons = computed(() => seasonsOf(props.title))
const season = ref(target.value.season || seasons.value[0] || 1)
const episodes = computed(() => props.title.videos.filter(video => (video.season || 1) === season.value))
const hasProgress = computed(() => props.title.videos.some(video => video.progress))

const targetProgress = computed(() => {
  const watched = target.value.progress
  if (!watched || watched.finished || !watched.duration) return null
  return Math.min(1, watched.position / watched.duration)
})
const remaining = computed(() => {
  const watched = target.value.progress
  return watched ? Math.max(0, watched.duration - watched.position) : 0
})

const resumeLabel = computed(() => {
  const watched = target.value.progress
  const label = props.title.kind === 'series' ? ` ${episodeLabel(target.value)}` : ''
  if (watched && !watched.finished && watched.position > 5) return `继续播放${label}`
  return `播放${label}`
})

const qualityBadge = computed(() => {
  const name = target.value.name.toLowerCase()
  if (/2160p|4k|uhd/.test(name)) return '4K'
  if (/1080[pi]/.test(name)) return 'HD'
  return ''
})

function formatSize(size: number) {
  if (size >= 1024 ** 3) return `${(size / 1024 ** 3).toFixed(2)} GB`
  return `${(size / 1024 ** 2).toFixed(1)} MB`
}

function onKey(event: KeyboardEvent) {
  // 播放器叠在详情上面时 Esc 归播放器（它先处理并 preventDefault）
  if (event.key === 'Escape' && !event.defaultPrevented) emit('close')
}
const playRef = ref<HTMLButtonElement | null>(null)
useFocusRestore()
onMounted(() => {
  window.addEventListener('keydown', onKey)
  nextTick(() => focusElement(playRef.value!))
})
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>
