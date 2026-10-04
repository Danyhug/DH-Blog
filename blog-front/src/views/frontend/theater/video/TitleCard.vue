<template>
  <!-- 卡片内容全压在剧照上，日夜都用深色玻璃与白字：根节点挂 dark -->
  <div
    class="dark group/card relative shrink-0 cursor-pointer transition-transform duration-300 ease-out hover:z-10 hover:scale-[1.06] max-md:hover:scale-100"
    :class="sizeClass"
    @click="emit('open')"
  >
    <VideoThumb
      :image-id="title.imageId"
      :frame-id="frameVideo.id"
      :seed="title.name"
      :label="title.name"
      class="aspect-video w-full rounded-2xl shadow-[0_10px_30px_-8px_rgba(0,0,0,0.7)] ring-1 ring-white/10 transition-shadow duration-300 group-hover/card:shadow-[0_20px_50px_-10px_rgba(0,0,0,0.85)] group-hover/card:ring-white/25"
    >
      <!-- 有图时把片名压在左下角，Netflix 的卡片都带标题字 -->
      <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-transparent"></div>
      <!-- 标题栏是一条贴在卡片底部的毛玻璃，悬停时整块玻璃浮起来 -->
      <div class="absolute inset-x-1.5 bottom-1.5 rounded-xl border border-transparent p-2 transition-all duration-300 group-hover/card:glass">
        <div class="flex items-end justify-between gap-2">
          <div class="min-w-0">
            <p class="m-0 truncate text-sm font-bold text-white drop-shadow">{{ title.name }}</p>
            <p v-if="subtitle" class="m-0 truncate text-[11px] text-white/70">{{ subtitle }}</p>
          </div>
          <div class="flex shrink-0 gap-1.5 opacity-0 transition-opacity group-hover/card:opacity-100 max-md:hidden">
            <button :class="circleButton" class="border-none bg-white/95 text-black shadow-[inset_0_1px_0_white] hover:bg-white" title="播放" @click.stop="emit('play')">
              <PlayIcon class="ml-0.5 size-3.5" />
            </button>
            <button :class="circleButton" class="glass-button text-white" title="详情" @click.stop="emit('open')">
              <ChevronDownIcon class="size-4" />
            </button>
          </div>
        </div>
        <div v-if="progress !== null" class="mt-1.5 h-[3px] w-full overflow-hidden rounded-full bg-white/30">
          <div class="h-full bg-[#e50914]" :style="{ width: `${progress * 100}%` }"></div>
        </div>
      </div>
      <span v-if="title.kind === 'series'" class="glass absolute left-2 top-2 rounded-full px-2 py-0.5 text-[10px] font-semibold tracking-wide text-white">
        剧集 · {{ title.videos.length }} 集
      </span>
      <button
        v-if="removable"
        class="glass-button absolute right-1.5 top-1.5 flex size-7 cursor-pointer items-center justify-center rounded-full text-white opacity-0 transition-opacity group-hover/card:opacity-100 max-md:opacity-100"
        title="从继续观看中移除"
        @click.stop="emit('remove')"
      >
        <CloseIcon class="size-3.5" />
      </button>
    </VideoThumb>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Video } from '@/api/media'
import VideoThumb from './VideoThumb.vue'
import { episodeLabel, type Title } from './catalog'
import { formatRuntime } from '../utils/format'
import { ChevronDownIcon, CloseIcon, PlayIcon } from '../components/icons'

const props = withDefaults(defineProps<{
  title: Title
  /** 继续观看行里具体指向的那一集 */
  video?: Video
  removable?: boolean
  /** 行内卡片固定宽度；网格里由网格决定宽度 */
  inRow?: boolean
}>(), { video: undefined, removable: false, inRow: true })

const emit = defineEmits<{ (e: 'open'): void; (e: 'play'): void; (e: 'remove'): void }>()

const circleButton = 'flex size-7 cursor-pointer items-center justify-center rounded-full transition-transform active:scale-90'
const sizeClass = computed(() => (props.inRow ? 'w-[18.4vw] max-xl:w-[23vw] max-lg:w-[30vw] max-md:w-[42vw]' : 'w-full'))

const frameVideo = computed(() => props.video ?? props.title.videos[0])

const progress = computed(() => {
  const watched = props.video?.progress
  if (!watched || !watched.duration) return null
  return Math.min(1, watched.position / watched.duration)
})

const subtitle = computed(() => {
  if (props.video && props.title.kind === 'series') return [episodeLabel(props.video), props.video.title].filter(Boolean).join(' · ')
  const parts: string[] = []
  if (props.title.year) parts.push(String(props.title.year))
  if (props.title.kind === 'movie') parts.push(formatRuntime(props.title.videos[0].duration))
  return parts.filter(Boolean).join(' · ')
})
</script>
