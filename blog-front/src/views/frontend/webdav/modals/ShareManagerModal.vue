<template>
  <div class="absolute top-16 left-1/2 z-[25] w-[min(560px,calc(100%-2rem))] [transform:translateX(-50%)]">
    <div class="bg-[rgba(255,255,255,0.97)] dark:bg-zinc-900/95 backdrop-blur-[24px] rounded-2xl shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25)] border border-[rgba(255,255,255,0.2)] p-5">
      <div class="flex items-center justify-between mb-4">
        <h3 class="font-semibold! flex items-center gap-2 m-0 text-[1rem] text-[#111827] dark:text-zinc-100 truncate">
          <button v-if="logsFor" class="px-2 py-[0.35rem] rounded-md cursor-pointer text-[#6b7280] dark:text-zinc-400 text-[1rem] transition-[background-color] duration-200 ease-[ease] hover:bg-[rgba(156,163,175,0.12)]" @click="closeLogs">←</button>
          {{ logsFor ? `访问日志 · ${logsFor.file_name || logsFor.file_key}` : '我的分享' }}
        </h3>
        <button class="px-2 py-[0.35rem] rounded-md cursor-pointer text-[#6b7280] dark:text-zinc-400 text-[1rem] transition-[background-color] duration-200 ease-[ease] hover:bg-[rgba(156,163,175,0.12)]" @click="$emit('close')">
          <XIcon class="w-4 h-4" />
        </button>
      </div>

      <!-- 分享列表 -->
      <div v-if="!logsFor" class="max-h-[26rem] overflow-y-auto">
        <p v-if="loading" class="text-[#9ca3af] dark:text-zinc-500 text-[0.8rem] text-center py-8 px-0 m-0">加载中…</p>
        <p v-else-if="shares.length === 0" class="text-[#9ca3af] dark:text-zinc-500 text-[0.8rem] text-center py-8 px-0 m-0">还没有创建过分享链接</p>

        <ul v-else class="list-none m-0 p-0">
          <li v-for="share in shares" :key="share.id" class="py-3 border-b border-[rgba(229,231,235,0.8)] dark:border-white/10 last:border-b-0">
            <div class="flex items-center justify-between gap-2">
              <p class="m-0 text-[0.875rem] font-medium truncate" :class="share.file_missing ? 'text-[#9ca3af] dark:text-zinc-500 italic' : 'text-[#111827] dark:text-zinc-100'">
                {{ share.file_missing ? '文件已删除' : share.file_name }}
              </p>
              <div class="flex gap-1 shrink-0">
                <span v-if="share.has_password" class="text-[0.6875rem] py-[0.1rem] px-[0.4rem] rounded whitespace-nowrap bg-[#eef2ff] dark:bg-indigo-500/15 text-[#4f46e5] dark:text-indigo-400">密码</span>
                <span v-if="share.is_expired" class="text-[0.6875rem] py-[0.1rem] px-[0.4rem] rounded whitespace-nowrap bg-[#fee2e2] dark:bg-red-500/15 text-[#b91c1c] dark:text-red-400">已过期</span>
                <span v-else-if="share.expire_at" class="text-[0.6875rem] py-[0.1rem] px-[0.4rem] rounded whitespace-nowrap bg-[#eef2ff] dark:bg-indigo-500/15 text-[#4f46e5] dark:text-indigo-400">{{ formatExpire(share.expire_at) }} 到期</span>
                <span v-else class="text-[0.6875rem] py-[0.1rem] px-[0.4rem] rounded whitespace-nowrap bg-[#fef3c7] dark:bg-amber-500/15 text-[#b45309] dark:text-amber-400">永不过期</span>
              </div>
            </div>

            <p class="mt-1 mb-2 text-[0.75rem] text-[#6b7280] dark:text-zinc-400">
              {{ share.file_missing ? '—' : formatFileSize(share.file_size) }}
              · 浏览 {{ share.view_count }}
              · 下载 {{ share.download_count }}<template v-if="share.max_download_count">/{{ share.max_download_count }}</template>
              · {{ share.create_time }}
            </p>

            <div class="flex gap-3">
              <button class="bg-transparent border-none p-0 text-[0.75rem] cursor-pointer [&:hover:not(:disabled)]:underline disabled:cursor-not-allowed text-[#2563eb] dark:text-blue-400 disabled:text-[#9ca3af]" @click="copyLink(share)">复制链接</button>
              <button class="bg-transparent border-none p-0 text-[0.75rem] cursor-pointer [&:hover:not(:disabled)]:underline disabled:cursor-not-allowed text-[#2563eb] dark:text-blue-400 disabled:text-[#9ca3af]" @click="openLogs(share)">访问日志</button>
              <button
                class="bg-transparent border-none p-0 text-[0.75rem] cursor-pointer [&:hover:not(:disabled)]:underline disabled:cursor-not-allowed text-[#dc2626] dark:text-red-400"
                :disabled="revoking[share.id]"
                @click="revoke(share)"
              >
                {{ confirmingId === share.id ? '确认撤销？' : '撤销' }}
              </button>
            </div>
          </li>
        </ul>

        <div v-if="totalPages > 1" class="flex items-center justify-center gap-4 pt-3 border-t border-[rgba(229,231,235,0.8)] dark:border-white/10 mt-2">
          <button class="bg-transparent border-none p-0 text-[0.75rem] cursor-pointer [&:hover:not(:disabled)]:underline disabled:cursor-not-allowed text-[#2563eb] dark:text-blue-400 disabled:text-[#9ca3af]" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
          <span class="text-[0.75rem] text-[#6b7280] dark:text-zinc-400">{{ page }} / {{ totalPages }}</span>
          <button class="bg-transparent border-none p-0 text-[0.75rem] cursor-pointer [&:hover:not(:disabled)]:underline disabled:cursor-not-allowed text-[#2563eb] dark:text-blue-400 disabled:text-[#9ca3af]" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
        </div>
      </div>

      <!-- 访问日志 -->
      <div v-else class="max-h-[26rem] overflow-y-auto">
        <p v-if="logsLoading" class="text-[#9ca3af] dark:text-zinc-500 text-[0.8rem] text-center py-8 px-0 m-0">加载中…</p>
        <p v-else-if="logs.length === 0" class="text-[#9ca3af] dark:text-zinc-500 text-[0.8rem] text-center py-8 px-0 m-0">这个分享还没有被访问过</p>

        <ul v-else class="list-none m-0 p-0">
          <li v-for="log in logs" :key="log.id" class="flex items-center gap-3 py-2 border-b border-[rgba(229,231,235,0.6)] dark:border-white/10 last:border-b-0 text-[0.75rem] text-[#4b5563] dark:text-zinc-400">
            <span class="shrink-0 py-[0.1rem] px-[0.4rem] rounded" :class="log.action_type === 'download' ? 'bg-[#dcfce7] dark:bg-green-500/15 text-[#166534] dark:text-green-400' : 'bg-[#eef2ff] dark:bg-indigo-500/15 text-[#4f46e5] dark:text-indigo-400'">{{ log.action_type === 'download' ? '下载' : '浏览' }}</span>
            <span class="flex-1 [font-family:Monaco,Menlo,monospace]">{{ log.ip }}</span>
            <span class="shrink-0 text-[#9ca3af] dark:text-zinc-500">{{ log.create_time }}</span>
          </li>
        </ul>

        <div v-if="logsTotalPages > 1" class="flex items-center justify-center gap-4 pt-3 border-t border-[rgba(229,231,235,0.8)] dark:border-white/10 mt-2">
          <button class="bg-transparent border-none p-0 text-[0.75rem] cursor-pointer [&:hover:not(:disabled)]:underline disabled:cursor-not-allowed text-[#2563eb] dark:text-blue-400 disabled:text-[#9ca3af]" :disabled="logsPage <= 1" @click="goLogsPage(logsPage - 1)">上一页</button>
          <span class="text-[0.75rem] text-[#6b7280] dark:text-zinc-400">{{ logsPage }} / {{ logsTotalPages }}</span>
          <button class="bg-transparent border-none p-0 text-[0.75rem] cursor-pointer [&:hover:not(:disabled)]:underline disabled:cursor-not-allowed text-[#2563eb] dark:text-blue-400 disabled:text-[#9ca3af]" :disabled="logsPage >= logsTotalPages" @click="goLogsPage(logsPage + 1)">下一页</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { XIcon } from '../utils/icons'
