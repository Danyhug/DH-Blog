<template>
  <div class="w-full">
    <el-row>
      <el-col>
        <el-card class="w-full min-h-screen bg-white! dark:bg-[#191919]! dark:text-[#d4d4d4]! dark:border-[#303030]! dark:[--el-bg-color:#191919] dark:[--el-text-color-regular:#aaa] dark:[--el-border-color:#303030]">
          <img :src="store.homeHeaderInfo.thumbnailUrl" class="image w-[95%] mx-auto" />

          <div>
            <p class="title text-[1.15rem] my-3">{{ store.homeHeaderInfo.title }}</p>
            <div class="schedule">
              <el-progress :color="customColors" :percentage="sideInfo.process"></el-progress>
              <p class="text-[14px] text-[#606266] dark:text-[#aaa]">已阅读时长：{{ formatSeconds(second) }}</p>
            </div>
          </div>

          <ArticleReadingTools class="mt-[14px] pt-2" />

          <div class="tags w-full text-[#909399] text-[14px]">
            <el-divider>
              <Icon iconName="icon-shili" iconSize="1.56"></Icon>
            </el-divider>
            <div class="tag-list flex flex-wrap gap-2 w-full mb-4">
              <span class="tag inline-flex items-center max-w-full rounded-full py-0.5 px-2.5 text-[13px] leading-[1.6] text-(--tag) bg-(--tag)/10 dark:bg-(--tag)/20 dark:text-[color-mix(in_oklab,var(--tag)_60%,white)] [overflow-wrap:anywhere] [word-break:break-word] whitespace-normal" v-for="(item, index) in store.homeHeaderInfo.tags" :key="item.id ?? item.name"
                :style="{ '--tag': tagColors[index] }"><span class="mr-0.5 opacity-60" aria-hidden="true">#</span>{{ item.name }}</span>
            </div>
          </div>

          <div class="catelog flex-1 overflow-y-auto text-left">
            <MdCatalog editorId="dh-editor" :scrollElement="scrollElement" :theme="store.aritcleModel.isDarkMode ? 'dark' : 'light'" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { useUserStore } from '@/store';
import ArticleReadingTools from '@/components/frontend/ArticleReadingTools.vue';
import { computed, reactive, onMounted, onBeforeUnmount, ref } from 'vue'
import { debounce } from '@/utils/tool'
const store = useUserStore()

const scrollElement = document.documentElement

// Ordered by hue: red → orange → … → pink → rose.
const tagPalette = [
  "#dc2626", "#ea580c", "#d97706", "#65a30d", "#16a34a", "#0d9488", "#0891b2",
  "#0284c7", "#2563eb", "#4f46e5", "#7c3aed", "#c026d3", "#db2777", "#e11d48",
]
// Coprime with the palette length, so probing visits every colour; 5 of 14 also jumps well away in hue.
const TAG_PROBE_STEP = 5
// Each tag starts from the colour its name hashes to, so it keeps that colour across articles.
// Within one article it moves on a clash: first to a colour whose hue neighbours are free too
// (red beside rose reads as the same colour), else to any unused one, so no two chips share a colour.
const tagColors = computed(() => {
  const size = tagPalette.length
  const used = new Set()
  const isolated = (index) => [size - 1, 0, 1].every(offset => !used.has((index + offset) % size))
  return (store.homeHeaderInfo.tags ?? []).map(({ name }) => {
    let hash = 0
    for (const char of name) hash = (hash * 31 + char.codePointAt(0)) >>> 0
    const probes = Array.from({ length: size }, (_, step) => (hash + step * TAG_PROBE_STEP) % size)
    const index = probes.find(isolated) ?? probes.find(index => !used.has(index)) ?? probes[0]
    used.add(index)
    return tagPalette[index]
  })
})

const customColors = [
  { color: 'rgb(57,157,254)', percentage: 80 },
  { color: 'rgb(0,207,102)', percentage: 100 },
]

const sideInfo = reactive({
  process: 0
})

// 计算分钟和秒
function formatSeconds(seconds) {
  if (seconds >= 60) {
    var minutes = Math.floor(seconds / 60);
    var remainingSeconds = seconds % 60;
    return minutes + '分' + remainingSeconds + '秒';
  } else {
    return seconds + '秒';
  }
}

let timer = null
let second = ref(0)
const scrollListener = debounce(() => {
  let scrollTop = window.scrollY

  // 获取div的高度
  const container = document.querySelector('.blog-container')
  if (!container || store.aritcleModel.isFullPreview) return
  let height = container.scrollHeight

  // 计算阅读百分比
  let process = Math.floor((scrollTop / height) * 100)
  sideInfo.process = process > 100 ? 100 : process
}, 16)

onMounted(() => {
  window.addEventListener('scroll', scrollListener)
  scrollTo(0, 60)

  timer = setInterval(() => second.value++, 1000)
})

onBeforeUnmount(() => {
  clearInterval(timer)
  window.removeEventListener('scroll', scrollListener)
})

</script>

<style lang="less" scoped>
/* :deep()：EP 卡片内容区加不上 class */
:deep(.el-card__body) {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

/* 裸伪元素选择器，无工具类写法 */
::-webkit-scrollbar {
  width: 0;
}
</style>
