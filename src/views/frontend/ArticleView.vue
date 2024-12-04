<template>
  <!-- 文章浏览页 -->
  <div>
    <!-- 全屏观看文章信息 -->
    <div :class="`blog-container ${store.aritcleModel.isFullPreview ? 'full-screen-preview' : ''}`">
      <p class="title" v-show="store.aritcleModel.isFullPreview" @click="changeIsFullPreview()">{{ title }}</p>
      <MdPreview :editorId="system.mdEditorInit.editorId" :modelValue="content"
        :previewTheme="system.mdEditorInit.previewTheme" :codeFoldable="system.mdEditorInit.codeFoldable"
        :theme="system.mdEditorInit.theme" :scrollElement="scrollElement" />
    </div>
    <div class="info">
      <span>
        更新于 {{ update }}
      </span>
      <span>
        阅读次数 {{ viewnum }} 次
      </span>
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
import { getArticleBg } from '@/utils/tool';
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
      if (err.message.indexOf('输入密码') != -1) {
        router.replace({ name: 'Lock', query: { id: this.$route.params.id } })
      }
    })
  },
  methods: {
    changeIsFullPreview() {
      this.store.aritcleModel.isFullPreview = !this.store.aritcleModel.isFullPreview
    },
    changeArticleInfo(article: Article<Tag>) {
      this.id = article.id || 0
      this.title = article.title || ''
      this.content = article.content || ''
      this.created = article.createTime || ''
      this.update = article.updateTime || ''
      this.viewnum = article.views || 0

      // 更改pinia内容
      this.store.homeHeaderInfo = {
        id: this.id,
        title: this.title,
        created: this.created,
        wordNum: article.wordNum || 0,
        timConSum: article.wordNum ? (article.wordNum / 400 + 0.5).toFixed(0) : '0',
        thumbnailUrl: getArticleBg(article.thumbnailUrl),
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
.blog-container {
  padding: 1.25rem 1.5rem;
  padding-top: 0;

  .title {
    font-size: 1.6rem;
    font-family: "宋体";
    font-weight: bold;
    text-align: center;
    padding: 1.875rem 0 1rem;
    cursor: pointer;
  }

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

.full-screen-preview {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  overflow-y: auto;
  padding: 0;
  background-color: #fff;

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

.left {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 30%;

  top: 0;
  margin: .6rem 0;
}

.info {
  padding: 10px 0;
  font-size: 12px;
  color: #606266;
  text-align: right;
  border-top: 1px solid var(--grey-4);

  span {
    margin: 0 10px;
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
