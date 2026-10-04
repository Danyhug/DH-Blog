<template>
  <!-- Netflix 的横向片单行：标题 + 左右翻页箭头（悬停整行时出现） -->
  <section class="group/row relative mb-10 max-md:mb-6">
    <h2 class="m-0 mb-1 px-[4%] text-[1.35vw] font-bold text-black/90 dark:text-white/90 dark:drop-shadow max-lg:text-lg">{{ title }}</h2>
    <div class="relative">
      <button
        v-show="canLeft"
        class="glass-button absolute left-[1%] top-1/2 z-20 flex size-11 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full text-zinc-900 dark:text-white opacity-0 transition-opacity group-hover/row:opacity-100 max-md:hidden"
        title="上一页"
        @click="page(-1)"
      >
        <ChevronLeftIcon class="size-6" />
      </button>
      <div ref="rowRef" class="flex gap-3 overflow-x-auto scroll-smooth px-[4%] py-5 [scrollbar-width:none] max-md:gap-1.5 max-md:py-2" @scroll="update">
        <slot />
      </div>
      <button
        v-show="canRight"
        class="glass-button absolute right-[1%] top-1/2 z-20 flex size-11 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full text-zinc-900 dark:text-white opacity-0 transition-opacity group-hover/row:opacity-100 max-md:hidden"
        title="下一页"
        @click="page(1)"
      >
        <ChevronRightIcon class="size-6" />
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, onUpdated, ref } from 'vue'
import { ChevronLeftIcon, ChevronRightIcon } from '../components/icons'

defineProps<{ title: string }>()

const rowRef = ref<HTMLElement | null>(null)
const canLeft = ref(false)
const canRight = ref(false)

function update() {
  const row = rowRef.value
  if (!row) return
  canLeft.value = row.scrollLeft > 4
  canRight.value = row.scrollLeft + row.clientWidth < row.scrollWidth - 4
}

function page(direction: number) {
  const row = rowRef.value
  if (row) row.scrollBy({ left: direction * row.clientWidth * 0.92 })
}

onMounted(() => {
  nextTick(update)
  window.addEventListener('resize', update)
})
// 片单内容变化（加载完成、移出继续观看）后箭头的可用状态也要跟着变
onUpdated(update)
onUnmounted(() => window.removeEventListener('resize', update))
</script>
