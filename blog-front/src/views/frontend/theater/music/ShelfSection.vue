<template>
  <!-- 首页的一行横向货架：标题 + 「查看全部」+ 可左右滚动的内容 -->
  <section class="mb-10">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="m-0 flex items-center gap-1 text-xl font-bold">
        <button class="flex cursor-pointer items-center gap-1 border-none bg-transparent p-0 text-inherit hover:text-black/80 dark:hover:text-white/80" @click="emit('more')">
          {{ title }}<ChevronRightIcon class="size-5 text-black/40 dark:text-white/40" />
        </button>
      </h2>
      <div class="flex gap-2 max-md:hidden">
        <button :class="arrowClass" title="向左" @click="scroll(-1)"><ChevronLeftIcon class="size-4" /></button>
        <button :class="arrowClass" title="向右" @click="scroll(1)"><ChevronRightIcon class="size-4" /></button>
      </div>
    </div>
    <div ref="rowRef" class="-mx-1 flex snap-x gap-5 overflow-x-auto scroll-smooth px-1 pb-2 [scrollbar-width:none]">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ChevronLeftIcon, ChevronRightIcon } from '../components/icons'

defineProps<{ title: string }>()
const emit = defineEmits<{ (e: 'more'): void }>()
const rowRef = ref<HTMLElement | null>(null)
const arrowClass = 'glass-button flex size-8 cursor-pointer items-center justify-center rounded-full text-black/80 dark:text-white/80 hover:text-zinc-900 dark:hover:text-white'

function scroll(direction: number) {
  const row = rowRef.value
  if (row) row.scrollBy({ left: direction * row.clientWidth * 0.85 })
}
</script>
