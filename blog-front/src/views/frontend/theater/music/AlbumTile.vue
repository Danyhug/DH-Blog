<template>
  <div class="group min-w-0 cursor-pointer" @click="emit('open')">
    <div class="relative">
      <Artwork
        :src="cover"
        :seed="seed"
        :icon="icon"
        class="aspect-square w-full shadow-[0_12px_30px_-8px_rgba(0,0,0,0.6)] ring-1 ring-black/5 dark:ring-white/10 transition-all duration-300 group-hover:-translate-y-1 group-hover:shadow-[0_20px_40px_-10px_rgba(0,0,0,0.7)]"
        :class="round ? 'rounded-full' : 'rounded-[18px]'"
      />
      <div class="pointer-events-none absolute inset-0 bg-black/0 transition-all duration-300 group-hover:-translate-y-1 group-hover:bg-black/20" :class="round ? 'rounded-full' : 'rounded-[18px]'"></div>
      <button
        v-if="!round"
        class="glass-button absolute bottom-3 left-3 flex size-10 translate-y-1 cursor-pointer items-center justify-center rounded-full text-zinc-900 dark:text-white opacity-0 transition-all duration-200 group-hover:-translate-y-0.5 group-hover:opacity-100"
        title="播放"
        @click.stop="emit('play')"
      >
        <PlayIcon class="ml-0.5 size-4" />
      </button>
    </div>
    <p class="m-0 mt-2.5 truncate text-[13px] font-medium text-zinc-900 dark:text-white" :class="round && 'text-center'">{{ title }}</p>
    <p v-if="subtitle" class="m-0 truncate text-[13px] text-black/50 dark:text-white/50" :class="round && 'text-center'">{{ subtitle }}</p>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import Artwork from '../components/Artwork.vue'
import { MusicNoteIcon, PlayIcon } from '../components/icons'

withDefaults(defineProps<{
  title: string
  subtitle?: string
  cover?: string
  seed: string
  /** 艺人用圆形头像 */
  round?: boolean
  icon?: Component
}>(), { subtitle: '', cover: '', round: false, icon: () => MusicNoteIcon })

const emit = defineEmits<{ (e: 'open'): void; (e: 'play'): void }>()
</script>
