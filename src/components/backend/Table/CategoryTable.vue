<template>
  <el-table :data="props.categories">
    <el-table-column prop="id" label="编号" />
    <el-table-column prop="name" label="名称" />
    <el-table-column prop="slug" label="编码名称" />
    <el-table-column prop="createdAt" label="发布时间" />
    <el-table-column prop="updatedAt" label="更新时间" />
    <el-table-column fixed="right" label="操作">
      <template #header>
        <el-button type="success" round @click="add">新增分类</el-button>
      </template>
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row.id)">编辑</el-button>
        <el-button link type="danger" size="small">删除</el-button>
      </template>
    </el-table-column>
  </el-table>

  <el-dialog v-model="visible" title="新增" :show-close="false" width="300">
    <el-form :model="category" size="small" label-position="top">
      <el-form-item label="名称">
        <el-input v-model="category.name" />
      </el-form-item>
      <el-form-item label="别名(SEO优化)">
        <el-input v-model="category.slug" />
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
import { addCategory } from '@/api/api';
import { useAdminStore } from '@/store';
import { Category } from '@/types/Category';
import { reactive, ref } from 'vue';

const store = useAdminStore()
const visible = ref(false)
const props = defineProps(['categories'])
const category = reactive<Category>({
  name: '',
  slug: ''
})

// 新增
const add = () => {
  visible.value = true

  addCategory(category).then(() => {
    visible.value = false
    ElMessage.success('新增分类成功')
    store.getCategories()
  })
}

// 编辑
const edit = (id: number) => {
  ElMessage.success('修改分类成功')
}
</script>