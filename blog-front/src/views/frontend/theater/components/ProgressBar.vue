<template>
  <!-- 可拖动的进度条：视频进度、音乐进度、音量共用。
       拖动过程中只回显本地比例（scrub），松手才提交 seek，避免拖动时反复跳播。 -->
  <div
    ref="trackRef"
    class="group/bar relative flex items-center cursor-pointer touch-none select-none"
    :class="hitClass"
    @pointerdown="onDown"
    @pointermove="onMove"
    @pointerup="onUp"
    @pointercancel="onCancel"
    @pointerleave="hoverRatio = null"
  >
    <div
      class="relative w-full overflow-hidden rounded-full transition-[height] duration-150"
      :class="[trackClass, dragging ? dragHeightClass : heightClass]"
    >
      <div v-if="buffered" class="absolute inset-y-0 left-0 bg-black/15 dark:bg-white/25" :style="{ width: percent(buffered) }"></div>
      <div v-if="hoverRatio !== null && hoverPreview" class="absolute inset-y-0 left-0 bg-black/10 dark:bg-white/20" :style="{ width: percent(hoverRatio) }"></div>
      <div class="absolute inset-y-0 left-0" :class="fillClass" :style="{ width: percent(shown) }"></div>
    </div>
    <div
      v-if="knob"
      class="pointer-events-none absolute top-1/2 size-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full shadow transition-transform duration-150"
      :class="[knobClass, dragging ? 'scale-100' : 'scale-0 group-hover/bar:scale-100']"
      :style="{ left: percent(shown) }"
    ></div>
    <div
      v-if="hoverRatio !== null && $slots.tooltip"
      class="pointer-events-none absolute bottom-full mb-3 -translate-x-1/2"
      :style="{ left: percent(hoverRatio) }"
    >
      <slot name="tooltip" :ratio="hoverRatio" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = withDefaults(defineProps<{
  /** 0..1 */
  value: number
  /** 已缓冲比例 0..1 */
  buffered?: number
  fillClass?: string
  trackClass?: string
  knobClass?: string
  knob?: boolean
  /** 静止 + 悬停时的高度，须写成完整类名（Tailwind 只识别字面量） */
  heightClass?: string
  /** 拖动中的高度 */
  dragHeightClass?: string
  /** 可点击区域的高度，比可见条更高，方便命中 */
  hitClass?: string
  hoverPreview?: boolean
}>(), {
  buffered: 0,
  fillClass: 'bg-zinc-800 dark:bg-white',
  trackClass: 'bg-black/15 dark:bg-white/20',
  knobClass: 'bg-zinc-800 dark:bg-white',
  knob: true,
  heightClass: 'h-1 group-hover/bar:h-1.5',
  dragHeightClass: 'h-1.5',
  hitClass: 'h-4',
  hoverPreview: false
})

const emit = defineEmits<{
  (e: 'scrub', ratio: number): void
  (e: 'seek', ratio: number): void
}>()

const trackRef = ref<HTMLElement | null>(null)
const dragging = ref(false)
const dragRatio = ref(0)
const hoverRatio = ref<number | null>(null)

const shown = computed(() => (dragging.value ? dragRatio.value : clamp(props.value)))

function clamp(value: number) {
  return Number.isFinite(value) ? Math.min(1, Math.max(0, value)) : 0
}

function percent(value: number) {
  return `${clamp(value) * 100}%`
}

function ratioAt(event: PointerEvent) {
  const rect = trackRef.value!.getBoundingClientRect()
  return clamp((event.clientX - rect.left) / rect.width)
}

function onDown(event: PointerEvent) {
  if (event.button !== 0) return
  trackRef.value?.setPointerCapture(event.pointerId)
  dragging.value = true
  dragRatio.value = ratioAt(event)
  emit('scrub', dragRatio.value)
}

function onMove(event: PointerEvent) {
  const ratio = ratioAt(event)
  hoverRatio.value = ratio
  if (!dragging.value) return
  dragRatio.value = ratio
  emit('scrub', ratio)
}

function onUp(event: PointerEvent) {
  if (!dragging.value) return
  dragging.value = false
  trackRef.value?.releasePointerCapture(event.pointerId)
  emit('seek', ratioAt(event))
}

function onCancel() {
  dragging.value = false
}
</script>
