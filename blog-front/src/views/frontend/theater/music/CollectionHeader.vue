<template>
  <!-- 专辑 / 歌单 / 文件夹 / 艺人详情页的头部：大封面 + 标题 + 播放与随机播放 -->
  <header class="flex items-end gap-8 max-md:flex-col max-md:items-center max-md:gap-5 max-md:text-center">
    <div class="w-[min(270px,60vw)] shrink-0">
      <!-- 歌单用前四首的封面拼成 2×2 马赛克，和 Apple Music 一致 -->
      <div v-if="covers.length >= 4" class="grid aspect-square grid-cols-2 overflow-hidden rounded-[22px] shadow-[0_24px_60px_rgba(0,0,0,0.55)] ring-1 ring-black/10 dark:ring-white/15">
        <Artwork v-for="(src, i) in covers.slice(0, 4)" :key="i" :src="src" :seed="`${title}-${i}`" class="aspect-square" />
      </div>
      <Artwork
        v-else
        :src="covers[0] || ''"
        :seed="title"
        :icon="icon"
        class="aspect-square w-full shadow-[0_24px_60px_rgba(0,0,0,0.55)] ring-1 ring-black/10 dark:ring-white/15"
        :class="round ? 'rounded-full' : 'rounded-[22px]'"
      />
    </div>
    <div class="min-w-0 flex-1 pb-1">
      <p class="m-0 text-xs font-semibold uppercase tracking-wider text-black/50 dark:text-white/50">{{ kind }}</p>
      <h1 class="m-0 mt-1 text-[34px] font-bold leading-tight text-zinc-900 dark:text-white max-md:text-2xl">{{ title }}</h1>
      <p v-if="subtitle" class="m-0 mt-1 text-xl text-[#fa2d48] max-md:text-lg">
        <button v-if="subtitleClickable" class="cursor-pointer border-none bg-transparent p-0 text-inherit hover:underline" @click="emit('subtitle')">{{ subtitle }}</button>
        <template v-else>{{ subtitle }}</template>
      </p>
      <p v-if="meta" class="m-0 mt-2 text-[13px] text-black/50 dark:text-white/50">{{ meta }}</p>
      <div class="mt-5 flex flex-wrap items-center gap-3 max-md:justify-center">
        <button :class="[primaryButton]" :disabled="!playable" @click="emit('play')">
          <PlayIcon class="size-4" /> 播放
        </button>
        <button :class="[secondaryButton]" :disabled="!playable" @click="emit('shuffle')">
          <ShuffleIcon class="size-4" /> 随机播放
        </button>
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import Artwork from '../components/Artwork.vue'
import { MusicNoteIcon, PlayIcon, ShuffleIcon } from '../components/icons'

withDefaults(defineProps<{
  kind: string
  title: string
  subtitle?: string
  subtitleClickable?: boolean
  meta?: string
  covers?: string[]
  round?: boolean
  icon?: Component
  playable?: boolean
}>(), { subtitle: '', subtitleClickable: false, meta: '', covers: () => [], round: false, icon: () => MusicNoteIcon, playable: true })

const emit = defineEmits<{ (e: 'play'): void; (e: 'shuffle'): void; (e: 'subtitle'): void }>()

const primaryButton = 'flex min-w-[120px] cursor-pointer items-center justify-center gap-2 rounded-full border border-white/25 bg-[#fa2d48] px-5 py-2 text-sm font-semibold text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.4),0_8px_24px_-6px_rgba(250,45,72,0.7)] transition-all hover:bg-[#ff4f67] active:scale-95 disabled:cursor-not-allowed disabled:opacity-40'
const secondaryButton = 'glass-button flex min-w-[120px] cursor-pointer items-center justify-center gap-2 rounded-full px-5 py-2 text-sm font-semibold text-[#fa2d48] disabled:cursor-not-allowed disabled:opacity-40'
</script>
