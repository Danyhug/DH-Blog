<template>
  <div class="flex flex-1 flex-col justify-between">
    <!-- 文章浏览页。必须是单根元素且根部不能有注释：dev 模式会保留根部注释，组件变成片段，
         HomeView 的 out-in 过渡就收不到离开完成的回调，首页再也渲染不出来。
         flex 布局沿用父容器的 justify-between，让评论区在短文时仍贴底 -->
    <div>
      <Teleport to="body" :disabled="!store.aritcleModel.isFullPreview">
        <section id="article-reading" ref="readingContainer" tabindex="-1" aria-label="文章阅读区"
          :class="[store.aritcleModel.isFullPreview ? 'fixed inset-0 z-[2000] overflow-y-auto' : '',
            store.aritcleModel.isDarkMode ? 'bg-[#191919] text-[#d4d4d4]' : 'bg-white text-[#333]']">
          <div v-if="store.aritcleModel.isFullPreview" class="sticky top-0 z-10 border-b px-4 py-2"
            :class="store.aritcleModel.isDarkMode ? 'bg-[#191919] border-[#303030]' : 'bg-white border-grey-3'">
            <ArticleReadingTools class="mx-auto max-w-[320px]" />
          </div>
          <ArticleReadingTools v-else class="mx-4 mb-2 [@media(min-width:1025px)]:hidden" />
          <div id="article-body" class="blog-container px-6 pt-0 pb-5 [&_.md-editor-preview]:text-[length:var(--article-font-size)]! [&_.md-editor-preview_pre_code]:text-[length:var(--article-code-font-size)]!"
            :class="{ 'mx-auto max-w-[960px] px-4! sm:px-6!': store.aritcleModel.isFullPreview }"
            :style="{ '--article-font-size': `${store.aritcleModel.fontSize}px`, '--article-code-font-size': `${store.aritcleModel.fontSize - 1}px` }"
            @click="openPreviewLinkInNewTab">
            <h1 v-if="store.aritcleModel.isFullPreview" class="pt-6 pb-4 text-[1.6rem] font-bold! text-center">{{ title }}</h1>
            <MdPreview :editorId="system.mdEditorInit.editorId" :modelValue="content"
              :previewTheme="system.mdEditorInit.previewTheme" :codeFoldable="system.mdEditorInit.codeFoldable"
              :theme="store.aritcleModel.isDarkMode ? 'dark' : 'light'"
              :class="store.aritcleModel.isDarkMode ? '[--md-bk-color:#191919]! [--md-color:#d4d4d4]!' : ''" />
          </div>
        </section>
      </Teleport>
      <div class="info py-[10px] px-0 text-[12px] text-[#606266] [.article-night_&]:text-[#aaa] text-right border-t border-grey-4 [.article-night_&]:border-[#303030]">
        <span class="mx-[10px]">
          更新于 {{ update }}
        </span>
        <span class="mx-[10px]">
          阅读次数 {{ viewnum }} 次
        </span>
        <el-tag v-if="authorType === 'agent'" size="small" type="warning" effect="plain" class="ml-2">
          AI · {{ authorName }}
        </el-tag>
      </div>
    </div>
    <div class="comment" :style="{ display: store.aritcleModel.isFullPreview ? 'none' : '' }">
      <Comment />
    </div>
  </div>
</template>

<script lang="ts">
import { getArticleInfo } from '@/api/user';
import { Article } from '@/types/Article.ts'
import { Tag } from 'element-plus';
import { useUserStore, useSystemStore } from '@/store';
import { getArticleBg, formatDate } from '@/utils/tool';
import { nextTick, watch } from 'vue';
import router from '@/router';
import ArticleReadingTools from '@/components/frontend/ArticleReadingTools.vue';

