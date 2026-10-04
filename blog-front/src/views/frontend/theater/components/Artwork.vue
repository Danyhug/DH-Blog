<template>
  <!-- 封面：图片加载失败或根本没有封面时，用由名称派生的渐变色 + 图标占位，
       同一张专辑/同一部片子每次看到的占位色都一样。 -->
  <div class="relative overflow-hidden" :style="showImage ? undefined : { background: gradientFor(seed) }">
    <img
      v-if="showImage"
      :src="src"
      :alt="alt"
      loading="lazy"
      decoding="async"
      class="absolute inset-0 size-full object-cover"
      @error="failed = true"
    />
    <div v-else class="absolute inset-0 flex items-center justify-center text-white/80">
      <slot name="placeholder">
        <component :is="icon" class="size-1/3 max-w-16" />
      </slot>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type Component } from 'vue'
import { gradientFor } from '../utils/format'
import { MusicNoteIcon } from './icons'

const props = withDefaults(defineProps<{
  src?: string
  seed: string
  alt?: string
  icon?: Component
}>(), {
  src: '',
  alt: '',
  icon: () => MusicNoteIcon
})

const failed = ref(false)
watch(() => props.src, () => (failed.value = false))
const showImage = computed(() => !!props.src && !failed.value)
</script>
