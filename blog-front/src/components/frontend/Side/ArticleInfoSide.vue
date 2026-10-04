<template>
  <div class="w-full">
    <el-row>
      <el-col>
        <el-card class="w-full min-h-screen bg-white! dark:bg-[#191919]! dark:text-[#d4d4d4]! dark:border-[#303030]! dark:[--el-bg-color:#191919] dark:[--el-text-color-regular:#aaa] dark:[--el-border-color:#303030]">
          <!-- 加载中先按默认封面的 8:5 占位，侧栏下方内容不会在图片到达时整体下移太多 -->
          <img v-img-fade="'aspect-[8/5]'" :src="store.homeHeaderInfo.thumbnailUrl" alt="" decoding="async" class="image w-[95%] mx-auto" />

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
              <!-- 弹层不设 effect：默认 light 在 html.dark 下已取 Element Plus 的夜间变量；
                   el-zoom-in-top 是 Element Plus 自带的过渡（下拉框同款），弹层从标签处向下展开 -->
              <el-popover v-for="(item, index) in store.homeHeaderInfo.tags" :key="item.id ?? item.name"
                trigger="click" placement="bottom" :width="300" :persistent="false" transition="el-zoom-in-top"
                popper-class="p-0! overflow-hidden rounded-xl! shadow-lg!"
                :visible="openTag === item.name" @update:visible="visible => visible ? onTagShow(item.name) : onTagHide(item.name)">
                <template #reference>
                  <button type="button" :aria-label="`查看标签 ${item.name} 的相关文章`" :aria-expanded="openTag === item.name"
                    class="tag inline-flex items-center max-w-full cursor-pointer rounded-full py-0.5 px-2.5 text-[13px] leading-[1.6] text-left text-(--tag) bg-(--tag)/10 hover:bg-(--tag)/20 hover:-translate-y-0.5 hover:shadow-sm active:scale-95 dark:bg-(--tag)/20 dark:hover:bg-(--tag)/30 dark:text-[color-mix(in_oklab,var(--tag)_60%,white)] [overflow-wrap:anywhere] [word-break:break-word] whitespace-normal transition-[background-color,translate,scale,box-shadow] duration-200 focus-visible:outline-2 focus-visible:outline-(--tag)"
                    :class="{ 'bg-(--tag)/20! ring-1 ring-(--tag)/40 dark:bg-(--tag)/30!': openTag === item.name }"
                    :style="{ '--tag': tagColors[index] }"><span class="mr-0.5 opacity-60" aria-hidden="true">#</span>{{ item.name }}</button>
                </template>
                <div role="group" :aria-label="`标签 ${item.name} 的相关文章`" :style="{ '--tag': tagColors[index] }">
                  <div class="flex items-center justify-between gap-3 px-4 py-3 bg-linear-to-br from-(--tag)/15 to-transparent border-b border-(--tag)/15">
                    <div class="min-w-0">
                      <p class="m-0 truncate text-[15px] font-semibold text-(--tag) dark:text-[color-mix(in_oklab,var(--tag)_60%,white)]">
                        <span class="mr-0.5 opacity-60" aria-hidden="true">#</span>{{ item.name }}
                      </p>
                      <p class="m-0 mt-0.5 text-[12px] text-[#909399]">相关文章</p>
                    </div>
                    <span v-if="tagArticles[item.name]"
                      class="shrink-0 rounded-full bg-(--tag)/15 px-2 py-0.5 text-[12px] text-(--tag) dark:text-[color-mix(in_oklab,var(--tag)_60%,white)] animate-fade-in">{{ tagArticles[item.name].length }} 篇</span>
                  </div>
                  <!-- overscroll-contain：列表滚到头不把滚动传给页面，否则页面一动弹层就被关掉 -->
                  <div class="max-h-[320px] overflow-y-auto overscroll-contain p-2">
                    <!-- 骨架屏：与列表项同尺寸，数据回来时不跳动 -->
                    <div v-if="!tagArticles[item.name]" aria-busy="true" aria-label="加载中">
                      <div v-for="n in 3" :key="n" class="flex items-center gap-3 px-2.5 py-2 animate-pulse">
                        <div class="size-6 shrink-0 rounded-md bg-black/5 dark:bg-white/10"></div>
                        <div class="flex-1 space-y-1.5">
                          <div class="h-3 w-4/5 rounded bg-black/5 dark:bg-white/10"></div>
                          <div class="h-2.5 w-2/5 rounded bg-black/5 dark:bg-white/10"></div>
                        </div>
                      </div>
                    </div>
                    <p v-else-if="!tagArticles[item.name].length" class="m-0 py-6 text-center text-[13px] text-[#909399] animate-fade-in">暂无相关文章</p>
                    <ul v-else class="m-0 p-0 list-none">
                      <li v-for="(article, i) in tagArticles[item.name]" :key="article.id"
                        class="animate-tag-item-in" :style="{ animationDelay: `${Math.min(i, 8) * 40}ms` }">
                        <div v-if="article.id === store.homeHeaderInfo.id" aria-current="page"
                          class="flex items-center gap-3 rounded-lg px-2.5 py-2 bg-(--tag)/8 dark:bg-(--tag)/15">
                          <span class="grid size-6 shrink-0 place-items-center rounded-md bg-(--tag) text-[12px] text-white">{{ i + 1 }}</span>
                          <div class="min-w-0 flex-1">
                            <p class="m-0 line-clamp-2 text-[13px] leading-snug text-(--tag) dark:text-[color-mix(in_oklab,var(--tag)_60%,white)]">{{ article.title }}</p>
                            <p class="m-0 mt-0.5 text-[12px] text-[#a8abb2]">正在阅读</p>
                          </div>
                        </div>
                        <button v-else type="button" @click="goToArticle(article.id)"
                          class="group flex w-full cursor-pointer items-center gap-3 rounded-lg px-2.5 py-2 text-left transition-colors duration-200 hover:bg-(--tag)/10 dark:hover:bg-(--tag)/20 focus-visible:outline-2 focus-visible:outline-(--tag)">
                          <span class="grid size-6 shrink-0 place-items-center rounded-md bg-black/5 text-[12px] text-[#909399] transition-colors duration-200 group-hover:bg-(--tag) group-hover:text-white dark:bg-white/10">{{ i + 1 }}</span>
                          <div class="min-w-0 flex-1">
                            <p class="m-0 line-clamp-2 text-[13px] leading-snug text-[#303133] transition-colors duration-200 group-hover:text-(--tag) dark:text-[#d4d4d4] dark:group-hover:text-[color-mix(in_oklab,var(--tag)_60%,white)]">{{ article.title }}</p>
                            <p class="m-0 mt-0.5 text-[12px] text-[#a8abb2]">{{ article.createTime?.slice(0, 10) }} · {{ article.views ?? 0 }} 次阅读</p>
                          </div>
                          <ArrowRight class="size-3.5 shrink-0 text-(--tag) opacity-0 -translate-x-1 transition-[opacity,translate] duration-200 group-hover:opacity-100 group-hover:translate-x-0 group-focus-visible:opacity-100 group-focus-visible:translate-x-0" aria-hidden="true" />
                        </button>
                      </li>
                    </ul>
                  </div>
                </div>
              </el-popover>
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
import { useRouter } from 'vue-router'
import { ArrowRight } from '@element-plus/icons-vue'
import { getArticlesByTaxonomy } from '@/api/user'
import { debounce } from '@/utils/tool'
import { vImgFade } from '@/directives/imgFade'
import { TAG_PALETTE as tagPalette, tagNameHash } from '@/utils/tagColor'
const store = useUserStore()
const router = useRouter()

