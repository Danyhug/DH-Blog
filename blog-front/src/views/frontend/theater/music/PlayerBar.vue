<template>
  <!-- 底部迷你播放条（Apple Music 的 mini player）：悬浮的液态玻璃胶囊。点封面或标题展开全屏「正在播放」。 -->
  <div class="glass-thick fixed inset-x-3 bottom-3 z-[60] mx-auto max-w-[1400px] overflow-hidden rounded-[24px] max-md:inset-x-2 max-md:bottom-2 max-md:rounded-[20px]">
    <!-- 窄屏只留一条细进度线，完整的拖动条放在全屏播放页 -->
    <div class="absolute inset-x-4 bottom-0 h-0.5 overflow-hidden rounded-full bg-black/5 dark:bg-white/10 md:hidden">
      <div class="h-full bg-[#fa2d48]" :style="{ width: `${ratio * 100}%` }"></div>
    </div>

    <div class="flex h-[76px] items-center gap-4 px-3 max-md:h-[68px] max-md:gap-3 max-md:px-2.5">
      <!-- 当前曲目 -->
      <button class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 border-none bg-transparent p-0 text-left md:max-w-[30%]" @click="expand('lyrics')">
        <Artwork :src="cover" :seed="track.album" class="size-[52px] shrink-0 rounded-[14px] shadow-[0_6px_20px_rgba(0,0,0,0.5)] ring-1 ring-black/10 dark:ring-white/15 max-md:size-12" />
        <span class="min-w-0">
          <span class="block truncate text-sm font-medium text-zinc-900 dark:text-white">{{ track.title }}</span>
          <span class="block truncate text-xs text-black/55 dark:text-white/55">{{ track.artist }} — {{ track.album }}</span>
        </span>
      </button>

      <!-- 播放控制 + 进度 -->
      <div class="flex flex-col items-center gap-1 md:flex-[1.4]">
        <div class="flex items-center gap-5 max-md:gap-3">
          <button :class="[iconButton, 'max-md:hidden', player.shuffle ? accent : 'text-black/60 dark:text-white/60']" title="随机播放" @click="player.toggleShuffle()">
            <ShuffleIcon class="size-4" />
          </button>
          <button :class="[iconButton, 'text-black/90 dark:text-white/90 max-md:hidden']" title="上一首" @click="player.prev()">
            <PrevIcon class="size-5" />
          </button>
          <button :class="[iconButton, 'glass-button size-11 rounded-full text-zinc-900 dark:text-white']" :title="player.playing ? '暂停' : '播放'" @click="player.toggle()">
            <span v-if="player.buffering && player.playing" class="block size-6 animate-spin rounded-full border-2 border-black/20 dark:border-white/30 border-t-zinc-800 dark:border-t-white"></span>
            <PauseIcon v-else-if="player.playing" class="size-5" />
            <PlayIcon v-else class="ml-0.5 size-5" />
          </button>
          <button :class="[iconButton, 'text-black/90 dark:text-white/90']" title="下一首" @click="player.next()">
            <NextIcon class="size-5" />
          </button>
          <button :class="[iconButton, 'max-md:hidden', player.repeat !== 'off' ? accent : 'text-black/60 dark:text-white/60']" :title="repeatTitle" @click="player.cycleRepeat()">
            <RepeatOneIcon v-if="player.repeat === 'one'" class="size-4" />
            <RepeatIcon v-else class="size-4" />
          </button>
        </div>
        <div class="flex w-full max-w-[560px] items-center gap-2 text-[11px] tabular-nums text-black/50 dark:text-white/50 max-md:hidden">
          <span class="w-10 text-right">{{ formatTime(scrubTime ?? player.currentTime) }}</span>
          <ProgressBar
            class="flex-1"
            :value="scrubRatio ?? ratio"
            :buffered="player.duration ? player.buffered / player.duration : 0"
            fill-class="bg-zinc-700 dark:bg-white/80"
            height-class="h-1 group-hover/bar:h-1.5"
            @scrub="r => (scrubRatio = r)"
            @seek="commitSeek"
          />
          <span class="w-10">{{ formatTime(player.duration) }}</span>
        </div>
      </div>

      <!-- 歌词 / 队列 / 音量 -->
      <div class="flex items-center justify-end gap-4 md:flex-1 max-md:hidden">
        <button :class="[iconButton, 'text-black/60 dark:text-white/60']" title="歌词" @click="expand('lyrics')">
          <LyricsIcon class="size-[18px]" />
        </button>
        <button :class="[iconButton, 'text-black/60 dark:text-white/60']" title="待播清单" @click="expand('queue')">
          <QueueIcon class="size-[18px]" />
        </button>
        <div class="flex w-32 items-center gap-2">
          <button :class="[iconButton, 'text-black/60 dark:text-white/60']" :title="player.muted ? '取消静音' : '静音'" @click="player.toggleMute()">
            <MuteIcon v-if="player.muted || player.volume === 0" class="size-4" />
            <VolumeIcon v-else class="size-4" />
          </button>
          <ProgressBar class="flex-1" :value="player.muted ? 0 : player.volume" fill-class="bg-zinc-700 dark:bg-white/80" @scrub="player.setVolume" @seek="player.setVolume" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMusicPlayerStore } from '@/store'
import { trackCoverUrl } from '@/api/media'
import Artwork from '../components/Artwork.vue'
import ProgressBar from '../components/ProgressBar.vue'
import { formatTime } from '../utils/format'
import {
  LyricsIcon, MuteIcon, NextIcon, PauseIcon, PlayIcon, PrevIcon, QueueIcon,
  RepeatIcon, RepeatOneIcon, ShuffleIcon, VolumeIcon
} from '../components/icons'

const player = useMusicPlayerStore()
const track = computed(() => player.current!)
const cover = computed(() => trackCoverUrl(track.value))
const ratio = computed(() => (player.duration ? player.currentTime / player.duration : 0))

const scrubRatio = ref<number | null>(null)
const scrubTime = computed(() => (scrubRatio.value === null ? null : scrubRatio.value * player.duration))

const iconButton = 'flex cursor-pointer items-center justify-center border-none bg-transparent p-0 transition-colors hover:text-zinc-900 dark:hover:text-white'
const accent = 'text-[#fa2d48] hover:text-[#ff4f67]'
const repeatTitle = computed(() => ({ off: '循环：关', all: '循环：全部', one: '循环：单曲' })[player.repeat])

function commitSeek(r: number) {
  scrubRatio.value = null
  player.seek(r * player.duration)
}

function expand(panel: 'lyrics' | 'queue') {
  player.panel = panel
  player.expanded = true
}
</script>
