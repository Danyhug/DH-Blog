<template>
  <nav v-if="totalPages > 1" class="pagination-shell" aria-label="文章列表分页">
    <el-pagination
      v-model:current-page="currentPageModel"
      class="article-pagination"
      layout="prev, pager, next"
      :page-size="props.pageSize"
      :pager-count="5"
      :total="props.total"
    />
  </nav>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

interface PaginationProps {
  currentPage: number
  pageSize: number
  total: number
}

const props = defineProps<PaginationProps>()
const emit = defineEmits<{
  'update:currentPage': [page: number]
}>()

const currentPageModel = computed({
  get: () => props.currentPage,
  set: (page: number) => emit('update:currentPage', page)
})

const totalPages = computed(() => Math.ceil(props.total / props.pageSize))
</script>

<style lang="less" scoped>
.pagination-shell {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 46px;
  padding: 4px 0;
}

:deep(.article-pagination.el-pagination) {
  --el-pagination-hover-color: var(--primary-color);
  flex-wrap: nowrap;
  padding: 0;
}

:deep(.article-pagination.el-pagination .btn-prev),
:deep(.article-pagination.el-pagination .btn-next),
:deep(.article-pagination.el-pagination .el-pager li) {
  min-width: 38px;
  height: 38px;
  margin: 0 3px;
  color: var(--grey-6);
  font-size: .8125rem;
  font-weight: 650;
  line-height: 38px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  box-shadow: none;
  transition: color .2s ease, border-color .2s ease, background-color .2s ease, transform .2s ease, box-shadow .2s ease;
}

:deep(.article-pagination.el-pagination .btn-prev:not(:disabled):hover),
:deep(.article-pagination.el-pagination .btn-next:not(:disabled):hover),
:deep(.article-pagination.el-pagination .el-pager li:not(.is-active):hover) {
  color: var(--primary-color);
  background: var(--color-red-a1);
  transform: translateY(-1px);
}

:deep(.article-pagination.el-pagination .el-pager li.is-active) {
  color: var(--grey-0);
  border-color: transparent;
  background: linear-gradient(135deg, var(--color-pink), var(--color-orange));
  box-shadow: 0 7px 16px rgba(233, 84, 107, .25);
  transform: translateY(-1px);
}

:deep(.article-pagination.el-pagination button:disabled) {
  color: var(--grey-4);
  background: transparent;
  box-shadow: none;
}

:deep(.article-pagination button:focus-visible),
:deep(.article-pagination .el-pager li:focus-visible) {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

@media (max-width: 600px) {
  .pagination-shell {
    min-height: 42px;
    padding: 2px 0;
  }

  :deep(.article-pagination.el-pagination .btn-prev),
  :deep(.article-pagination.el-pagination .btn-next),
  :deep(.article-pagination.el-pagination .el-pager li) {
    min-width: 34px;
    height: 34px;
    margin: 0 2px;
    line-height: 34px;
    border-radius: 10px;
  }
}

@media (prefers-reduced-motion: reduce) {
  :deep(.article-pagination.el-pagination .btn-prev),
  :deep(.article-pagination.el-pagination .btn-next),
  :deep(.article-pagination.el-pagination .el-pager li) {
    transition: none;
  }
}
</style>
