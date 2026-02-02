<template>
  <el-tabs v-model="activeName" type="border-card">
    <el-tab-pane label="文章管理" name="first">
      <ArticleTable :articles="articles" @refresh="loadArticles"></ArticleTable>
    </el-tab-pane>

    <el-tab-pane label="分类管理" name="second">
      <CategoryTable :categories="categories" :tags="tags"></CategoryTable>
    </el-tab-pane>

    <el-tab-pane label="标签管理" name="third">
      <TagTable :tags="tags"></TagTable>
    </el-tab-pane>
  </el-tabs>
</template>

<script lang="ts" setup>
import ArticleTable from '@/components/backend/Table/ArticleTable.vue'
import CategoryTable from '@/components/backend/Table/CategoryTable.vue'
import TagTable from '@/components/backend/Table/TagTable.vue'

import { getArticleList } from '@/api/admin';
import { Article } from '@/types/Article';

import { Tag } from '@/types/Tag';
import { onMounted, reactive } from 'vue';
import { ref } from 'vue';
import { useAdminStore } from '@/store/';


const activeName = ref('first')
const store = useAdminStore()

const articles = reactive<Article<Tag>[]>([])
const categories = store.categories
const tags = store.tags

// 加载文章列表
const loadArticles = async () => {
  // 清空现有文章列表
  articles.splice(0, articles.length)
  
  // 重新获取文章列表
  const res = await getArticleList({ pageNum: 1, pageSize: 10, total: 10 })
  const articleList: Article<Tag>[] = []
  res.list.forEach(item => {
    item.categoryName = categories.find(c => c.id === item.categoryId)?.name
    articleList.push({ ...item })
  })
  articles.push(...articleList)
}

onMounted(async () => {
  await store.getCategories()
  await store.getTags()
  await loadArticles()
})
</script>
<style scoped>
.box-card {
  width: 460px;
}
</style>
