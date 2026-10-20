<template>
  <el-table :data="pagedTags">
    <el-table-column prop="id" label="编号" />
    <el-table-column prop="name" label="名称" />
    <el-table-column prop="createTime" label="发布时间" />
    <el-table-column prop="updateTime" label="更新时间" />
    <el-table-column fixed="right" label="操作">
      <template #header>
        <el-button type="success" round @click="add">新增标签</el-button>
      </template>
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row)">编辑</el-button>
        <el-button link type="danger" size="small" @click.prevent="del(scope.row.id)">删除</el-button>
      </template>
    </el-table-column>
  </el-table>

  <div v-if="props.tags.length > PAGE_SIZE" class="mt-3 flex justify-end">
    <el-pagination v-model:current-page="page" :page-size="PAGE_SIZE" :total="props.tags.length"
      layout="total, prev, pager, next" />
  </div>

  <el-dialog v-model="visible" title="标签管理" :show-close="false" width="300">
    <el-form :model="tag" size="small" label-position="top">
      <el-form-item label="标签名称">
        <el-input v-model="tag.name" />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="cancel">取消</el-button>
        <el-button type="primary" @click="confirmAdd" v-show="state == 'add'">新增</el-button>
        <el-button type="primary" @click="update" v-show="state == 'edit'">修改</el-button>
      </div>
    </template>
  </el-dialog>
</template>
<script lang="ts" setup>
import { addTag, updateTag, deleteTag } from '@/api/admin';
import { Tag } from '@/types/Tag'
import { computed, reactive, ref, watch } from 'vue'
import { useAdminStore } from '@/store/index'
import { notify } from '@/utils/notification'

const props = defineProps(['tags'])
const visible = ref(false)
const state = ref('add')
const store = useAdminStore()

const tag = reactive<Tag>({
  name: '',
})

// 和分类表一样，分页只在前端做：store.tags 还要给分类弹窗的「绑定标签」用，改服务端分页会牵连它
const PAGE_SIZE = 10
const page = ref(1)
const pagedTags = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return props.tags.slice(start, start + PAGE_SIZE)
})

watch(() => props.tags.length, (length) => {
  const lastPage = Math.max(1, Math.ceil(length / PAGE_SIZE))
  if (page.value > lastPage) page.value = lastPage
})

// 新增
const add = () => {
  visible.value = true
  state.value = 'add'
  delete tag.id
  Object.assign(tag, { name: '' })
}

const confirmAdd = () => {
  addTag(tag).then(() => {
    visible.value = false
    notify.success('新增标签成功')
    store.getTags()
  })
}

// 编辑
const edit = (row: Tag) => {
  state.value = 'edit'
  Object.assign(tag, row)
  visible.value = true
}

const update = async () => {
  await updateTag(tag)
  notify.success('修改标签成功')
  visible.value = false
  await store.getTags()
}

// 删除
const del = (id: number) => {
  deleteTag(id).then(() => {
    notify.success('删除标签成功')
    store.getTags()
  })
}

const cancel = () => visible.value = false
</script>
