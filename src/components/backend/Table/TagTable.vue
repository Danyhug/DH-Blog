<template>
  <el-table :data="props.tags">
    <el-table-column prop="id" label="编号" width="90" />
    <el-table-column prop="name" label="名称" width="90" />
    <el-table-column prop="slug" label="编码名称" width="90" />
    <el-table-column prop="createdAt" label="发布时间" width="170" />
    <el-table-column prop="updatedAt" label="更新时间" width="170" />
  </el-table>
</template>
<script lang="ts" setup>
const props = defineProps(['tags'])
</script>