<template>
  <!-- 文章浏览页 -->
  <div>
    <!-- 全屏观看文章信息 -->
    <div :class="`blog-container ${store.aritcleModel.isFullPreview ? 'full-screen-preview' : ''}`">
      <p class="title" v-show="store.aritcleModel.isFullPreview" @click="changeIsFullPreview()">{{ title }}</p>
      <MdPreview :editorId="state.id" :modelValue="content" previewTheme="cyanosis" codeFoldable="false"
        :theme="state.theme" :scrollElement="scrollElement" />
    </div>
    <div class="info">
      <span>
        更新于 {{ created }}
      </span>
      <span>
        阅读次数 {{ viewnum }} 次
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { getArticleInfo } from '@/api/api';
import { Article } from '@/types/Article.ts'
import { Tag } from 'element-plus';
import { useUserStore } from '@/store';
import { getArticleBg } from '@/utils/tool';
import { watch } from 'vue';

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
      scrollElement: document.documentElement,
      state: {
        theme: 'light',
        id: 'dh-editor'
      }
    }
  },
  created() {
    const store = this.store

    // 使用ua检测是否是移动端访问，是移动端直接变为全屏浏览
    const width = window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;
    if (width <= 768) {
      store.aritcleModel.isFullPreview = true;
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


    getArticleInfo(this.$route.params.id as string).then((res: Article<Tag>) => {
      this.id = res.id || 0
      this.title = res.title || ''
      this.content = res.content || ''
      this.created = res.createTime || ''
      this.update = res.updateTime || ''
      this.viewnum = res.views || 0

      // 更改pinia内容
      this.store.homeHeaderInfo = {
        title: this.title,
        created: this.created,
        wordNum: res.wordNum || 0,
        timConSum: res.wordNum ? (res.wordNum / 400 + 0.5).toFixed(0) : '0',
        thumbnailUrl: getArticleBg(res.thumbnailUrl),
        tags: res.tags
      }
    })
  },
  methods: {
    changeIsFullPreview() {
      this.store.aritcleModel.isFullPreview = !this.store.aritcleModel.isFullPreview
    }
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
  ;

  :deep(.md-editor-preview) {
    padding: 0 10px;
    background-color: rgb(250, 250, 250);
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
</style>
