<template>
  <!-- 网盘路径面包屑。「我的网盘」是固定根节点，进子目录后也要保留，否则只剩一个图标。 -->
  <nav class="inline-flex max-w-full items-center gap-2 overflow-x-auto text-sm bg-[#f8f9fa] px-4 py-2.5 rounded-[50px] shadow-[0_2px_8px_rgba(0,0,0,0.06)]">
    <HomeIcon class="shrink-0 cursor-pointer text-[#666] w-4 h-4 transition-all duration-200 hover:text-[#2a8aff]" @click="emit('navigate-root')" />
    <span :class="SEGMENT_CLASS" @click="emit('navigate-root')">我的网盘</span>
    <template v-for="(segment, index) in segments" :key="segment.id || index">
      <ChevronRightIcon class="shrink-0 text-[#aaa] w-3 h-3" />
      <span :class="SEGMENT_CLASS" @click="emit('navigate-segment', index)">{{ segment.name }}</span>
    </template>
  </nav>
</template>

<script setup lang="ts">
import type { PathSegment } from '../utils/types/file'
import { HomeIcon, ChevronRightIcon } from '../utils/icons'

defineProps<{
  segments: PathSegment[]
}>()

const emit = defineEmits<{
  (e: 'navigate-root'): void
  (e: 'navigate-segment', index: number): void
}>()

const SEGMENT_CLASS = 'cursor-pointer whitespace-nowrap text-[#666] font-medium px-2 py-0.5 rounded transition-all duration-200 hover:text-[#2a8aff] hover:bg-[rgba(42,138,255,0.1)]'
</script>
