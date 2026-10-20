<template>
  <el-table :data="pagedCategories">
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

  <div v-if="props.categories.length > PAGE_SIZE" class="mt-3 flex justify-end">
    <el-pagination v-model:current-page="page" :page-size="PAGE_SIZE" :total="props.categories.length"
      layout="total, prev, pager, next" />
  </div>

  <TableDialog :visible="visible" :state="state" :data="category" @close="visible = false" @add="confirmAdd"
    @update="update" @cancel="cancel">
    <el-form-item label="绑定标签">
      <el-checkbox v-model="bindTags" :label="tag.name" :value="tag.id" :key="tag.id" v-for="tag in tags" />
    </el-form-item>
  </TableDialog>
</template>

<script lang="ts" setup>
import { addCategory, deleteCategory, getTagListByCategoryId, updateCategory } from '@/api/admin';
import { useAdminStore } from '@/store';
import { Category } from '@/types/Category';
import { computed, reactive, ref, watch } from 'vue';
import TableDialog from '@/components/backend/Table/TableDialog.vue'
import { notify } from '@/utils/notification'

// 绑定的标签
const bindTags = ref<number[]>([])

const store = useAdminStore()
const state = ref('add')
const visible = ref(false)
const props = defineProps(['categories', 'tags'])
const category = reactive<Category>({
  name: '',
  slug: ''
})

// 分类总量是几十条的量级，分页故意留在前端：store.categories 同时喂给发布页的分类单选，
// 改成服务端分页会让那边的下拉框只剩当前页
const PAGE_SIZE = 10
const page = ref(1)
const pagedCategories = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return props.categories.slice(start, start + PAGE_SIZE)
})

// 删到当前页空了（或新增后跳了页）就把页码收回有效范围，否则会停在空白页
watch(() => props.categories.length, (length) => {
  const lastPage = Math.max(1, Math.ceil(length / PAGE_SIZE))
  if (page.value > lastPage) page.value = lastPage
})

// 新增
const add = () => {
  visible.value = true
  state.value = 'add'
  Object.assign(category, { name: '', slug: '' })
}

const confirmAdd = () => {
  category.tagIds = bindTags.value
  addCategory(category).then(() => {
    visible.value = false
    notify.success('新增分类成功')
    store.getCategories()
  })
}

// 编辑
const edit = (row: Category) => {
  state.value = 'edit'
  Object.assign(category, row)
  visible.value = true

  if (category.id) {
    getTagListByCategoryId(category.id).then(res => {
      bindTags.value = res.map(r => r.id!)
    })
  }
}

const update = () => {
  category.tagIds = bindTags.value
  updateCategory(category).then(() => {
    notify.success('修改分类成功')
  })
  visible.value = false
  store.getCategories()
}

// 删除
const del = (id: String) => {
  deleteCategory(id).then(() => {
    notify.success('删除分类成功')
    store.getCategories()
  })
}

const cancel = () => visible.value = false
</script>
