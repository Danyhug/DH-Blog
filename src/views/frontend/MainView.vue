<template>
  <div>
    <!-- 这里是文章列表 -->
    <div class="tip">
      <p>文章列表</p>
    </div>
    <div class="posts">
      <ArticleBox v-for="item in articleList" :article="item" :key="item.id"></ArticleBox>
    </div>
    <div class="page">
      <Pagination></Pagination>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { getArticleList } from '@/api/api';
import ArticleBox from '@/components/frontend/ArticleBox.vue'
import Pagination from '@/components/frontend/Pagination.vue';
import { onMounted, reactive } from 'vue'
import { Page } from '@/types/Page.ts'
import { Article } from '@/types/Article';
import { Tag } from '@/types/Tag';

const articleList = reactive<Article<Tag[]>[]>([])
const page = reactive<Page>({
  pageNum: 1,
  pageSize: 10
})

const getPageList = () => {
  getArticleList(page).then(res => {
    articleList.push(...res.list)
    console.log(articleList)
  })
}

onMounted(() => {
  getPageList()
})

</script>
<style lang="less" scoped>
.tip {
  display: block;
  width: 100%;
  text-align: center;
  font-weight: 700;

  p {
    display: inline-block;
    transform: translateY(-50%);
    background-color: #fff;
    padding: 0 16px;
    font-weight: 400;
    color: #606266;
    font-size: 24px;
  }
}

/* 文章父元素 */
.posts {
  padding: 18px;
}

Article {
  margin-bottom: 46px;
}

.page {
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>