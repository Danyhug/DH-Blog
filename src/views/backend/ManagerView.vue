<template>
  <el-tabs v-model="activeName" type="border-card">
    <el-tab-pane label="文章管理" name="first">
      <ArticleTable :articles="articles"></ArticleTable>
    </el-tab-pane>

    <el-tab-pane label="分类管理" name="second">
      <CategoryTable :categories="categories"></CategoryTable>
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

onMounted(() => {
  store.getCategories();
  store.getTags();

  getArticleList({ pageNum: 1, pageSize: 10, total: 10 }).then((res) => {
    let articleList: Article<Tag>[] = [];
    res.list.forEach(item => {
      item.categoryName = categories.find(c => c.id === item.categoryId)?.name
      articleList.push({ ...item })
    })
    articles.push(...articleList);
  })
})
</script>
<style scoped>
.box-card {
  width: 460px;
}
</style>