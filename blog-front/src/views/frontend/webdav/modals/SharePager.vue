<template>
  <nav v-if="totalPages > 1"
    class="flex flex-wrap items-center justify-center gap-1 pt-3 mt-2 border-t border-[rgba(229,231,235,0.8)] dark:border-white/10"
    :aria-label="label">
    <button type="button" :class="stepClass" :disabled="page <= 1" aria-label="上一页" @click="go(page - 1)">‹</button>
    <template v-for="(item, index) in pages" :key="`${item}-${index}`">
      <span v-if="item === 'gap'" class="px-1 text-[0.75rem] text-[#9ca3af] dark:text-zinc-500" aria-hidden="true">…</span>
      <button v-else type="button" :class="item === page ? activePageClass : pageClass"
        :aria-current="item === page ? 'page' : undefined" @click="go(item)">{{ item }}</button>
    </template>
    <button type="button" :class="stepClass" :disabled="page >= totalPages" aria-label="下一页"
      @click="go(page + 1)">›</button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface SharePagerProps {
  page: number
  totalPages: number
  // 分享列表和访问日志都用它，读屏时要能分清是哪一处翻页
  label: string
}

const props = defineProps<SharePagerProps>()
const emit = defineEmits<{ 'update:page': [page: number] }>()

// 这个翻页器在网盘里，跟同目录其它控件一样不引 Element Plus，样式全走工具类。
// 激活态和普通态是互斥的两套类，不能叠加：Tailwind 里 .bg-transparent 和 .text-[#4b5563]
// 在产物中的位置都排在 .bg-[#2563eb] / .text-white 之后，同优先级会把当前页的底色盖掉
// （夜间看不出问题，因为 dark: 变体多了 .dark 前缀、特异性更高）。
const baseClass = 'min-w-7 h-7 px-2 border-none rounded-md cursor-pointer text-[0.75rem] tabular-nums transition-colors duration-200 focus-visible:outline-2 focus-visible:outline-[#2563eb] focus-visible:outline-offset-1'
const hoverClass = 'hover:bg-[rgba(156,163,175,0.12)] dark:hover:bg-white/10'
const stepClass = `${baseClass} bg-transparent text-[#6b7280] dark:text-zinc-400 ${hoverClass} disabled:text-[#d1d5db] dark:disabled:text-zinc-600 disabled:cursor-not-allowed`
const pageClass = `${baseClass} bg-transparent text-[#4b5563] dark:text-zinc-300 ${hoverClass}`
const activePageClass = `${baseClass} font-semibold text-white bg-[#2563eb] dark:bg-[#3b82f6]`

/**
 * 最多画 7 个格子：首页、末页固定，中间跟着当前页滑动，跳过的区段用省略号。
 * 影集式的窗口比「上一页 / 3 / 下一页」多给了一个「一共几页、现在在哪」的直观感。
 */
const pages = computed<Array<number | 'gap'>>(() => {
  const total = props.totalPages
  if (total <= 7) {
    return Array.from({ length: total }, (_, index) => index + 1)
  }

  const current = Math.min(Math.max(props.page, 1), total)
  const anchors = [...new Set([1, current - 1, current, current + 1, total])]
    .filter((value) => value >= 1 && value <= total)
    .sort((left, right) => left - right)

  const items: Array<number | 'gap'> = []
  anchors.forEach((value, index) => {
    if (index > 0 && value - anchors[index - 1] > 1) items.push('gap')
    items.push(value)
  })
  return items
})

function go(next: number) {
  if (next < 1 || next > props.totalPages || next === props.page) return
  emit('update:page', next)
}
</script>
