<template>
  <div
    class="group flex cursor-pointer items-center gap-3 rounded-xl border border-transparent px-2 py-2 transition-colors hover:bg-black/5 focus-visible:bg-black/5 dark:hover:bg-white/10 dark:focus-visible:bg-white/10"
    :class="active && 'glass'"
    tabindex="0"
    role="button"
    :aria-label="track.title"
    @click="emit('play')"
    @keydown.enter.self.prevent="emit('play')"
  >
    <Artwork :src="trackCoverUrl(track)" :seed="track.album" class="size-11 shrink-0 rounded-[10px] ring-1 ring-black/5 dark:ring-white/10" />
    <div class="min-w-0 flex-1">
      <p class="m-0 truncate text-[15px] font-medium" :class="active ? 'text-[#fa2d48]' : 'text-zinc-900 dark:text-white'">{{ track.title }}</p>
      <p class="m-0 truncate text-sm text-black/50 dark:text-white/50">{{ track.artist }}</p>
    </div>
    <span class="text-xs tabular-nums text-black/40 dark:text-white/40">{{ track.duration ? formatTime(track.duration) : '' }}</span>
    <button
      v-if="removable"
      class="flex size-7 cursor-pointer items-center justify-center rounded-full border-none bg-transparent text-black/50 dark:text-white/50 opacity-0 transition hover:bg-black/5 dark:hover:bg-white/10 hover:text-zinc-900 dark:hover:text-white group-hover:opacity-100"
      title="从待播清单移除"
      @click.stop="emit('remove')"
    >
      <CloseIcon class="size-4" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { trackCoverUrl, type Track } from '@/api/media'
import Artwork from '../components/Artwork.vue'
import { CloseIcon } from '../components/icons'
import { formatTime } from '../utils/format'

defineProps<{ track: Track; active?: boolean; removable?: boolean }>()
const emit = defineEmits<{ (e: 'play'): void; (e: 'remove'): void }>()
</script>
