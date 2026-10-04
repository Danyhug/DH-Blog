<template>
  <!-- 索引里有、磁盘上没有的媒体文件：它们在网盘列表里看得到，却放不了，必须说清楚为什么没出现在影院里 -->
  <div v-if="missing.count" class="glass flex gap-3 rounded-2xl px-4 py-3 text-left text-sm leading-relaxed text-amber-800 dark:text-[#f5d39a]">
    <InfoIcon class="mt-0.5 size-4 shrink-0 text-[#f5a524]" />
    <div class="min-w-0">
      <p class="m-0">
        网盘列表里有 <strong>{{ missing.count }}</strong> 个{{ kind }}文件在存储目录中找不到，已跳过：
        <span class="text-black/80 dark:text-white/80">{{ missing.samples.join('、') }}{{ missing.count > missing.samples.length ? ' 等' : '' }}</span>
      </p>
      <p class="m-0 mt-1 text-xs text-black/50 dark:text-white/50">
        通常是文件在 DH-Blog 之外被移走、删除，或者后台「文件存储路径」改过。把文件放回存储目录或重新上传；
        确认不要了，就在网盘里点「从磁盘同步」清掉这些失效记录。
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { MissingMedia } from '@/api/media'
import { InfoIcon } from './icons'

defineProps<{ missing: MissingMedia; kind: string }>()
</script>
