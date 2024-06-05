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

import { getArticleCategoryList, getArticleList, getArticleTagList } from '@/api/api';
import { Article } from '@/types/Article';
import { Category } from '@/types/Category';
import { Tag } from '@/types/Tag';
import { onMounted, reactive } from 'vue';
import { useRouter } from 'vue-router'

const router = useRouter()

const articles = reactive<Article<Tag>[]>([])
const categories = reactive<Category[]>([]);
const tags = reactive<Tag[]>([]);

const edit = (id: number) => {
  router.push({ name: 'publish', query: { articleId: id } });
}

// 获取分类列表
const getCategories = async () => {
  const data = await getArticleCategoryList();
  categories.push(...data);
};

// 获取标签列表
const getTags = async () => {
  const data = await getArticleTagList();
  tags.push(...data);
};

onMounted(() => {
  getCategories();
  getTags();

  getArticleList({ pageNum: 2, pageSize: 10 }).then((res) => {
    let articleList: Article<Tag>[] = [];
    res.list.forEach(item => {
      item.categoryName = categories.find(c => c.id === item.categoryId)?.name
      articleList.push({ ...item })
    })
    articles.push(...articleList);
    console.log(articles)
  })
})
</script>
<style scoped>
.box-card {
  width: 460px;
}
</style>