export default {
  name: 'HomeView',
  components: { ArticleReadingTools },
  data() {
    return {
      // 文章信息
      id: -1,
      title: '',
      content: ``,
      created: '',
      update: '',
      viewnum: 0,
      authorType: '',
      authorName: '',
      store: useUserStore(),
      system: useSystemStore(),
      savedScrollY: 0,
      savedBodyOverflow: '',
      savedFocus: null as HTMLElement | null,
    }
  },
  mounted() {
    // Teleport avoids transformed ancestors constraining the fixed reading pane.
    watch(() => this.store.aritcleModel.isFullPreview, async (val) => {
      if (val) {
        this.savedScrollY = window.scrollY;
        this.savedBodyOverflow = document.body.style.overflow;
        this.savedFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
        const offset = Math.max(0, -(document.getElementById('article-body')?.getBoundingClientRect().top || 0));
        document.body.style.overflow = 'hidden';
        await nextTick();
        const container = this.$refs.readingContainer as HTMLElement;
        if (!container) return;
        container.scrollTop = offset;
        container.focus({ preventScroll: true });
      } else {
        document.body.style.overflow = this.savedBodyOverflow;
        await nextTick();
        if (!this.$refs.readingContainer) return;
        window.scrollTo({ top: this.savedScrollY, behavior: 'instant' });
        this.savedFocus?.focus({ preventScroll: true });
      }
    })
    window.addEventListener('keydown', this.onReadingKeydown);

    let unlockData = localStorage.getItem('unlockArticle')
    if (unlockData) {
      const data = JSON.parse(unlockData) as Article<any>
      this.changeArticleInfo(data)
      localStorage.removeItem('unlockArticle')
      return
    }

    getArticleInfo(this.$route.params.id as string).then((res: Article<Tag>) => {
      this.changeArticleInfo(res)
    }).catch(err => {
      const message = err?.response?.data?.msg || err?.message || ''
      if (message.indexOf('输入密码') !== -1) {
        router.replace({ name: 'Lock', query: { id: this.$route.params.id } })
      }
    })
  },
  methods: {
    onReadingKeydown(event: KeyboardEvent) {
      if (event.key === 'Escape' && !event.defaultPrevented) {
        this.store.aritcleModel.isFullPreview = false;
      }
    },
    // 正文里的 <a> 没有 target，单击会直接替换掉当前文章页；
    // 改为补上 target=_blank 后交给浏览器默认行为，在新标签页打开。
    // #锚点（页内跳转）与 mailto: 等非 http(s) 链接保持原行为，修饰键/中键点击也不干涉。
    openPreviewLinkInNewTab(e: MouseEvent) {
      if (e.defaultPrevented) return
      const target = e.target
      if (!(target instanceof Element)) return
      const anchor = target.closest('a')
      if (!(anchor instanceof HTMLAnchorElement)) return

      const rawHref = anchor.getAttribute('href') || ''
      if (rawHref.startsWith('#')) return
      if (anchor.protocol !== 'http:' && anchor.protocol !== 'https:') return
      if (e.button !== 0 || e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) return

      anchor.target = '_blank'
      anchor.rel = 'noopener noreferrer'
    },
    changeArticleInfo(article: Article<Tag>) {
      this.id = article.id || 0
      this.title = article.title || ''
      this.content = article.content || ''
      this.created = article.createTime ? formatDate(article.createTime) : ''
      this.update = article.updateTime ? formatDate(article.updateTime) : ''
      this.viewnum = article.views || 0
      this.authorType = article.authorType || ''
      this.authorName = article.authorName || ''

      // 更改pinia内容
      this.store.homeHeaderInfo = {
        id: this.id,
        title: this.title,
        created: this.created,
        wordNum: article.wordNum || 0,
        timConSum: article.wordNum ? (article.wordNum / 400 + 0.5).toFixed(0) : '0',
        thumbnailUrl: getArticleBg(article.thumbnailUrl, article.id),
        tags: article.tags
      }
    }
  },
  // 卸载组件时
  beforeUnmount() {
    window.removeEventListener('keydown', this.onReadingKeydown);
    if (this.store.aritcleModel.isFullPreview) document.body.style.overflow = this.savedBodyOverflow;
    this.store.aritcleModel.isFullPreview = false;
  }
}
</script>

<style lang="less" scoped>
/* 类名必须保留：style.less 在 1024 断点用它作全局钩子，且是下方 :deep() 的锚点 */
.blog-container {
  /* 正文由 md-editor 运行时生成，加不上 class */
  :deep(.md-editor-preview) {
    font-family: 'Microsoft YaHei';
  }

  :deep(.md-editor-preview .md-editor-code pre code) {
    font-size: 15px;
  }

  :deep(.hljs-comment) {
    font-style: normal;
  }
}

/** 平板移动端适配 */
@media screen and (max-width: 1024px) {
  .comment {
    :deep(.author-info) {
      grid-template-columns: repeat(1, 1fr);
      text-align: center;
      border-bottom: 1px solid #ccc;
      div {
        margin-right: 0!important;
      }
    }
  }
}
</style>
