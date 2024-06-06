<template>
  <el-dialog v-model="visible" title="新增" :show-close="false" width="300">
    <el-form :model="data" size="small" label-position="top">
      <el-form-item label="名称">
        <el-input v-model="data.name" />
      </el-form-item>
      <el-form-item label="别名(SEO优化)">
        <el-input v-model="data.slug" />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="cancel">取消</el-button>
        <el-button type="primary" @click="add" v-show="state == 'add'">新增</el-button>
        <el-button type="primary" @click="update" v-show="state == 'edit'">修改</el-button>
      </div>
    </template>
  </el-dialog>
</template>
<script lang="ts" setup>
import { defineProps, toRefs } from 'vue'
const porps = defineProps(['visible', 'data', 'state'])
const { visible, data, state } = toRefs(porps)
// 触发父元素的update方法
const emit = defineEmits(['update', 'add', 'cancel'])
const update = () => emit('update')
const add = () => emit('add')
const cancel = () => emit('cancel')
</script>