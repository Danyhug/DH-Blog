<template>
  <el-table :data="props.categories">
    <el-table-column prop="id" label="编号"/>
    <el-table-column prop="name" label="名称" />
    <el-table-column prop="slug" label="编码名称" />
    <el-table-column prop="createdAt" label="发布时间" />
    <el-table-column prop="updatedAt" label="更新时间" />
    <el-table-column fixed="right" label="操作">
      <template #header>
        <el-button type="success" round>新增分类</el-button>
      </template>
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row.id)">编辑</el-button>
        <el-button link type="danger" size="small">删除</el-button>
      </template>
    </el-table-column>
  </el-table>
</template>
<script lang="ts" setup>
const props = defineProps(['categories'])
</script>