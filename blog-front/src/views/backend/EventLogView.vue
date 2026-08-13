<template>
  <div class="flex flex-col gap-3">
    <!-- 标题一行说完：连接状态是次要信息，跟在右边，不再单开一层眉标 -->
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h2 class="text-[16px] font-semibold leading-6 text-[#1f2937]">运行日志</h2>
        <p class="mt-0.5 text-xs text-[#98a2b3]">服务器输出与后台任务的实时流水</p>
      </div>
      <div class="flex items-center gap-2">
        <span class="flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs" :class="liveLook.pill">
          <span class="inline-block h-1.5 w-1.5 rounded-full motion-reduce:animate-none" :class="liveLook.dot"></span>
          {{ liveLook.text }}
        </span>
      </div>
    </div>

    <!-- 左右两栏等高；任一栏可以展开占满，另一栏让位 -->
    <div class="grid gap-3" :class="pane === 'split' ? 'lg:grid-cols-[1.3fr_1fr]' : 'grid-cols-1'">
      <ConsolePane v-show="pane !== 'tasks'" :class="paneHeight" :lines="lines" :max-lines="MAX_LINES"
        :connected="socketState === 'open'" :visible="pane !== 'tasks'" :expanded="pane === 'console'"
        @clear="lines = []" @toggle-expand="togglePane('console')" />

      <TaskFeed v-show="pane !== 'console'" :class="paneHeight" :events="events" :total="total" :page="page"
        :page-size="pageSize" :loading="loading" :live="live" :pending-count="pending.length" :source="source"
        :status="status" :expanded="pane === 'tasks'" @update:source="value => { source = value; reload(1) }"
        @update:status="value => { status = value; reload(1) }" @reload="reload" @back-to-live="backToLive"
        @open-article="openArticle" @toggle-expand="togglePane('tasks')" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import ConsolePane from '@/components/backend/eventlog/ConsolePane.vue'
import TaskFeed from '@/components/backend/eventlog/TaskFeed.vue'
import { notify } from '@/utils/notification'
import { EventSocket, type HelloPayload, type SocketState } from '@/utils/eventSocket'
import {
  getBackgroundEvents,
  getServerLogs,
  replayBackgroundEvents,
  type BackgroundEvent,
  type ServerLogLine,
} from '@/api/eventlog'

const router = useRouter()

/* ---------- 分栏 ---------- */

type Pane = 'split' | 'console' | 'tasks'

const PANE_KEY = 'eventlog:pane'
const pane = ref<Pane>((localStorage.getItem(PANE_KEY) as Pane) || 'split')

/** 两栏等高、各自内部滚动，页面本身不滚 */
const paneHeight = 'h-[calc(100vh-11.5rem)] min-h-[26rem]'

const togglePane = (target: Exclude<Pane, 'split'>) => {
  pane.value = pane.value === target ? 'split' : target
  localStorage.setItem(PANE_KEY, pane.value)
}

/* ---------- 服务器输出 ---------- */

/** 客户端最多留这么多行，和后端 ring buffer 一样大 */
const MAX_LINES = 1000

const lines = ref<ServerLogLine[]>([])
/** 已经见过的最大日志序号，用来丢掉重复行 */
const logCursor = ref(0)

// 日志可能一瞬间刷进来几十行。每行都触发一次渲染加一次滚动的话，
// 页面会明显卡顿，所以攒到下一帧一次性并进列表
let logBuffer: ServerLogLine[] = []
let flushHandle: number | null = null

const flushLogs = () => {
  flushHandle = null
  if (!logBuffer.length) return
  lines.value.push(...logBuffer)
  logBuffer = []
  if (lines.value.length > MAX_LINES) {
    lines.value.splice(0, lines.value.length - MAX_LINES)
  }
}

