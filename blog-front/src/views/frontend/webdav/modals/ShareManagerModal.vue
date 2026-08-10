<template>
  <div class="share-manager">
    <div class="panel">
      <div class="panel-header">
        <h3 class="panel-title">
          <button v-if="logsFor" class="back-btn" @click="closeLogs">←</button>
          {{ logsFor ? `访问日志 · ${logsFor.file_name || logsFor.file_key}` : '我的分享' }}
        </h3>
        <button class="close-btn" @click="$emit('close')">
          <XIcon class="icon-sm" />
        </button>
      </div>

      <!-- 分享列表 -->
      <div v-if="!logsFor" class="panel-body">
        <p v-if="loading" class="hint">加载中…</p>
        <p v-else-if="shares.length === 0" class="hint">还没有创建过分享链接</p>

        <ul v-else class="share-list">
          <li v-for="share in shares" :key="share.id" class="share-item">
            <div class="share-main">
              <p class="share-name" :class="{ missing: share.file_missing }">
                {{ share.file_missing ? '文件已删除' : share.file_name }}
              </p>
              <div class="badges">
                <span v-if="share.has_password" class="badge">密码</span>
                <span v-if="share.is_expired" class="badge danger">已过期</span>
                <span v-else-if="share.expire_at" class="badge">{{ formatExpire(share.expire_at) }} 到期</span>
                <span v-else class="badge warn">永不过期</span>
              </div>
            </div>

            <p class="share-meta">
              {{ share.file_missing ? '—' : formatFileSize(share.file_size) }}
              · 浏览 {{ share.view_count }}
              · 下载 {{ share.download_count }}<template v-if="share.max_download_count">/{{ share.max_download_count }}</template>
              · {{ share.create_time }}
            </p>

            <div class="share-actions">
              <button class="link-btn" @click="copyLink(share)">复制链接</button>
              <button class="link-btn" @click="openLogs(share)">访问日志</button>
              <button
                class="link-btn danger"
                :disabled="revoking[share.id]"
                @click="revoke(share)"
              >
                {{ confirmingId === share.id ? '确认撤销？' : '撤销' }}
              </button>
            </div>
          </li>
        </ul>

        <div v-if="totalPages > 1" class="pager">
          <button class="link-btn" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
          <span class="pager-info">{{ page }} / {{ totalPages }}</span>
          <button class="link-btn" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
        </div>
      </div>

      <!-- 访问日志 -->
      <div v-else class="panel-body">
        <p v-if="logsLoading" class="hint">加载中…</p>
        <p v-else-if="logs.length === 0" class="hint">这个分享还没有被访问过</p>

        <ul v-else class="log-list">
          <li v-for="log in logs" :key="log.id" class="log-item">
            <span class="log-action" :class="log.action_type">{{ log.action_type === 'download' ? '下载' : '浏览' }}</span>
            <span class="log-ip">{{ log.ip }}</span>
            <span class="log-time">{{ log.create_time }}</span>
          </li>
        </ul>

        <div v-if="logsTotalPages > 1" class="pager">
          <button class="link-btn" :disabled="logsPage <= 1" @click="goLogsPage(logsPage - 1)">上一页</button>
          <span class="pager-info">{{ logsPage }} / {{ logsTotalPages }}</span>
          <button class="link-btn" :disabled="logsPage >= logsTotalPages" @click="goLogsPage(logsPage + 1)">下一页</button>
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

<style scoped>
.share-manager {
  position: absolute;
  top: 4rem;
  left: 50%;
  transform: translateX(-50%);
  width: min(560px, calc(100% - 2rem));
  z-index: 25;
}

.panel {
  background: rgba(255, 255, 255, 0.97);
  backdrop-filter: blur(24px);
  border-radius: 1rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.2);
  padding: 1.25rem;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.back-btn,
.close-btn {
  background: none;
  border: none;
  padding: 0.35rem 0.5rem;
  border-radius: 0.375rem;
  cursor: pointer;
  color: #6b7280;
  font-size: 1rem;
  transition: background-color 0.2s;
}

.back-btn:hover,
.close-btn:hover {
  background: rgba(156, 163, 175, 0.12);
}

.panel-body {
  max-height: 26rem;
  overflow-y: auto;
}

.hint {
  color: #9ca3af;
  font-size: 0.8rem;
  text-align: center;
  padding: 2rem 0;
  margin: 0;
}

.share-list,
.log-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.share-item {
  padding: 0.75rem 0;
  border-bottom: 1px solid rgba(229, 231, 235, 0.8);
}

.share-item:last-child {
  border-bottom: none;
}

.share-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.share-name {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 500;
  color: #111827;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.share-name.missing {
  color: #9ca3af;
  font-style: italic;
}

.badges {
  display: flex;
  gap: 0.25rem;
  flex-shrink: 0;
}

.badge {
  font-size: 0.6875rem;
  padding: 0.1rem 0.4rem;
  border-radius: 0.25rem;
  background: #eef2ff;
  color: #4f46e5;
  white-space: nowrap;
}

.badge.danger {
  background: #fee2e2;
  color: #b91c1c;
}

.badge.warn {
  background: #fef3c7;
  color: #b45309;
}

.share-meta {
  margin: 0.25rem 0 0.5rem 0;
  font-size: 0.75rem;
  color: #6b7280;
}

.share-actions {
  display: flex;
  gap: 0.75rem;
}

.link-btn {
  background: none;
  border: none;
  padding: 0;
  font-size: 0.75rem;
  color: #2563eb;
  cursor: pointer;
}

.link-btn:hover:not(:disabled) {
  text-decoration: underline;
}

.link-btn:disabled {
  color: #9ca3af;
  cursor: not-allowed;
}

.link-btn.danger {
  color: #dc2626;
}

.pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(229, 231, 235, 0.8);
  margin-top: 0.5rem;
}

.pager-info {
  font-size: 0.75rem;
  color: #6b7280;
}

.log-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0;
  border-bottom: 1px solid rgba(229, 231, 235, 0.6);
  font-size: 0.75rem;
  color: #4b5563;
}

.log-item:last-child {
  border-bottom: none;
}

.log-action {
  flex-shrink: 0;
  padding: 0.1rem 0.4rem;
  border-radius: 0.25rem;
  background: #eef2ff;
  color: #4f46e5;
}

.log-action.download {
  background: #dcfce7;
  color: #166534;
}

.log-ip {
  flex: 1;
  font-family: 'Monaco', 'Menlo', monospace;
}

.log-time {
  flex-shrink: 0;
  color: #9ca3af;
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}
</style>
