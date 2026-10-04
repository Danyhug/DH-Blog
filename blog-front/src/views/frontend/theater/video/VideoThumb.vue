<template>
  <!-- 视频缩略图：同目录海报/剧照优先，其次从视频里截一帧，都不行就用渐变 + 片名占位 -->
  <div ref="rootRef" class="relative overflow-hidden bg-[#2a2a2a]" :style="src ? undefined : { background: gradientFor(seed) }">
    <img v-if="src" :src="src" alt="" class="absolute inset-0 size-full object-cover" draggable="false" @error="onImageError" />
    <div v-else-if="label" class="absolute inset-0 flex items-end p-3">
      <span class="line-clamp-2 text-lg font-black leading-tight text-white/90 drop-shadow-lg">{{ label }}</span>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { mediaStreamUrl } from '@/api/media'
import { captureFrame } from './frameCapture'
import { gradientFor } from '../utils/format'

const props = withDefaults(defineProps<{
  /** 海报或剧照的文件 ID */
  imageId?: string
  /** 没有图片时从这个视频截帧 */
  frameId?: string
  seed: string
  /** 占位时显示的文字 */
  label?: string
}>(), { imageId: '', frameId: '', label: '' })

const rootRef = ref<HTMLElement | null>(null)
const src = ref('')
let observer: IntersectionObserver | null = null
let requested = false

function resolveImage() {
  requested = false
  src.value = props.imageId ? mediaStreamUrl(props.imageId) : ''
  if (!src.value) requestFrame()
}

function onImageError() {
  src.value = ''
  requestFrame()
}

// 只在缩略图进入视口后才去截帧，一整行几十部片子不会同时开几十个视频连接
function requestFrame() {
  if (!props.frameId || requested) return
  requested = true
  const frameId = props.frameId
  const start = () => captureFrame(frameId).then(data => {
    if (props.frameId === frameId && !props.imageId && data) src.value = data
  })
  if (!('IntersectionObserver' in window) || !rootRef.value) return void start()
  observer?.disconnect()
  observer = new IntersectionObserver(entries => {
    if (entries.some(entry => entry.isIntersecting)) {
      observer?.disconnect()
      start()
    }
  }, { rootMargin: '200px' })
  observer.observe(rootRef.value)
}

watch(() => [props.imageId, props.frameId], resolveImage)
onMounted(resolveImage)
onUnmounted(() => observer?.disconnect())
</script>
