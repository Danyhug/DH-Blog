<template>
  <el-tabs v-model="activeName" type="border-card">
    <el-tab-pane label="文章管理" name="first">
      <div class="flex items-center justify-between mb-3">
        <el-button type="warning" plain @click="openGrantDialog">
          <el-icon class="mr-1"><Key /></el-icon>
          生成 AI 修改授权
        </el-button>
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

  <el-dialog v-model="grantDialogVisible" title="生成 AI 修改授权" width="620px" @close="stopCountdown">
    <el-form :model="{ note: grantNote }" label-width="90px" size="default" @submit.prevent>
      <el-form-item label="授权范围">
        <el-radio-group v-model="grantScope">
          <el-radio value="one">指定文章</el-radio>
          <el-radio value="all">全站所有文章</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="grantScope === 'one'" label="文章编号">
        <el-input-number v-model="grantArticleId" :min="1" :controls="false" />
        <span v-if="grantArticleTitle" class="ml-2 text-xs text-gray-500">《{{ grantArticleTitle }}》</span>
        <span v-else class="ml-2 text-xs text-gray-400">左侧文章列表第一列的「编号」</span>
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="grantNote" placeholder="让 Claude 改错别字" clearable />
      </el-form-item>
    </el-form>
    <div class="flex gap-2 mb-4">
      <el-button type="warning" plain :loading="grantCreating" @click="createGrant">
        <el-icon class="mr-1"><Key /></el-icon>
        生成授权
      </el-button>
    </div>

    <el-alert type="info" :closable="false" show-icon class="mb-3">
      把这段 Token 交给 Agent，作为 update_article 的 edit_token 参数，1 小时内有效；也可吊销使其立即失效。
      {{ grantScopeHint }}
    </el-alert>

    <template v-if="createdGrantVisible">
      <div class="text-xs text-gray-400 mb-1">Token（只有这次能看到明文，复制后请妥善保存）</div>
      <div class="flex gap-2 items-center">
        <el-input :model-value="createdGrantToken" readonly class="flex-1" />
        <el-button :icon="CopyDocument" :disabled="countdownExpired" @click="copy(createdGrantToken, 'Token')">复制</el-button>
      </div>
      <div class="text-xs text-gray-500 mt-2">
        有效期剩余：
        <span :class="countdownExpired ? 'text-red-500 font-medium' : ''">{{ countdownText }}</span>
      </div>
      <el-divider class="my-4" />
    </template>

    <div class="text-sm font-medium mb-2">当前有效授权</div>
    <el-table :data="grants" v-loading="grantsLoading" size="small" empty-text="暂无有效授权">
      <el-table-column prop="tokenPrefix" label="Token 前缀" min-width="120" />
      <el-table-column label="授权范围" width="110">
        <template #default="scope">
          <el-tag v-if="scope.row.articleId" size="small" type="success" effect="plain">
            文章 #{{ scope.row.articleId }}
          </el-tag>
          <el-tag v-else size="small" type="danger" effect="plain">全站</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="note" label="备注" min-width="140">
        <template #default="scope">{{ scope.row.note || '—' }}</template>
      </el-table-column>
      <el-table-column prop="usedCount" label="已用次数" width="90" />
      <el-table-column prop="expireAt" label="过期时间" min-width="150" />
      <el-table-column label="操作" width="150">
        <template #default="scope">
          <el-button link type="primary" size="small" @click="revealGrant(scope.row)">再看明文</el-button>
          <el-button link type="danger" size="small" @click="revokeGrant(scope.row)">吊销</el-button>
        </template>
      </el-table-column>
    </el-table>

    <template #footer>
      <el-button type="primary" @click="grantDialogVisible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import ArticleTable from '@/components/backend/Table/ArticleTable.vue'
import CategoryTable from '@/components/backend/Table/CategoryTable.vue'
import TagTable from '@/components/backend/Table/TagTable.vue'

import { getArticleList, startBatchAISummary, getBatchAISummaryStatus } from '@/api/admin';
import type { BatchSummaryMode } from '@/api/admin';
import { createAgentGrant, deleteAgentGrant, getAgentGrants, revealAgentGrant } from '@/api/agent';
import type { AgentGrant } from '@/api/agent';
import { Article } from '@/types/Article';

import { CopyDocument, Key, MagicStick } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import { notify } from '@/utils/notification'
import { Tag } from '@/types/Tag';
import { onMounted, onUnmounted, reactive, computed } from 'vue';
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

