<template>
  <el-table :data="props.tags">
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
import { reactive, ref } from 'vue'
import { useAdminStore } from '@/store/index'
import { ElMessage } from 'element-plus'

const props = defineProps(['tags'])
const visible = ref(false)
const state = ref('add')
const store = useAdminStore()

const tag = reactive<Tag>({
  name: '',
})

// 新增
const add = () => {
  visible.value = true
  state.value = 'add'
  Object.assign(tag, { name: '' })
}

const confirmAdd = () => {
  addTag(tag).then(() => {
    visible.value = false
    ElMessage.success('新增标签成功')
    store.getTags()
  })
}

// 编辑
const edit = (row: Tag) => {
  state.value = 'edit'
  Object.assign(tag, row)
  visible.value = true
}

const update = () => {
  updateTag(tag).then(() => {
    ElMessage.success('修改标签成功')
  })
  visible.value = false
  store.getTags()
}

// 删除
const del = (id: String) => {
  deleteTag(id).then(() => {
    ElMessage.success('删除标签成功')
    store.getTags()
  })
}

const cancel = () => visible.value = false
</script>