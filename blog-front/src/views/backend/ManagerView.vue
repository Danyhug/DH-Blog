<template>
  <el-tabs v-model="activeName" type="border-card">
    <el-tab-pane label="文章管理" name="first">
      <div class="flex justify-end mb-3">
        <el-button type="primary" :loading="batchRunning" @click="batchDialogVisible = true">
          <el-icon class="mr-1"><MagicStick /></el-icon>
          {{ batchRunning ? '摘要生成中' : '一键生成摘要' }}
        </el-button>
      </div>
      <ArticleTable :articles="articles" @refresh="loadArticles"></ArticleTable>
    </el-tab-pane>

    <el-tab-pane label="分类管理" name="second">
      <CategoryTable :categories="categories" :tags="tags"></CategoryTable>
    </el-tab-pane>

    <el-tab-pane label="标签管理" name="third">
      <TagTable :tags="tags"></TagTable>
    </el-tab-pane>
  </el-tabs>

  <el-dialog v-model="batchDialogVisible" title="一键生成文章摘要" width="430px">
    <el-radio-group v-model="batchMode" class="flex flex-col items-start gap-2">
      <el-radio value="fill">只补无摘要的文章</el-radio>
      <el-radio value="overwrite">替换所有文章的摘要</el-radio>
    </el-radio-group>
    <p class="mt-4 text-xs text-gray-400">由 AI 在后台生成，同时最多处理 5 篇，完成后会自动刷新列表。</p>
    <template #footer>
      <el-button @click="batchDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="startBatch">开始生成</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import ArticleTable from '@/components/backend/Table/ArticleTable.vue'
import CategoryTable from '@/components/backend/Table/CategoryTable.vue'
import TagTable from '@/components/backend/Table/TagTable.vue'

import { getArticleList, startBatchAISummary, getBatchAISummaryStatus } from '@/api/admin';
import type { BatchSummaryMode } from '@/api/admin';
import { Article } from '@/types/Article';

import { MagicStick } from '@element-plus/icons-vue'
import { notify } from '@/utils/notification'
import { Tag } from '@/types/Tag';
import { onMounted, onUnmounted, reactive } from 'vue';
import { ref } from 'vue';
import { useAdminStore } from '@/store/';


const activeName = ref('first')
const store = useAdminStore()

const articles = reactive<Article<Tag>[]>([])
const categories = store.categories
const tags = store.tags

// 摘要批量生成：只在开始与结束时提示，中途 3 秒静默轮询进度
const batchDialogVisible = ref(false)
const batchMode = ref<BatchSummaryMode>('fill')
const batchRunning = ref(false)
let batchTimer: ReturnType<typeof setInterval> | null = null

const stopPolling = () => {
  if (batchTimer !== null) {
    clearInterval(batchTimer)
    batchTimer = null
  }
}

const startPolling = () => {
  stopPolling()
  batchRunning.value = true
  batchTimer = setInterval(async () => {
    try {
      const status = await getBatchAISummaryStatus()
      if (status.running) return
      stopPolling()
      batchRunning.value = false
      notify.success(`摘要生成完成：成功 ${status.done - status.failed} 篇，失败 ${status.failed} 篇`)
      await loadArticles()
    } catch {
      // 状态查不到时按结束处理，错误提示由 axios 拦截器负责
      stopPolling()
      batchRunning.value = false
    }
  }, 3000)
}

const startBatch = async () => {
  batchDialogVisible.value = false
  try {
    const res = await startBatchAISummary(batchMode.value)
    if (!res.started) {
      notify.info('没有需要生成摘要的文章')
      return
    }
    notify.success(`已开始为 ${res.total} 篇文章生成摘要，完成后会自动刷新`)
    startPolling()
  } catch {
    // 已有批次在跑等错误由 axios 拦截器提示
  }
}

// 加载文章列表
const loadArticles = async () => {
  // 清空现有文章列表
  articles.splice(0, articles.length)

  // 重新获取文章列表
  const res = await getArticleList({ pageNum: 1, pageSize: 10, total: 10 })
  const articleList: Article<Tag>[] = []
  res.list.forEach(item => {
    item.categoryName = categories.find(c => c.id === item.categoryId)?.name
    articleList.push({ ...item })
  })
  articles.push(...articleList)
}

onMounted(async () => {
  await store.getCategories()
  await store.getTags()
  await loadArticles()
  // 页面刷新时批次可能仍在运行，恢复轮询，避免按钮显示为空闲
  try {
    if ((await getBatchAISummaryStatus()).running) startPolling()
  } catch {
    // 状态未知时按空闲处理
  }
})

onUnmounted(stopPolling)
</script>
<style scoped>
.box-card {
  width: 460px;
}
</style>
