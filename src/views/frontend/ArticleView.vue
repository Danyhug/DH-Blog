<template>
  <!-- 文章浏览页 -->
  <div>
    <div class="blog-container">
      <MdPreview :modelValue="content" previewTheme="github" />
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
      viewnum: 0
    }
  },
  mounted() {
    const store = useUserStore()

    getArticleInfo(this.$route.params.id as string).then((res: Article<Tag>) => {
      this.id = res.id || 0
      this.title = res.title || ''
      this.content = res.content || ''
      this.created = res.createTime || ''
      this.update = res.updateTime || ''
      this.viewnum = res.views || 0

      // 更改pinia内容
      store.homeHeaderInfo = {
        title: this.title,
        created: this.created,
        wordNum: res.wordNum || 0,
        timConSum: res.wordNum ? (res.wordNum / 200 + 0.5).toFixed(0) : '0',
        thumbnailUrl: getArticleBg(res.thumbnailUrl),
        tags: res.tags
      }
    })
  },
  methods: {

  }
}
</script>

<style lang="less" scoped>
.blog-container {
  padding: 1.25rem 1.5rem;
  padding-top: 0;
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
