<template>
  <!-- 影院的氛围背景：一张被极度模糊的图 + 几团缓慢漂移的光斑。
       毛玻璃只有在背后有颜色时才看得出来，这一层就是给玻璃「透」的东西。 -->
  <div class="pointer-events-none fixed inset-0 z-0 overflow-hidden bg-[#f5f5f7] dark:bg-[#07070a]">
    <transition
      enter-active-class="transition-opacity duration-[1200ms] ease-out"
      enter-from-class="opacity-0"
      leave-active-class="transition-opacity duration-[1200ms] ease-in absolute inset-0"
      leave-to-class="opacity-0"
    >
      <img v-if="image" :key="image" :src="image" alt="" class="absolute inset-0 size-full scale-125 object-cover opacity-45 blur-[90px] saturate-[1.8] dark:opacity-60" />
    </transition>

    <!-- TV 模式下光斑不再漂移：大面积模糊层每帧重绘，电视盒子的 GPU 吃不消（tv 变体见 tailwind.css） -->
    <div class="absolute -left-[10%] -top-[15%] size-[55vw] animate-ambient-drift rounded-full tv:animate-none blur-[120px]" :class="palette[0]"></div>
    <div class="absolute -right-[15%] top-[20%] size-[50vw] animate-ambient-drift rounded-full tv:animate-none blur-[130px] [animation-delay:-9s] [animation-duration:34s]" :class="palette[1]"></div>
    <div class="absolute -bottom-[25%] left-[25%] size-[45vw] animate-ambient-drift rounded-full tv:animate-none blur-[120px] [animation-delay:-17s] [animation-duration:40s]" :class="palette[2]"></div>

    <!-- 日间罩一层白雾、夜间压暗，前景文字两种模式下都读得清 -->
    <div class="absolute inset-0 bg-gradient-to-b from-white/10 via-white/30 to-white/60 dark:from-black/25 dark:via-black/45 dark:to-black/75"></div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  image: string
  /** cinema：Netflix 的红与深紫；music：Apple Music 的粉红与蓝紫 */
  mode: 'cinema' | 'music'
}>()

const palette = computed(() =>
  props.mode === 'cinema'
    ? ['bg-[#e50914]/30', 'bg-[#5b21b6]/25', 'bg-[#0e7490]/20']
    : ['bg-[#fa2d48]/30', 'bg-[#5e5ce6]/30', 'bg-[#ff9f0a]/15']
)
</script>
