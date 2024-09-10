<template>
  <!-- 分页组件 -->
  <el-pagination background layout="prev, pager, next" :current-page="currPage" :page-size="props.pageSize"
    :page-count="props.total" @current-change="currChange($event)" />

  {{ props }}

</template>
<script lang="ts" setup>
import { defineProps, ref, defineEmits } from 'vue';

// 定义接收的 prop
const props = defineProps({
  currentPage: {
    type: Number,
    required: true
  },
  pageSize: {
    type: Number,
    required: true
  },
  total: {
    type: Number,
    required: true
  }
});
const emit = defineEmits(['update:currentPage']);

const currPage = ref<number>(props.currentPage)
const currChange = (page: number) => {
  currPage.value = page
  emit('update:currentPage', page);
}

</script>
