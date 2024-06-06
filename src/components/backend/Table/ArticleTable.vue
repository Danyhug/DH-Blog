<template>
  <el-table :data="props.articles" style="width: 100%;">
    <el-table-column prop="id" label="编号" width="90" />
    <el-table-column prop="title" label="文章标题" width="180" />
    <el-table-column prop="categoryName" label="分类">
      <template #default="scope">
        <el-tag type="primary">{{ scope.row.categoryName }}</el-tag>
      </template>
    </el-table-column>

    <el-table-column prop="tags" label="标签">
      <template #default="scope">
        <el-tag type="primary" v-for="tag in scope.row.tags" effect="plain" round>{{ tag.name }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="views" label="浏览数" />
    <el-table-column prop="wordNum" label="字数" />
    <el-table-column prop="publishDate" label="发布时间" width="170" />
    <el-table-column prop="updateDate" label="更新时间" width="170" />
    <el-table-column fixed="right" label="操作" width="100">
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row.id)">编辑</el-button>
        <el-button link type="danger" size="small">删除</el-button>
      </template>
    </el-table-column>
  </el-table>
</template>
<script lang="ts" setup>
import { useRouter } from 'vue-router'

const props = defineProps(['articles'])
const router = useRouter()

const edit = (id: number) => {
  router.push({ name: 'publish', query: { articleId: id } });
}

</script>