const appendLine = (line: ServerLogLine) => {
  if (!line || line.seq <= logCursor.value) return
  logCursor.value = line.seq
  logBuffer.push(line)
  if (flushHandle === null) flushHandle = requestAnimationFrame(flushLogs)
}

const loadLogs = async () => {
  const res = await getServerLogs(300)
  // 缓冲区里可能还压着比快照更旧的行，整批丢掉免得插到快照前面
  logBuffer = []
  lines.value = res.lines || []
  logCursor.value = res.cursor
}

/* ---------- 后台任务 ---------- */

const events = ref<BackgroundEvent[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 30
const loading = ref(false)
const source = ref('')
const status = ref('')
const socketState = ref<SocketState>('closed')
const cursor = ref(0)
const pending = ref<BackgroundEvent[]>([])

let socket: EventSocket | null = null

/** 只有停在第一页且没筛选时才让新事件直接插进列表，否则会打乱正在看的内容 */
const live = computed(() => page.value === 1 && !source.value && !status.value)

const liveLook = computed(() => {
  switch (socketState.value) {
    case 'open':
      return { text: '实时', dot: 'bg-[#1fa97c]', pill: 'border-[#c7ecdd] bg-[#f2fbf7] text-[#1fa97c]' }
    case 'connecting':
      return { text: '重连中', dot: 'bg-[#e0a030] animate-pulse', pill: 'border-[#fde9c8] bg-[#fffaf0] text-[#b45309]' }
    default:
      return { text: '已断开', dot: 'bg-[#cbd5e1]', pill: 'border-[#eaecf0] bg-[#f9fafb] text-[#98a2b3]' }
  }
})

const openArticle = (id: number) => {
  router.push({ name: 'publish', query: { articleId: id } })
}

const reload = async (target: number) => {
  page.value = target
  loading.value = true
  try {
    const res = await getBackgroundEvents({
      page: target,
      pageSize,
      source: source.value || undefined,
      status: status.value || undefined,
    })
    events.value = res.list || []
    total.value = res.total
    if (target === 1 && events.value.length) {
      cursor.value = Math.max(cursor.value, events.value[0].id)
    }
  } finally {
    loading.value = false
  }
}

const prepend = (event: BackgroundEvent) => {
  if (events.value.some(item => item.id === event.id)) return
  events.value.unshift(event)
  if (events.value.length > pageSize) events.value.pop()
  total.value += 1
}

const backToLive = async () => {
  source.value = ''
  status.value = ''
  pending.value = []
  await reload(1)
}

const onEvent = (event: BackgroundEvent) => {
  if (!event) return
  cursor.value = Math.max(cursor.value, event.id)
  // 失败是这个页面最该被看见的东西，弹一条出来，避免正好在别的标签页
  if (event.status === 'failed') {
    notify.error({ message: event.title, duration: 6000 })
  }
  if (live.value) prepend(event)
  else pending.value.push(event)
}

/** 重连成功后补齐断连期间的事件，缺口太大就直接重新拉第一页 */
const onHello = async (hello: HelloPayload) => {
  // 控制台不做增量补齐：日志是易逝的，重新取一次回滚更简单也更准
  if (hello.logCursor > logCursor.value) loadLogs()

  if (!cursor.value || hello.cursor <= cursor.value) return
  try {
    const res = await replayBackgroundEvents(cursor.value)
    if (res.truncated) {
      await reload(1)
      notify.info('断线期间事件较多，已重新加载列表')
      return
    }
    res.events.forEach(onEvent)
  } catch {
    // 补齐失败不致命，错误提示由 axios 拦截器负责
  }
}

onMounted(async () => {
  await Promise.all([reload(1), loadLogs()])
  socket = new EventSocket({
    onEvent,
    onLog: appendLine,
    onHello,
    onState: state => (socketState.value = state),
  })
  socket.open()
})

onUnmounted(() => {
  socket?.close()
  socket = null
  if (flushHandle !== null) cancelAnimationFrame(flushHandle)
})
</script>
