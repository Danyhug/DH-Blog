<template>
  <el-table :data="props.tags">
    <el-table-column prop="id" label="编号" />
    <el-table-column prop="name" label="名称" />
    <el-table-column prop="slug" label="编码名称" />
    <el-table-column prop="createdAt" label="发布时间" />
    <el-table-column prop="updatedAt" label="更新时间" />
    <el-table-column fixed="right" label="操作">
      <template #header>
        <el-button type="success" round>新增标签</el-button>
      </template>
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row.id)">编辑</el-button>
        <el-button link type="danger" size="small">删除</el-button>
      </template>
    </el-table-column>
  </el-table>

  <el-dialog v-model="visible" title="新增" :show-close="false" width="300">
    <el-form :model="tag" size="small" label-position="top">
      <el-form-item label="名称">
        <el-input v-model="tag.name" />
      </el-form-item>
      <el-form-item label="别名(SEO优化)">
        <el-input v-model="tag.slug" />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="add">
          新增
        </el-button>
      </div>
    </template>
  </el-dialog>

</template>
<script lang="ts" setup>
import { addTag } from '@/api/api';
import { Tag } from '@/types/Tag'
import { reactive, ref } from 'vue'
const props = defineProps(['tags'])
const visible = ref(true)

const tag = reactive<Tag>({
  name: '',
  slug: '',
})

// 新增标签
const add = () => {
  addTag(tag).then(() => {
    visible.value = false
    ElMessage.success('新增标签成功')
  })
}

</script>