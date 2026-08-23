<template>
  <!-- 网盘文件列表页与文件预览页共用的顶部导航头。
       sticky：分享页是独立页面，需要吸顶栏体并居中收窄；
       嵌在网盘内部时只是内容区顶部的一行，不要栏体、不居中、贴左边缘。 -->
  <div
    class="drive-header"
    :class="sticky
      ? 'sticky top-0 z-10 backdrop-blur-[20px] bg-white/90 border-b border-black/5 shadow-[0_4px_20px_rgba(0,0,0,0.05)] py-2.5'
      : 'shrink-0 mb-5'"
  >
    <!-- 窄屏只有在有标题时才换成竖排：标题要单独占一行并提到最前面（order-[-1]）；
         没有标题的列表页保持横排，否则右侧单个按钮会被挤到左边。 -->
    <div
      class="flex justify-between items-center gap-3"
      :class="[
        sticky ? 'max-w-[1400px] mx-auto px-6 py-3 max-md:px-3' : '',
        hasTitle ? 'max-md:flex-col' : '',
      ]"
    >
      <div class="flex-[2] min-w-0" :class="hasTitle ? 'max-md:w-full' : ''">
        <slot name="left" />
      </div>

      <div v-if="hasTitle" class="flex-1 min-w-0 text-center max-md:w-full max-md:order-[-1]">
        <slot name="title">
          <h2 class="inline-block max-w-[400px] max-md:max-w-full m-0 px-4 py-2 text-lg font-semibold text-[#333] whitespace-nowrap overflow-hidden text-ellipsis rounded-lg bg-white/80 backdrop-blur-[4px] shadow-[0_2px_8px_rgba(0,0,0,0.03)]">{{ title }}</h2>
        </slot>
      </div>

      <div
        class="flex-[2] flex justify-end items-center gap-3"
        :class="hasTitle ? 'max-md:w-full max-md:justify-between' : ''"
      >
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue'

const props = defineProps<{
  /** 独立页面（分享页）用吸顶栏体，嵌入网盘时留空 */
  sticky?: boolean
  /** 居中标题，列表页不需要 */
  title?: string
}>()

const slots = useSlots()
const hasTitle = computed(() => Boolean(props.title || slots.title))
</script>
