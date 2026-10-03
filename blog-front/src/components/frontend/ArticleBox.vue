<template>
  <article class="type-img-left">
    <router-link :to="'./article/' + article.id">
      <div class="cover flex w-full min-h-[244px] rounded-2xl bg-grey-0 overflow-hidden border border-[#eee] dark:bg-[#202020] dark:border-[#303030]">
        <div class="left w-1/2 relative overflow-hidden [clip-path:polygon(0_0,92%_0%,100%_100%,0%_100%)]">
          <img :src="getArticleBg(article.thumbnailUrl, article.id)" :alt="article.title" loading="lazy"
            class="block w-full h-full object-cover [transition:all_0.5s]">
        </div>
        <!-- 手机端 style.less 把这里画成盖在封面上的白色毛玻璃（无层级规则），夜间要 ! 才换得掉 -->
        <div class="right w-1/2 relative flex flex-col p-4 pt-[0.3rem] max-[768px]:dark:bg-black/50!">
          <div class="top w-full flex justify-end text-[0.75rem] text-[#606266] dark:text-[#aaa]">
            <span class="date ml-[1.45rem]">
              <Icon iconName="icon-calendar" iconSize="1.35"></Icon>
              {{ article.createTime?.slice(0, 10) }}
            </span>
            <span class="num-word ml-[1.45rem]">
              <Icon iconName="icon-image-text" iconSize="1.35"></Icon>
              {{ article.wordNum }} 字
            </span>
            <span class="time-consum ml-[1.45rem]">
              <Icon iconName="icon-browse" iconSize="1.35"></Icon>
              {{ article.views }} 次
            </span>
            <span v-if="article.authorType === 'agent'" class="ml-[1.45rem]">
              <el-tag size="small" type="warning" effect="plain">AI · {{ article.authorName }}</el-tag>
            </span>
          </div>
          <!-- 字号写在被测量的元素上，v-clamp 才能按真实行高截断 -->
          <p v-clamp="{ text: article.title, lines: 2 }"
            class="title mt-[26px] mb-4 w-full min-w-0 font-bold text-[1.5rem] text-[rgb(233,84,107)] [overflow-wrap:anywhere]"
            :title="article.title" :aria-label="article.title"></p>
          <div v-if="article.isLocked && !article.canAccess"
            class="private-summary w-full relative flex items-center gap-[0.875rem] min-h-[82px] mt-[0.2rem] mb-4 py-[0.8rem] px-4 overflow-hidden border border-dashed border-color-pink-a3 rounded-xl bg-[linear-gradient(135deg,var(--color-red-a1),rgba(236,140,105,0.08))]"
            role="note" aria-label="私密文章提示">
            <span class="private-lock inline-flex [flex:0_0_2.65rem] items-center justify-center w-[2.65rem] h-[2.65rem] text-white rounded-[50%] bg-[linear-gradient(135deg,var(--color-pink),var(--color-orange))] shadow-[0_6px_16px_rgba(233,84,107,0.22)]" aria-hidden="true">
              <el-icon><Lock /></el-icon>
            </span>
            <span class="private-copy flex flex-col min-w-0 text-grey-7 dark:text-[#d4d4d4] leading-[1.35]">
              <span class="private-label mb-[0.15rem] text-color-red text-[0.625rem] font-bold tracking-[0.14em]">PRIVATE ENTRY</span>
              <strong class="text-[0.95rem]">这篇文章已上锁</strong>
              <span class="private-hint mt-[0.2rem] text-grey-6 dark:text-[#aaa] text-[0.75rem]">正文需使用密钥解锁后阅读</span>
            </span>
          </div>
          <!-- 窄屏卡片高度固定，摘要少留一行，避免和两行标题一起被封面裁掉 -->
          <p v-else v-clamp="{ text: article.summary ?? '', lines: summaryLineCount }"
            class="text w-full min-w-0 text-[0.875rem] leading-[2] [overflow-wrap:anywhere]"
            :title="article.summary" :aria-label="article.summary || undefined"></p>
          <div class="bottom w-full flex items-end justify-end mt-auto">
            <span class="more [flex:0_0_6rem] w-24 h-[2.625rem] leading-[2.625rem] text-center text-[rgba(255,255,255,0.6)] [border-radius:1rem_0] bg-[linear-gradient(to_right,var(--color-pink)_0,var(--color-orange)_100%)] [transition:all_0.5s]">{{ article.isLocked && !article.canAccess ? '解锁...' : 'more...' }}</span>
          </div>
        </div>
      </div>
    </router-link>
  </article>
</template>

<script lang="ts" setup>
import { Article } from '@/types/Article.ts'
import { Tag } from '@/types/Tag'
import { getArticleBg } from '@/utils/tool'
import { vClamp } from '@/directives/clamp'
import { onMounted, onUnmounted, ref } from 'vue'
const props = defineProps(['article'])
const article: Article<Tag> = props.article

const NARROW_SCREEN = '(max-width: 768px)'
const narrowScreenQuery = window.matchMedia(NARROW_SCREEN)
const summaryLineCount = ref(narrowScreenQuery.matches ? 2 : 3)

const syncSummaryLineCount = () => {
  summaryLineCount.value = narrowScreenQuery.matches ? 2 : 3
}

onMounted(() => narrowScreenQuery.addEventListener('change', syncSummaryLineCount))
onUnmounted(() => narrowScreenQuery.removeEventListener('change', syncSummaryLineCount))
</script>

<style lang="less" scoped>


/* 祖先 :hover + 后代；原带 !important（特异性低于原 .cover .right .bottom .more），保留以保证行为不变 */
.cover:hover .more {
  color: #fff !important;
}

.cover:hover .left img {
  transform: scale(1.05);
}

/* Icon 渲染成 <svg class="icon">，类名由子组件决定 */
.cover .right .top span .icon {
  transform: translateY(-1px);
}

/* 伪元素装饰 */
.cover .right .private-summary::after {
  position: absolute;
  right: -1.3rem;
  bottom: -2.2rem;
  width: 5.5rem;
  height: 5.5rem;
  content: '';
  border: 1px solid rgba(233, 84, 107, .12);
  border-radius: 50%;
}

/* EP 内部元素 */
.cover .right .private-summary .private-lock :deep(.el-icon) {
  font-size: 1.2rem;
}

@media screen and (max-width: 768px) {
  .cover {
    height: 244px;
    min-height: 244px;

    .private-summary {
      .private-hint {
        display: none;
      }
    }
  }
}
</style>
