<template>
  <el-table :data="props.categories">
    <el-table-column prop="id" label="编号" />
    <el-table-column prop="name" label="名称" />
    <el-table-column prop="slug" label="编码名称" />
    <el-table-column prop="createTime" label="发布时间" />
    <el-table-column prop="updateTime" label="更新时间" />
    <el-table-column fixed="right" label="操作">
      <template #header>
        <el-button type="success" round @click="add">新增分类</el-button>
      </template>
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row)">编辑</el-button>
        <el-button link type="danger" size="small" @click.prevent="del(scope.row.id)">删除</el-button>
      </template>
    </el-table-column>
  </el-table>

  <TableDialog :visible="visible" :state="state" :data="category" @close="visible = false" @add="confirmAdd"
    @update="update" @cancel="cancel" />
</template>
<script lang="ts" setup>
import { addCategory, deleteCategory, updateCategory } from '@/api/api';
import { useAdminStore } from '@/store';
import { Category } from '@/types/Category';
import { reactive, ref } from 'vue';
import TableDialog from '@/components/backend/Table/TableDialog.vue'

const store = useAdminStore()
const state = ref('add')
const visible = ref(false)
const props = defineProps(['categories'])
const category = reactive<Category>({
  name: '',
  slug: ''
})

// 新增
const add = () => {
  visible.value = true
  state.value = 'add'
  Object.assign(category, { name: '', slug: '' })
}

const confirmAdd = () => {
  addCategory(category).then(() => {
    visible.value = false
    ElMessage.success('新增分类成功')
    store.getCategories()
  })
}

// 编辑
const edit = (row: Category) => {
  state.value = 'edit'
  Object.assign(category, row)
  visible.value = true
}

const update = () => {
  updateCategory(category).then(() => {
    ElMessage.success('修改分类成功')
  })
  visible.value = false
  store.getCategories()
}

// 删除
const del = (id: String) => {
  deleteCategory(id).then(() => {
    ElMessage.success('删除分类成功')
    store.getCategories()
  })
}

const cancel = () => visible.value = false
</script>
