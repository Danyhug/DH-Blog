<template>
  <!-- 文章浏览页 -->
  <div>
    <!-- 全屏观看文章信息 -->
    <div :class="['blog-container', store.aritcleModel.isFullPreview
      ? 'full-screen-preview fixed top-0 left-0 w-full h-full overflow-y-auto p-0 bg-white'
      : 'px-6 pt-0 pb-5']"
      @click="openPreviewLinkInNewTab">
      <p class="title pt-[1.875rem] px-0 pb-4 text-[1.6rem] font-bold text-center cursor-pointer [font-family:宋体]" v-show="store.aritcleModel.isFullPreview" @click="changeIsFullPreview()">{{ title }}</p>
      <MdPreview :editorId="system.mdEditorInit.editorId" :modelValue="content"
        :previewTheme="system.mdEditorInit.previewTheme" :codeFoldable="system.mdEditorInit.codeFoldable"
        :theme="system.mdEditorInit.theme" :scrollElement="scrollElement" />
    </div>
    <div class="info py-[10px] px-0 text-[12px] text-[#606266] text-right border-t border-grey-4">
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
</template>

<script lang="ts">
import { getArticleInfo } from '@/api/user';
import { Article } from '@/types/Article.ts'
import { Tag } from 'element-plus';
import { useUserStore, useSystemStore } from '@/store';
import { getArticleBg, formatDate } from '@/utils/tool';
import { watch } from 'vue';
import router from '@/router';

export default {
  name: 'HomeView',
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
      scrollElement: document.documentElement,
    }
  },
  mounted() {
    // 监听是否全屏浏览状态
    watch(() => this.store.aritcleModel.isFullPreview, (val) => {
      if (val) {
        document.body.style.overflow = 'hidden';
      } else {
        document.body.style.overflow = '';
      }
    })

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
    changeIsFullPreview() {
      this.store.aritcleModel.isFullPreview = !this.store.aritcleModel.isFullPreview
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
    document.body.style.overflow = '';
  }
}
</script>

<style lang="less" scoped>
/*
  本文件剩下的几乎都是 `:deep()`：正文由 md-editor（markdown-it + highlight.js）
  在运行时生成，DOM 上加不上 class（§4.3.1）。组件自身的 padding/标题/信息条
  已内联到模板。

  `.blog-container` 这个类名必须保留：style.less 在 1024 断点下用
  `.blog-container { padding: 0 !important }` 覆盖它，属于全局钩子；
  它同时是下面三条 `:deep()` 的锚点。
*/
.blog-container {
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

/*
  `.full-screen-preview` 同样要保留类名（全屏态由 JS 切换，且是下面两条 :deep() 的锚点）；
  它自身那套定位/尺寸已内联成互斥的 class 串（原 `.blog-container` 与 `.full-screen-preview`
  都写 padding，且特异性相同、后者靠源码顺序取胜；内联后必须整体二选一，
  不能让 `p-0` 与 `px-*` 同时存在，否则谁赢取决于 Tailwind 的生成顺序，见 §4.9）。
*/
.full-screen-preview {
  :deep(.md-editor-preview) {
    padding: 0 10px;
    background-color: rgb(250, 250, 250);
    font-size: 17.5px;
    line-height: 2em;
  }

  :deep(.md-editor-preview .md-editor-code pre code) {
    font-size: 18px;
  }
}

/* 原 `.left` 规则在本文件模板里没有任何对应元素（从 HomeView 复制过来的死样式），已删除 */

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
