<template>
  <section
    class="flex flex-col overflow-hidden rounded-[14px] border border-[#1e2a44] bg-[#0f1729] shadow-[0_1px_2px_rgba(16,24,40,0.06)]">
    <!-- 头部两行：上行是身份 + 级别筛选，下行是搜索 + 动作。
         挤成一行的话，分栏模式下宽度不够，按钮会换行成锯齿状 -->
    <header class="shrink-0 border-b border-[#1e2a44] bg-[#131d31] px-3 py-2">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <el-icon :size="14" class="text-[#60a5fa]">
            <Monitor />
          </el-icon>
          <span class="text-[13px] font-medium text-slate-200">服务器输出</span>
        </div>

        <!-- 级别既是筛选也是分布：带上条数才能一眼看出这一屏里错了几条 -->
        <div class="flex items-center gap-0.5">
          <button v-for="option in levelFilters" :key="option.value" type="button"
            class="rounded-md px-2 py-1 text-[11px] transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[#60a5fa]/60"
            :class="level === option.value
              ? 'bg-[#24334f] text-slate-100'
              : 'text-slate-500 hover:bg-white/5 hover:text-slate-300'"
            @click="level = option.value">
            <span :class="option.tone">{{ option.label }}</span>
            <span class="ml-1 font-mono tabular-nums text-slate-500">{{ counts[option.value] }}</span>
          </button>
        </div>
      </div>

      <div class="mt-2 flex items-center gap-1.5">
        <!-- 原生 input：Element Plus 的输入框是浅色主题，放进深色面板要覆一堆 :deep -->
        <div class="flex min-w-0 flex-1 items-center gap-1.5 rounded-md bg-[#0f1729] px-2 py-1
                    ring-1 ring-[#24334f] focus-within:ring-[#60a5fa]/70">
          <el-icon :size="12" class="shrink-0 text-slate-500">
            <Search />
          </el-icon>
          <input v-model="keyword" type="text" placeholder="搜索日志内容…"
            class="min-w-0 flex-1 bg-transparent font-mono text-[12px] text-slate-200 placeholder:font-sans
                   placeholder:text-slate-600 focus:outline-none" />
          <button v-if="keyword" type="button" class="shrink-0 text-slate-500 hover:text-slate-300"
            title="清除搜索" @click="keyword = ''">
            <el-icon :size="12">
              <Close />
            </el-icon>
          </button>
        </div>

        <button type="button" :title="follow ? '暂停自动滚动' : '继续自动滚动'" :class="[iconButton, follow ? '' : 'text-[#fbbf24]']"
          @click="toggleFollow">
          <el-icon :size="13">
            <VideoPause v-if="follow" />
            <VideoPlay v-else />
          </el-icon>
        </button>
        <button type="button" :title="wrap ? '关闭自动换行' : '开启自动换行'"
          :class="[iconButton, wrap ? 'text-[#60a5fa]' : '']" @click="wrap = !wrap">
          <el-icon :size="13">
            <Sort />
          </el-icon>
        </button>
        <button type="button" title="复制当前可见日志" :class="iconButton" @click="copyVisible">
          <el-icon :size="13">
            <CopyDocument />
          </el-icon>
        </button>
        <button type="button" title="下载当前可见日志" :class="iconButton" @click="downloadVisible">
          <el-icon :size="13">
            <Download />
          </el-icon>
        </button>
        <button type="button" title="清空控制台" :class="iconButton" @click="emit('clear')">
          <el-icon :size="13">
            <Delete />
          </el-icon>
        </button>
        <button type="button" :title="expanded ? '还原为两栏' : '展开占满'" :class="iconButton"
          @click="emit('toggle-expand')">
          <el-icon :size="13">
            <ScaleToOriginal v-if="expanded" />
            <FullScreen v-else />
          </el-icon>
        </button>
      </div>
    </header>

    <div class="relative min-h-0 flex-1">
      <div ref="body" class="h-full overflow-auto py-1.5" @scroll="onScroll">
        <div v-if="!visibleLines.length" class="px-4 py-14 text-center">
          <p class="text-xs leading-6 text-slate-500">{{ emptyText }}</p>
          <button v-if="filtered" type="button"
            class="mt-2 text-xs text-[#60a5fa] hover:underline" @click="resetFilters">
            清除筛选条件
          </button>
        </div>

        <!-- 关掉换行时整块跟着最长的一行变宽，这样横向滚动过去，
             每行的悬停底色和错误底色仍然铺满，不会只涂到容器宽度就断掉 -->
        <div :class="wrap ? '' : 'w-max min-w-full'">
          <!-- 签名元素：每行左侧的级别色轨。滚起来错误会自己聚成一段红带，
               不用读字就知道哪一段出了事；级别缩写仍然保留，颜色不能是唯一信息 -->
          <div v-for="line in visibleLines" :key="line.seq"
            class="flex items-start gap-2.5 px-3 py-[3px]"
            :class="bucketOf(line.level) === 'error' ? 'bg-[#f87171]/[0.07]' : 'hover:bg-white/[0.03]'">
            <span class="mt-[6px] h-2.5 w-[3px] shrink-0 rounded-full" :class="rail[bucketOf(line.level)]"></span>
            <span class="shrink-0 font-mono text-[12px] leading-[1.6] tabular-nums text-slate-500">{{ line.time }}</span>
            <span class="hidden w-9 shrink-0 font-mono text-[10px] uppercase leading-[1.9] tracking-[0.06em] sm:block"
              :class="tone[bucketOf(line.level)]">{{ short[bucketOf(line.level)] }}</span>
            <span class="font-mono text-[12px] leading-[1.6]"
              :class="[
                bucketOf(line.level) === 'error' ? 'text-[#fca5a5]' : 'text-slate-300',
                wrap ? 'min-w-0 flex-1 break-words' : 'whitespace-pre',
              ]"><span v-for="(part, index) in lineParts(line)" :key="index" :class="part.hit
                ? 'rounded-[2px] bg-[#fbbf24]/30 text-[#fde68a]'
                : (part.dim ? 'text-slate-600' : '')">{{ part.text }}</span></span>
          </div>
        </div>
      </div>

      <!-- 暂停期间来的新行不打断阅读，攒成一个可点的提示浮在底部 -->
      <button v-if="!follow && missed" type="button"
        class="absolute bottom-3 left-1/2 flex -translate-x-1/2 items-center gap-1.5 rounded-full bg-[#2563eb]
               px-3 py-1.5 text-[11px] text-white shadow-lg shadow-black/30 transition-colors hover:bg-[#3b82f6]"
        @click="toBottom">
        <el-icon :size="12">
          <Bottom />
        </el-icon>
        {{ missed }} 条新日志
      </button>
    </div>

    <footer
      class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-t border-[#1e2a44] bg-[#131d31]
             px-3 py-1.5 font-mono text-[11px] text-slate-500">
      <!-- 保留上限写在这里：看到的不是全部，得说出来 -->
      <span class="tabular-nums">
        显示 {{ visibleLines.length }} / {{ lines.length }} 行 · 最多保留 {{ maxLines }} 行
      </span>
      <span class="flex items-center gap-1.5" :class="follow ? 'text-[#60a5fa]' : 'text-[#fbbf24]'">
        {{ follow ? '跟随最新' : '已暂停' }}
        <span v-if="follow" class="motion-reduce:animate-none" :class="connected ? 'animate-pulse' : 'opacity-30'">▌</span>
      </span>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  Bottom, Close, CopyDocument, Delete, Download, FullScreen,
  Monitor, ScaleToOriginal, Search, Sort, VideoPause, VideoPlay,
} from '@element-plus/icons-vue'
import { notify } from '@/utils/notification'
import type { ServerLogLine } from '@/api/eventlog'

const props = defineProps<{
  lines: ServerLogLine[]
  /** 上限由调用方决定，页脚要把它说出来 */
  maxLines: number
  /** WebSocket 是否连着，只影响光标闪不闪 */
  connected: boolean
  /** 分栏切换时面板可能被 v-show 藏起来，露出来要补一次滚动到底 */
  visible: boolean
  expanded: boolean
}>()

const emit = defineEmits<{ clear: []; 'toggle-expand': [] }>()

const iconButton = 'flex shrink-0 items-center rounded-md p-1.5 text-slate-400 transition-colors '
  + 'hover:bg-white/5 hover:text-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#60a5fa]/60'

/* ---------- 筛选 ---------- */

const level = ref('all')
const keyword = ref('')
/** 长行默认折行：中文报错在窄栏里横向滚动几乎没法读 */
const wrap = ref(true)

const levelFilters = [
  { value: 'all', label: '全部', tone: '' },
  { value: 'error', label: '错误', tone: 'text-[#f87171]' },
  { value: 'warning', label: '警告', tone: 'text-[#fbbf24]' },
  { value: 'info', label: '信息', tone: 'text-[#60a5fa]' },
  { value: 'debug', label: '调试', tone: '' },
]

/** logrus 的七个级别归到四档，够用且筛选按钮不至于排成一排 */
const bucketOf = (raw: string) => {
  switch (raw) {
    case 'panic': case 'fatal': case 'error': return 'error'
    case 'warning': case 'warn': return 'warning'
    case 'debug': case 'trace': return 'debug'
    default: return 'info'
  }
}

const short: Record<string, string> = { error: 'ERR', warning: 'WARN', debug: 'DBG', info: 'INFO' }
const rail: Record<string, string> = {
  error: 'bg-[#f87171]', warning: 'bg-[#fbbf24]', debug: 'bg-[#475569]', info: 'bg-[#3b82f6]',
}
const tone: Record<string, string> = {
  error: 'text-[#f87171]', warning: 'text-[#fbbf24]', debug: 'text-slate-600', info: 'text-[#60a5fa]',
}

const counts = computed(() => {
  const result: Record<string, number> = { all: props.lines.length, error: 0, warning: 0, info: 0, debug: 0 }
  for (const line of props.lines) result[bucketOf(line.level)]++
  return result
})

/** 搜索连字段一起匹配，否则按 traceId 之类的东西搜会一条都搜不到 */
const haystack = (line: ServerLogLine) =>
  line.fields ? line.message + ' ' + Object.entries(line.fields).map(pair => pair.join('=')).join(' ') : line.message

const visibleLines = computed(() => {
  const needle = keyword.value.trim().toLowerCase()
  return props.lines.filter(line =>
    (level.value === 'all' || bucketOf(line.level) === level.value)
    && (!needle || haystack(line).toLowerCase().includes(needle)))
})

const filtered = computed(() => level.value !== 'all' || keyword.value.trim() !== '')

const emptyText = computed(() => {
  if (!props.lines.length) return '暂无输出。服务器有动静时会实时出现在这里。'
  return filtered.value ? '没有匹配当前筛选的日志。' : '暂无输出。'
})

const resetFilters = () => {
  level.value = 'all'
  keyword.value = ''
}

/** 命中片段单独切出来，模板里给它加底色 —— 比 v-html 安全，也不用引 DOMPurify */
const highlight = (text: string) => {
  const needle = keyword.value.trim().toLowerCase()
  if (!needle) return [{ text, hit: false }]
  const parts: { text: string; hit: boolean }[] = []
  const lower = text.toLowerCase()
  let from = 0
  for (;;) {
    const at = lower.indexOf(needle, from)
    if (at < 0) break
    if (at > from) parts.push({ text: text.slice(from, at), hit: false })
    parts.push({ text: text.slice(at, at + needle.length), hit: true })
    from = at + needle.length
  }
  if (from < text.length) parts.push({ text: text.slice(from), hit: false })
  return parts
}

const formatFields = (fields: Record<string, string>) =>
  '  ' + Object.entries(fields).map(([key, value]) => `${key}=${value}`).join(' ')

/**
 * 一行的正文切片。字段跟正文走同一套高亮，否则按字段名搜出来的行
 * 会一个亮点都没有，看着像匹配错了；dim 只是让字段比正文暗一档。
 */
const lineParts = (line: ServerLogLine) => {
  const parts = highlight(line.message).map(part => ({ ...part, dim: false }))
  if (line.fields) {
    parts.push(...highlight(formatFields(line.fields)).map(part => ({ ...part, dim: true })))
  }
  return parts
}

/* ---------- 跟随滚动 ---------- */

const follow = ref(true)
const body = ref<HTMLElement | null>(null)
/** 暂停那一刻的最新序号，之后来的行都算成"没看到的" */
const pausedSeq = ref(0)

/**
 * 按序号比而不是按条数累加：日志到了上限就从头丢，
 * 长度不再变化但内容一直在动，累加出来的数字会永远停在 0。
 */
const missed = computed(() =>
  follow.value ? 0 : visibleLines.value.filter(line => line.seq > pausedSeq.value).length)

const setFollow = (value: boolean) => {
  follow.value = value
  pausedSeq.value = value || !props.lines.length ? 0 : props.lines[props.lines.length - 1].seq
}

const scrollToTail = () => {
  const el = body.value
  // 不用平滑滚动：日志刷得快时平滑滚动一直追不上，看着晕
  if (el) el.scrollTop = el.scrollHeight
}

const toBottom = () => {
  setFollow(true)
  scrollToTail()
}

/** 用户往上翻就停住，翻回底部再继续跟随 —— 别跟人抢滚动条 */
const onScroll = () => {
  const el = body.value
  // 面板被 v-show 藏起来时高度为 0，别把这个当成"用户翻到了顶部"
  if (!el || !el.clientHeight) return
  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 24
  if (atBottom !== follow.value) setFollow(atBottom)
}

const toggleFollow = () => {
  if (follow.value) setFollow(false)
  else toBottom()
}

// flush: 'post' 保证 DOM 已经长出新行，这时候量到的 scrollHeight 才是对的
watch(() => visibleLines.value.length, () => {
  if (follow.value) scrollToTail()
}, { flush: 'post' })

// 藏起来的面板 scrollHeight 是 0，重新露出来（或换了宽度）要补一次滚动到底
watch(() => [props.visible, props.expanded], () => {
  if (follow.value) requestAnimationFrame(scrollToTail)
})

/* ---------- 导出 ---------- */

/** 导出的是"当前看到的"而不是全部，否则筛完再导出会拿到一堆无关的行 */
const asText = () => visibleLines.value
  .map(line => `${line.time} ${line.level.toUpperCase()} ${line.message}${line.fields ? formatFields(line.fields) : ''}`)
  .join('\n')

const copyVisible = async () => {
  if (!visibleLines.value.length) return
  try {
    await navigator.clipboard.writeText(asText())
    notify.success(`已复制 ${visibleLines.value.length} 行`)
  } catch {
    // http 环境下没有 clipboard API，给条明确提示而不是静默失败
    notify.error('当前环境不允许访问剪贴板，请改用下载')
  }
}

const downloadVisible = () => {
  if (!visibleLines.value.length) return
  const url = URL.createObjectURL(new Blob([asText()], { type: 'text/plain;charset=utf-8' }))
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `server-log-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '')}.log`
  anchor.click()
  URL.revokeObjectURL(url)
}
</script>