// AI 修改授权：签发弹窗 + 剩余时间倒计时 + 当前有效授权列表
const grantDialogVisible = ref(false)
const grantNote = ref('')
// 默认按最小权限走：只授权某一篇，全站授权必须显式选择
const grantScope = ref<'one' | 'all'>('one')
const grantArticleId = ref<number>()
const grantCreating = ref(false)
const grants = ref<AgentGrant[]>([])
const grantsLoading = ref(false)
const createdGrantVisible = ref(false)
const createdGrantToken = ref('')
// 记住已签发那条的范围：签出来之后再改单选框，说明文字不能跟着变
const createdGrantArticleId = ref<number>()
const countdownText = ref('')
const countdownExpired = ref(false)
let countdownTimer: ReturnType<typeof setInterval> | null = null

// 文章编号是手填的，能在当前列表里查到标题就回显一下，避免填错编号还浑然不觉
const grantArticleTitle = computed(() => {
  if (!grantArticleId.value) return ''
  return articles.find(item => item.id === grantArticleId.value)?.title ?? ''
})

const grantScopeHint = computed(() => {
  if (createdGrantVisible.value) {
    return createdGrantArticleId.value
      ? `这段 Token 只能修改文章 #${createdGrantArticleId.value}。`
      : '这段 Token 是全站授权：有效期内可以修改任意一篇文章。'
  }
  return grantScope.value === 'one'
    ? '将只授权修改指定编号的那一篇。'
    : '将签发全站授权：有效期内 Agent 可以修改任意一篇文章。'
})

const stopCountdown = () => {
  if (countdownTimer !== null) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

// 后端返回 "YYYY-MM-DD HH:mm:ss"，直转 new Date 在 Safari 会得到 NaN，先替换分隔符
const parseTime = (s: string) => new Date(s.replace(/-/g, '/')).getTime()

const startCountdown = (expireAt: string) => {
  stopCountdown()
  const expireMs = parseTime(expireAt)
  const tick = () => {
    const remain = Math.max(0, expireMs - Date.now())
    if (remain <= 0) {
      countdownExpired.value = true
      countdownText.value = '已过期'
      stopCountdown()
      notify.warning('授权已过期，如需继续编辑请重新生成')
      return
    }
    const totalSeconds = Math.floor(remain / 1000)
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60
    countdownText.value = minutes > 0
      ? `${minutes}:${String(seconds).padStart(2, '0')}`
      : `0:${String(seconds).padStart(2, '0')}`
  }
  tick()
  countdownTimer = setInterval(tick, 1000)
}

const openGrantDialog = async () => {
  grantDialogVisible.value = true
  createdGrantVisible.value = false
  createdGrantToken.value = ''
  createdGrantArticleId.value = undefined
  grantNote.value = ''
  grantScope.value = 'one'
  grantArticleId.value = undefined
  countdownText.value = ''
  countdownExpired.value = false
  stopCountdown()
  await loadGrants()
}

const loadGrants = async () => {
  grantsLoading.value = true
  try {
    grants.value = await getAgentGrants()
  } catch {
    // 错误提示由 axios 拦截器统一负责
  } finally {
    grantsLoading.value = false
  }
}

const createGrant = async () => {
  if (grantScope.value === 'one' && !grantArticleId.value) {
    notify.warning('请填写要授权修改的文章编号')
    return
  }
  grantCreating.value = true
  try {
    const created = await createAgentGrant({
      note: grantNote.value || undefined,
      articleId: grantScope.value === 'one' ? grantArticleId.value : undefined,
    })
    createdGrantToken.value = created.token
    createdGrantArticleId.value = created.articleId || undefined
    createdGrantVisible.value = true
    countdownExpired.value = false
    startCountdown(created.expireAt)
    notify.success(created.articleId ? `已生成授权：仅限文章 #${created.articleId}` : '已生成全站授权')
    grantNote.value = ''
    await loadGrants()
  } catch {
    // 错误提示由 axios 拦截器统一负责
  } finally {
    grantCreating.value = false
  }
}

const copy = async (text: string, label: string) => {
  try {
    await navigator.clipboard.writeText(text)
    notify.success(`${label}已复制`)
  } catch {
    notify.warning('复制失败，请手动选中复制')
  }
}

const revealGrant = async (grant: AgentGrant) => {
  try {
    const revealed = await revealAgentGrant(grant.id)
    await copy(revealed.token, 'Token')
  } catch {
    // 错误提示由 axios 拦截器统一负责
  }
}

const revokeGrant = async (grant: AgentGrant) => {
  try {
    await ElMessageBox.confirm(
      `确定要吊销「${grant.note || grant.tokenPrefix}」这条授权吗？使用该 Token 的 agent 将立即失效。`,
      '提示',
      { type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await deleteAgentGrant(grant.id)
    notify.success('已吊销')
    await loadGrants()
  } catch {
    // 错误提示由 axios 拦截器统一负责
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

onUnmounted(() => {
  stopPolling()
  stopCountdown()
})
</script>
<style scoped>
.box-card {
  width: 460px;
}
</style>