import {
  listShares, deleteShare, getShareAccessLogs,
  formatFileSize, generateShareLink,
  type ShareSummary, type ShareAccessLog
} from '@/api/share'
import { notify } from '@/utils/notification'

defineEmits(['close'])

const PAGE_SIZE = 8

const shares = ref<ShareSummary[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const revoking = ref<Record<number, boolean>>({})
// 撤销是不可逆操作，用两段式点击代替弹窗（网盘这块没有引入 Element Plus）
const confirmingId = ref<number | null>(null)
let confirmTimer: ReturnType<typeof setTimeout> | null = null

const logsFor = ref<ShareSummary | null>(null)
const logs = ref<ShareAccessLog[]>([])
const logsTotal = ref(0)
const logsPage = ref(1)
const logsLoading = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
const logsTotalPages = computed(() => Math.max(1, Math.ceil(logsTotal.value / PAGE_SIZE)))

async function loadShares() {
  loading.value = true
  try {
    const result = await listShares(page.value, PAGE_SIZE)
    shares.value = result.list || []
    total.value = result.total || 0
    // 删到当前页空了就回退一页
    if (shares.value.length === 0 && page.value > 1) {
      page.value -= 1
      await loadShares()
    }
  } catch {
    // 失败原因由 axios 拦截器提示
  } finally {
    loading.value = false
  }
}

function goPage(next: number) {
  page.value = next
  resetConfirm()
  loadShares()
}

function copyLink(share: ShareSummary) {
  const url = generateShareLink(share.share_id)
  navigator.clipboard.writeText(url)
    .then(() => notify.success('链接已复制到剪贴板'))
    .catch(() => notify.error('复制失败，请手动复制'))
}

function resetConfirm() {
  confirmingId.value = null
  if (confirmTimer) {
    clearTimeout(confirmTimer)
    confirmTimer = null
  }
}

async function revoke(share: ShareSummary) {
  if (confirmingId.value !== share.id) {
    // 第一次点击只进入确认态，3 秒无操作自动取消
    resetConfirm()
    confirmingId.value = share.id
    confirmTimer = setTimeout(resetConfirm, 3000)
    return
  }
  resetConfirm()

  revoking.value[share.id] = true
  try {
    await deleteShare(share.id)
    notify.success('分享已撤销')
    await loadShares()
  } catch {
    // 失败原因由 axios 拦截器提示
  } finally {
    revoking.value[share.id] = false
  }
}

async function loadLogs() {
  if (!logsFor.value) return
  logsLoading.value = true
  try {
    const result = await getShareAccessLogs(logsFor.value.id, logsPage.value, PAGE_SIZE)
    logs.value = result.list || []
    logsTotal.value = result.total || 0
  } catch {
    // 失败原因由 axios 拦截器提示
  } finally {
    logsLoading.value = false
  }
}

function openLogs(share: ShareSummary) {
  resetConfirm()
  logsFor.value = share
  logsPage.value = 1
  logs.value = []
  loadLogs()
}

function closeLogs() {
  logsFor.value = null
}

function goLogsPage(next: number) {
  logsPage.value = next
  loadLogs()
}

function formatExpire(value: string) {
  return value.slice(0, 10)
}

onMounted(loadShares)
</script>