// Keyed by tag name; fetched on first open only, so reopening a tag's popover is instant.
const tagArticles = reactive({})
const loadTagArticles = async (name) => {
  if (tagArticles[name]) return
  try {
    tagArticles[name] = (await getArticlesByTaxonomy(name, 'tag')) ?? []
  } catch {
    // Leave it unset so the next open retries; the axios interceptor already surfaced the error.
  }
}
// Controls which tag's popover is open (at most one) and drives the chip's pressed look.
const openTag = ref('')
const onTagShow = (name) => {
  openTag.value = name
  loadTagArticles(name)
}
// Switching chips fires the new show before the old hide, so only clear our own name.
const onTagHide = (name) => {
  if (openTag.value === name) openTag.value = ''
}
// The popper is teleported to <body> and only repositions after each scroll event, so it visibly
// trails the sticky sidebar while the page moves; close it instead, like a native dropdown.
const closeTagPopover = () => {
  openTag.value = ''
}
const goToArticle = (id) => router.push({ name: 'ArticleInfo', params: { id } })

const scrollElement = document.documentElement

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
    const hash = tagNameHash(name)
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
  window.addEventListener('scroll', closeTagPopover, { passive: true })
  scrollTo(0, 60)

  timer = setInterval(() => second.value++, 1000)
})

onBeforeUnmount(() => {
  clearInterval(timer)
  window.removeEventListener('scroll', scrollListener)
  window.removeEventListener('scroll', closeTagPopover)
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
