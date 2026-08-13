<template>
  <section
    class="flex flex-col overflow-hidden rounded-[14px] border border-[#edf0f3] bg-white shadow-[0_1px_2px_rgba(16,24,40,0.04)]">
    <header class="shrink-0 border-b border-[#f2f4f7] px-3 py-2">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <el-icon :size="14" class="text-[#3f8cff]">
            <Tickets />
          </el-icon>
          <span class="text-[13px] font-medium text-[#1f2937]">后台任务</span>
          <span class="font-mono text-[11px] tabular-nums text-[#98a2b3]">共 {{ total }} 条</span>
        </div>

        <div class="flex flex-wrap items-center gap-1.5">
          <el-select :model-value="source" placeholder="全部来源" clearable size="small" class="w-28"
            @update:model-value="pickSource">
            <el-option v-for="item in sourceOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-select :model-value="status" placeholder="全部状态" clearable size="small" class="w-28"
            @update:model-value="pickStatus">
            <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-button size="small" :icon="Refresh" circle title="刷新" @click="emit('reload', page)" />
          <el-button size="small" circle :title="expanded ? '还原为两栏' : '展开占满'" @click="emit('toggle-expand')">
            <el-icon :size="13">
              <ScaleToOriginal v-if="expanded" />
              <FullScreen v-else />
            </el-icon>
          </el-button>
        </div>
      </div>
    </header>

    <!-- feed 而不是数据网格，所以不用 el-table：状态色块要贴住整行左缘 -->
    <div v-loading="loading" class="min-h-0 flex-1 overflow-y-auto">
      <!-- 筛选或翻页时新事件会被拦下来，得说清楚现在看的不是实时的 -->
      <div v-if="!live"
        class="sticky top-0 z-10 flex flex-wrap items-center gap-x-2 gap-y-1 border-b border-[#fde9c8] bg-[#fffaf0]
               px-3 py-2 text-xs text-[#b45309]">
        <el-icon :size="13">
          <Warning />
        </el-icon>
        <span v-if="pendingCount">已暂存 {{ pendingCount }} 个新事件</span>
        <span v-else>正在查看历史，新事件不会自动插入</span>
        <el-button link type="primary" size="small" @click="emit('back-to-live')">回到实时</el-button>
      </div>

      <div v-if="!events.length" class="px-6 py-16 text-center">
        <p class="text-sm leading-7 text-[#98a2b3]">
          <template v-if="source || status">没有匹配当前筛选的事件。</template>
          <template v-else>还没有后台任务跑过。<br>发布文章会触发 AI 标签和摘要生成，跑完就会记在这里。</template>
        </p>
        <el-button v-if="source || status" link type="primary" class="mt-1" @click="emit('back-to-live')">
          清除筛选
        </el-button>
      </div>

      <button v-for="event in events" :key="event.id" type="button"
        class="flex w-full items-start gap-2.5 border-b border-[#f6f7f9] px-3 py-2.5 text-left transition-colors
               last:border-0 hover:bg-[#fafbfc] focus:outline-none focus-visible:bg-[#f4f8ff]"
        @click="openDetail(event)">
        <!-- 图标 + 文字一起表达状态，只靠颜色的话色觉障碍就读不出成败 -->
        <span class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg"
          :class="look[event.status]?.chip || look.queued.chip">
          <el-icon :size="14" :class="event.status === 'running' ? 'animate-spin' : ''">
            <component :is="look[event.status]?.icon || look.queued.icon" />
          </el-icon>
        </span>

        <div class="min-w-0 flex-1">
          <div class="text-[13px] leading-5 text-[#1f2937]">{{ event.title }}</div>
          <div class="mt-0.5 flex flex-wrap items-center gap-x-1.5 text-[11px] leading-4 text-[#98a2b3]">
            <span :class="look[event.status]?.text || look.queued.text">{{ statusLabel(event.status) }}</span>
            <span>·</span>
            <span>{{ sourceLabel(event.source) }}</span>
            <span>·</span>
            <span class="font-mono">{{ event.kind }}</span>
            <template v-if="event.attempt > 0">
              <span>·</span>
              <span>第 {{ event.attempt }} 次</span>
            </template>
            <template v-if="event.targetId > 0">
              <span>·</span>
              <span class="font-mono">#{{ event.targetId }}</span>
            </template>
          </div>
          <div v-if="event.detail" class="mt-1 truncate font-mono text-[11px] leading-4 text-[#b0b7c3]">
            {{ event.detail }}
          </div>
        </div>

        <!-- 相对时间好扫，绝对时间才是证据：绝对值挂在 title 上，详情里也有 -->
        <span class="shrink-0 pt-0.5 text-[11px] tabular-nums text-[#b0b7c3]" :title="event.createdAt">
          {{ relative(event.createdAt) }}
        </span>
      </button>
    </div>

    <div v-if="total > pageSize" class="flex shrink-0 justify-end border-t border-[#f2f4f7] px-3 py-2">
      <el-pagination layout="prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" size="small"
        @current-change="turnPage" />
    </div>
  </section>

  <!-- 详情用抽屉而不是跳转：翻页位置和筛选条件都还留着，看完就能接着往下扫 -->
  <el-drawer v-model="drawerOpen" title="事件详情" size="min(420px, 90vw)" @closed="detail = null">
    <div v-if="detail" class="flex flex-col gap-4 text-sm">
      <div class="flex items-start gap-2.5">
        <span class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg"
          :class="look[detail.status]?.chip || look.queued.chip">
          <el-icon :size="14">
            <component :is="look[detail.status]?.icon || look.queued.icon" />
          </el-icon>
        </span>
        <span class="leading-6 text-[#1f2937]">{{ detail.title }}</span>
      </div>

      <dl class="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px]">
        <dt class="text-[#98a2b3]">状态</dt>
        <dd :class="look[detail.status]?.text || look.queued.text">{{ statusLabel(detail.status) }}</dd>
        <dt class="text-[#98a2b3]">时间</dt>
        <dd class="font-mono tabular-nums text-[#1f2937]">{{ detail.createdAt }}</dd>
        <dt class="text-[#98a2b3]">来源</dt>
        <dd class="text-[#1f2937]">{{ sourceLabel(detail.source) }}</dd>
        <dt class="text-[#98a2b3]">任务标识</dt>
        <dd class="font-mono text-[#1f2937]">{{ detail.kind }}</dd>
        <dt class="text-[#98a2b3]">重试次数</dt>
        <dd class="text-[#1f2937]">{{ detail.attempt || '未重试' }}</dd>
        <dt class="text-[#98a2b3]">事件 ID</dt>
        <dd class="font-mono text-[#1f2937]">#{{ detail.id }}</dd>
      </dl>

      <div v-if="detail.detail">
        <p class="mb-1.5 text-[13px] text-[#98a2b3]">详情</p>
        <!-- 错误原文可能很长，这里不截断，整段可选可复制 -->
        <pre
          class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-[#f7f8fa] p-3 font-mono
                 text-[12px] leading-5 text-[#475569]">{{ detail.detail }}</pre>
      </div>

      <div class="flex gap-2">
        <el-button v-if="detail.targetId > 0" type="primary" @click="emit('open-article', detail.targetId)">
          打开文章 #{{ detail.targetId }}
        </el-button>
        <el-button :icon="CopyDocument" @click="copyDetail">复制</el-button>
      </div>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { onUnmounted, ref, type Component } from 'vue'
import {
  Clock, CircleCheck, CircleClose, CopyDocument, FullScreen, Loading,
  Refresh, RefreshRight, ScaleToOriginal, Tickets, Warning,
} from '@element-plus/icons-vue'
import { notify } from '@/utils/notification'
import type { BackgroundEvent } from '@/api/eventlog'

defineProps<{
  events: BackgroundEvent[]
  total: number
  page: number
  pageSize: number
  loading: boolean
  /** 停在第一页且没筛选时才是实时的，否则要提示用户 */
  live: boolean
  pendingCount: number
  source: string
  status: string
  expanded: boolean
}>()

const emit = defineEmits<{
  'update:source': [string]
  'update:status': [string]
  reload: [number]
  'toggle-expand': []
  'back-to-live': []
  'open-article': [number]
}>()

const detail = ref<BackgroundEvent | null>(null)
const drawerOpen = ref(false)

// 抽屉关到一半再清空数据的话，收起动画里会闪一下空白，所以等 closed 了再清
const openDetail = (event: BackgroundEvent) => {
  detail.value = event
  drawerOpen.value = true
}

// el-select 清空时给的是 undefined，统一收敛成空串再往上抛，父组件只认一种"没筛选"
const pickSource = (value: unknown) => emit('update:source', (value as string) || '')
const pickStatus = (value: unknown) => emit('update:status', (value as string) || '')
const turnPage = (value: number) => emit('reload', value)

const sourceOptions = [
  { value: 'task', label: '任务队列' },
  { value: 'webdav', label: '网盘' },
  { value: 'gateway', label: 'AI 网关' },
  { value: 'article', label: '文章' },
]

const statusOptions = [
  { value: 'queued', label: '已入队' },
  { value: 'running', label: '执行中' },
  { value: 'retrying', label: '重试中' },
  { value: 'success', label: '成功' },
  { value: 'failed', label: '失败' },
]

const statusLabel = (value: string) => statusOptions.find(item => item.value === value)?.label || value
const sourceLabel = (value: string) => sourceOptions.find(item => item.value === value)?.label || value

/** 一个状态的全部视觉表现集中在一处，免得图标、底色、文字色三处各写一遍对不上 */
const look: Record<string, { icon: Component; chip: string; text: string }> = {
  queued: { icon: Clock, chip: 'bg-[#f2f4f7] text-[#98a2b3]', text: 'text-[#98a2b3]' },
  running: { icon: Loading, chip: 'bg-[#eef4ff] text-[#3f8cff]', text: 'text-[#3f8cff]' },
  retrying: { icon: RefreshRight, chip: 'bg-[#fff6e6] text-[#b45309]', text: 'text-[#b45309]' },
  success: { icon: CircleCheck, chip: 'bg-[#eaf7f1] text-[#1fa97c]', text: 'text-[#1fa97c]' },
  failed: { icon: CircleClose, chip: 'bg-[#fdeced] text-[#e23d4d]', text: 'text-[#e23d4d]' },
}

/* ---------- 时间 ---------- */

// 相对时间会随时间走样，"刚刚"挂一分钟就不再是刚刚了，所以定时重算
const now = ref(Date.now())
const timer = setInterval(() => (now.value = Date.now()), 30000)
onUnmounted(() => clearInterval(timer))

const pad = (value: number) => String(value).padStart(2, '0')

/** 后端给的是 "2006-01-02 15:04:05"，补上 T 才能被各家浏览器当本地时间解析 */
const relative = (value: string) => {
  const parsed = new Date(value.replace(' ', 'T'))
  if (Number.isNaN(parsed.getTime())) return value

  const seconds = Math.floor((now.value - parsed.getTime()) / 1000)
  if (seconds < 60) return '刚刚'
  if (seconds < 3600) return `${Math.floor(seconds / 60)} 分钟前`

  const clock = `${pad(parsed.getHours())}:${pad(parsed.getMinutes())}`
  const today = new Date(now.value)
  const days = Math.round(
    (new Date(today.getFullYear(), today.getMonth(), today.getDate()).getTime()
      - new Date(parsed.getFullYear(), parsed.getMonth(), parsed.getDate()).getTime()) / 86400000)
  if (days === 0) return `${Math.floor(seconds / 3600)} 小时前`
  if (days === 1) return `昨天 ${clock}`
  return `${pad(parsed.getMonth() + 1)}-${pad(parsed.getDate())} ${clock}`
}

const copyDetail = async () => {
  const event = detail.value
  if (!event) return
  const text = [
    event.title,
    `状态：${statusLabel(event.status)}`,
    `时间：${event.createdAt}`,
    `来源：${sourceLabel(event.source)} / ${event.kind}`,
    `事件 ID：${event.id}`,
    event.detail && `详情：${event.detail}`,
  ].filter(Boolean).join('\n')
  try {
    await navigator.clipboard.writeText(text)
    notify.success('已复制事件详情')
  } catch {
    // http 环境下没有 clipboard API，静默失败会让人以为复制成功了
    notify.error('当前环境不允许访问剪贴板')
  }
}
</script>
