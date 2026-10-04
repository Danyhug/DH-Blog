<template>
  <!-- 日间/夜间切换：与博客文章页共用同一个偏好，网盘、影院里切了，回博客也是同一套 -->
  <button type="button" :title="label" :aria-label="label" :aria-pressed="reading.isDarkMode" @click="toggle">
    <Sunny v-if="reading.isDarkMode" :class="iconClass" />
    <Moon v-else :class="iconClass" />
    <!-- 需要文字时（侧边栏）由使用方放进来 -->
    <slot :dark="reading.isDarkMode" />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Moon, Sunny } from '@element-plus/icons-vue'
import { useUserStore } from '@/store'
import { revealThemeChange } from '@/utils/themeTransition'

withDefaults(defineProps<{ iconClass?: string }>(), { iconClass: 'size-[18px]' })

const reading = useUserStore().aritcleModel
const label = computed(() => (reading.isDarkMode ? '切换日间模式' : '切换夜间模式'))

function toggle(event: MouseEvent) {
  revealThemeChange(event.currentTarget as HTMLElement, () => {
    reading.isDarkMode = !reading.isDarkMode
  })
}
</script>
