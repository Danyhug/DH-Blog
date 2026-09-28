<template>
  <article class="type-img-left">
    <router-link :to="'./article/' + article.id">
      <div class="cover flex w-full min-h-[244px] rounded-2xl bg-grey-0 overflow-hidden border border-[#eee]">
        <div class="left w-1/2 relative overflow-hidden [clip-path:polygon(0_0,92%_0%,100%_100%,0%_100%)]">
          <img :src="getArticleBg(article.thumbnailUrl, article.id)" :alt="article.title" loading="lazy"
            class="block w-full h-full object-cover [transition:all_0.5s]">
        </div>
        <div class="right w-1/2 relative flex flex-col p-4 pt-[0.3rem]">
          <div class="top w-full flex justify-end text-[0.75rem] text-[#606266]">
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
          <p class="title mt-[26px] mb-4 font-bold">
            <span class="title-link text-[1.5rem] text-[rgb(233,84,107)]">{{ article.title }}</span>
          </p>
          <div v-if="article.isLocked && !article.canAccess"
            class="private-summary w-full relative flex items-center gap-[0.875rem] min-h-[82px] mt-[0.2rem] mb-4 py-[0.8rem] px-4 overflow-hidden border border-dashed border-color-pink-a3 rounded-xl bg-[linear-gradient(135deg,var(--color-red-a1),rgba(236,140,105,0.08))]"
            role="note" aria-label="私密文章提示">
            <span class="private-lock inline-flex [flex:0_0_2.65rem] items-center justify-center w-[2.65rem] h-[2.65rem] text-white rounded-[50%] bg-[linear-gradient(135deg,var(--color-pink),var(--color-orange))] shadow-[0_6px_16px_rgba(233,84,107,0.22)]" aria-hidden="true">
              <el-icon><Lock /></el-icon>
            </span>
            <span class="private-copy flex flex-col min-w-0 text-grey-7 leading-[1.35]">
              <span class="private-label mb-[0.15rem] text-color-red text-[0.625rem] font-bold tracking-[0.14em]">PRIVATE ENTRY</span>
              <strong class="text-[0.95rem]">这篇文章已上锁</strong>
              <span class="private-hint mt-[0.2rem] text-grey-6 text-[0.75rem]">正文需使用密钥解锁后阅读</span>
            </span>
          </div>
          <p v-else class="text text-[0.875rem] leading-[2] max-h-[128px] [display:-webkit-box] [-webkit-box-orient:vertical] [line-clamp:3] text-ellipsis overflow-hidden">{{ article.summary }}</p>
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
const props = defineProps(['article'])
const article: Article<Tag> = props.article
</script>

<style lang="less" scoped>
/*
  组件自身的布局/配色已内联。这里只留四类工具类表达不了的：

  1. 祖先 :hover + 后代选择器。`.cover:hover .more` 原本带 !important，
     是因为它的特异性(0,3,0)低于原 `.cover .right .bottom .more`(0,4,0)；
     现在基色已内联成工具类，保留 !important 可保证行为逐字节不变。
  2. 子组件的根元素：Icon 渲染成 `<svg class="icon">`，类名由子组件决定。
  3. 伪元素装饰 `.private-summary::after`。
  4. Element Plus 内部 `:deep(.el-icon)`。

  另外原来的 `.cover div { width: 50% }` 与 `.cover .right div { width: 100% }`
  是后代选择器：前者命中 `.left`/`.right`（各 w-1/2），后者命中 `.right` 内的
  `.top`/`.private-summary`/`.bottom`（各 w-full），已按此逐个内联。
*/
.cover:hover .more {
  color: #fff !important;
}

.cover:hover .left img {
  transform: scale(1.05);
}

.cover .right .top span .icon {
  transform: translateY(-1px);
}

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

.cover .right .private-summary .private-lock :deep(.el-icon) {
  font-size: 1.2rem;
}

@media screen and (max-width: 768px) {
  .cover {
    height: 244px;
    min-height: 244px;

    /*
      `.private-summary` 原本在这里还有 min-height: auto / margin-bottom: 0 / padding: .65rem，
      但它们是**死声明**：基础规则 `.cover .right .private-summary` 特异性更高(0,3,0 vs 0,2,0)，
      一直压着它们（实测 ≤768px 仍是 82px / 16px / .8rem 1rem）。
      基础值现已内联成工具类，若留下这几条，无层级声明会反过来生效而改变观感，故删除。
      `.private-hint { display: none }` 没有冲突（基础规则只设 margin-top/color/font-size），
      原样保留。
    */
    .private-summary {
      .private-hint {
        display: none;
      }
    }
  }
}
</style>
