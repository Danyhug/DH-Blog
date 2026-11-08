<template>
  <!-- 分享弹窗：与上传 / 重命名弹窗一样是「带遮罩的居中弹窗」。
       原先是一块浮在列表上方的面板，既没有遮罩也点不到外面关闭，文件多时容易被误认为页面的一角。 -->
  <div
    class="fixed inset-0 z-[1100] flex items-center justify-center p-4 bg-[rgba(0,0,0,0.5)] backdrop-blur-[2px] animate-fade-in"
    @click.self="requestClose"
  >
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="share-dialog-title"
      class="w-full max-w-[430px] max-h-[calc(100vh-2rem)] overflow-y-auto bg-white dark:bg-zinc-900 rounded-2xl shadow-[0_25px_50px_-12px_rgba(0,0,0,0.35)] animate-dialog-appear"
    >
      <div class="flex items-start justify-between gap-3 px-5 pt-5 pb-4">
        <div class="min-w-0">
          <h3 id="share-dialog-title" class="m-0 text-[1rem] font-semibold! text-[#111827] dark:text-zinc-100">
            {{ shareCreated ? '分享链接已创建' : '创建分享链接' }}
          </h3>
          <p class="m-0 mt-1 text-[0.75rem] text-[#6b7280] dark:text-zinc-400 truncate" :title="file.name">
            {{ shareCreated ? '扫码或复制链接发给对方' : '设置有效期和密码，然后生成链接' }}
          </p>
        </div>
        <button
          type="button"
          class="shrink-0 -mt-1 -mr-1 p-2 rounded-lg cursor-pointer text-[#6b7280] dark:text-zinc-400 transition-colors duration-200 hover:bg-[rgba(156,163,175,0.12)] hover:text-[#111827] dark:hover:text-zinc-100 focus-visible:outline-2 focus-visible:outline-[#2a8aff]"
          aria-label="关闭"
          @click="requestClose"
        >
          <XIcon class="w-4 h-4" />
        </button>
      </div>

      <div class="px-5 pb-5 flex flex-col gap-4">
        <!-- 文件摘要：标题只说「在做什么」，具体分享哪个文件放在这里 -->
        <div class="flex items-center gap-3 p-3 rounded-xl bg-[#f6f8fb] dark:bg-white/5">
          <div class="flex items-center justify-center w-9 h-9 shrink-0 rounded-lg bg-[#e3edff] dark:bg-blue-500/15">
            <component :is="fileIcon" class="w-4 h-4 text-[#2a8aff] dark:text-blue-400" />
          </div>
          <div class="flex-1 min-w-0">
            <p class="m-0 text-[0.875rem] font-medium text-[#111827] dark:text-zinc-100 truncate" :title="file.name">{{ file.name }}</p>
            <p class="m-0 mt-0.5 text-[0.75rem] text-[#6b7280] dark:text-zinc-400">{{ file.size }}</p>
          </div>
        </div>

        <!-- 创建前：分享设置 -->
        <template v-if="!shareCreated">
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <p class="m-0 text-[0.8125rem] text-[#374151] dark:text-zinc-200">访问密码</p>
              <p class="m-0 mt-0.5 text-[0.6875rem] text-[#9ca3af] dark:text-zinc-500">开启后需要密码才能查看和下载</p>
            </div>
            <button
              type="button"
              role="switch"
              :aria-checked="usePassword"
              aria-label="设置访问密码"
              class="relative w-[44px] h-6 shrink-0 rounded-full border-none cursor-pointer transition-colors duration-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#2a8aff]"
              :class="usePassword ? 'bg-[#2a8aff]' : 'bg-[#d1d5db] dark:bg-zinc-600'"
              @click="togglePassword"
            >
              <span
                class="absolute top-[3px] left-[3px] size-[18px] rounded-full bg-white shadow-sm transition-transform duration-200"
                :class="usePassword ? 'translate-x-5' : 'translate-x-0'"
              ></span>
            </button>
          </div>

          <div v-if="usePassword" class="flex items-center gap-2">
            <input
              v-model="password"
              type="text"
              maxlength="32"
              placeholder="输入或随机生成密码"
              aria-label="访问密码"
              class="flex-1 min-w-0 py-2 px-3 rounded-lg border border-[#e5e7eb] dark:border-white/15 text-[0.875rem] bg-[#f9fafb] dark:bg-white/5 text-[#111827] dark:text-zinc-100 placeholder:text-[#9ca3af] dark:placeholder:text-zinc-500 focus:outline-none focus:border-[#2a8aff] focus:bg-white dark:focus:bg-white/10"
            />
            <button
              type="button"
              title="随机生成密码"
              aria-label="随机生成密码"
              class="shrink-0 p-2 rounded-lg border border-[#e5e7eb] dark:border-white/15 bg-white dark:bg-zinc-900 text-[#6b7280] dark:text-zinc-400 cursor-pointer transition-colors duration-200 hover:text-[#2a8aff] hover:border-[#2a8aff] dark:hover:text-blue-400"
              @click="generateRandomPassword"
            >
              <RefreshIcon class="w-4 h-4" />
            </button>
          </div>

          <div class="flex flex-col gap-2">
            <div class="flex items-baseline justify-between gap-2">
              <p class="m-0 text-[0.8125rem] text-[#374151] dark:text-zinc-200">有效期</p>
              <p class="m-0 text-[0.6875rem] text-[#9ca3af] dark:text-zinc-500">{{ expireHint }}</p>
            </div>
            <div class="flex flex-wrap gap-2" role="group" aria-label="有效期">
              <button
                v-for="option in expireOptions"
                :key="option.value"
                type="button"
                :aria-pressed="expireDays === option.value"
                :class="segmentClass(expireDays === option.value)"
                @click="expireDays = option.value"
              >
                {{ option.label }}
              </button>
            </div>
          </div>

          <div class="flex flex-col gap-2">
            <div class="flex items-baseline justify-between gap-2">
              <p class="m-0 text-[0.8125rem] text-[#374151] dark:text-zinc-200">下载次数限制</p>
              <p class="m-0 text-[0.6875rem] text-[#9ca3af] dark:text-zinc-500">{{ downloadLimitHint }}</p>
            </div>
            <div class="flex flex-wrap gap-2" role="group" aria-label="下载次数限制">
              <button
                v-for="option in downloadOptions"
                :key="option.value"
                type="button"
                :aria-pressed="maxDownloadCount === option.value"
                :class="segmentClass(maxDownloadCount === option.value)"
                @click="maxDownloadCount = option.value"
              >
                {{ option.label }}
              </button>
            </div>
          </div>

          <button
            type="button"
            class="w-full mt-1 py-3 px-4 rounded-xl text-[0.9375rem] font-medium text-white border-none cursor-pointer bg-[#2a8aff] transition-colors duration-200 hover:bg-[#1a7aef] disabled:opacity-60 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#2a8aff]"
            :disabled="creating"
            @click="createShareLink"
          >
            {{ creating ? '创建中…' : '创建分享链接' }}
          </button>
        </template>

        <!-- 创建后：二维码 + 链接 + 凭据 -->
        <template v-else>
          <div class="flex items-center gap-2 p-3 rounded-xl bg-[#f0fdf4] dark:bg-green-500/10 border border-[#bbf7d0] dark:border-green-500/25">
            <CheckIcon class="w-4 h-4 shrink-0 text-[#16a34a] dark:text-green-400" />
            <p class="m-0 text-[0.75rem] text-[#166534] dark:text-green-400">链接已生成，扫一扫就能在手机上打开</p>
          </div>

          <!-- 二维码固定白底：深色模式下反色的码有些手机扫不出来 -->
          <div class="flex flex-col items-center gap-2">
            <div class="p-3 bg-white rounded-2xl ring-1 ring-black/5 shadow-[0_2px_12px_rgba(0,0,0,0.06)]">
              <QrcodeVue
                :value="shareUrl"
                :size="160"
                :margin="0"
                level="M"
                render-as="svg"
                background="#ffffff"
                foreground="#111827"
                role="img"
                aria-label="分享链接二维码"
              />
            </div>
            <p class="m-0 text-[0.6875rem] text-[#9ca3af] dark:text-zinc-500">分享页链接的二维码</p>
          </div>

          <div class="flex flex-col gap-1.5">
            <label class="text-[0.6875rem] text-[#6b7280] dark:text-zinc-400" for="share-link-input">分享链接</label>
            <div class="flex gap-2">
              <input
                id="share-link-input"
                :value="shareUrl"
                type="text"
                readonly
                class="flex-1 min-w-0 py-2 px-3 rounded-lg border border-[#e5e7eb] dark:border-white/15 text-[0.8125rem] bg-[#f9fafb] dark:bg-white/5 text-[#374151] dark:text-zinc-300 focus:outline-none focus:border-[#2a8aff]"
              />
              <button
                type="button"
                class="shrink-0 py-2 px-3 rounded-lg border-none bg-[#2a8aff] text-white text-[0.8125rem] cursor-pointer transition-colors duration-200 hover:bg-[#1a7aef]"
                @click="copyText(shareUrl, '链接已复制到剪贴板')"
              >
                复制
              </button>
            </div>
          </div>

          <div v-if="createdPassword" class="flex flex-col gap-1.5">
            <label class="text-[0.6875rem] text-[#6b7280] dark:text-zinc-400" for="share-password-input">访问密码</label>
            <div class="flex gap-2">
              <input
                id="share-password-input"
                :value="createdPassword"
                type="text"
                readonly
                class="flex-1 min-w-0 py-2 px-3 rounded-lg border border-[#e5e7eb] dark:border-white/15 text-[0.8125rem] bg-[#f9fafb] dark:bg-white/5 text-[#374151] dark:text-zinc-300 focus:outline-none focus:border-[#2a8aff]"
              />
              <button
                type="button"
                class="shrink-0 py-2 px-3 rounded-lg border-none bg-[#2a8aff] text-white text-[0.8125rem] cursor-pointer transition-colors duration-200 hover:bg-[#1a7aef]"
                @click="copyText(createdPassword, '密码已复制到剪贴板')"
              >
                复制
              </button>
            </div>
          </div>

          <!-- 把这次分享的凭据摊开写清楚，不用回看上面选了什么 -->
          <div class="flex flex-wrap gap-2">
            <span class="px-2 py-1 rounded-md text-[0.6875rem] bg-[#f3f4f6] dark:bg-white/5 text-[#4b5563] dark:text-zinc-400">
              有效期：{{ createdExpireLabel }}
            </span>
            <span class="px-2 py-1 rounded-md text-[0.6875rem] bg-[#f3f4f6] dark:bg-white/5 text-[#4b5563] dark:text-zinc-400">
              下载：{{ createdDownloadLabel }}
            </span>
          </div>

          <button
            v-if="createdPassword"
            type="button"
            class="w-full py-2.5 rounded-lg border border-[#e5e7eb] dark:border-white/15 bg-white dark:bg-zinc-900 text-[0.8125rem] text-[#4b5563] dark:text-zinc-300 cursor-pointer transition-colors duration-200 hover:bg-[#f5f5f5] dark:hover:bg-white/5"
            @click="copyText(shareUrl + '\n密码：' + createdPassword, '链接和密码已一起复制')"
          >
            复制链接及密码
          </button>

          <div class="flex gap-2 mt-1">
            <button
              type="button"
              class="flex-1 py-2.5 rounded-lg border border-[#e5e7eb] dark:border-white/15 bg-white dark:bg-zinc-900 text-[0.8125rem] text-[#4b5563] dark:text-zinc-300 cursor-pointer transition-colors duration-200 hover:bg-[#f5f5f5] dark:hover:bg-white/5"
              @click="resetShare"
            >
              再建一个
            </button>
            <button
              type="button"
              class="flex-1 py-2.5 rounded-lg border-none bg-[#2a8aff] text-white text-[0.8125rem] cursor-pointer transition-colors duration-200 hover:bg-[#1a7aef]"
              @click="requestClose"
            >
              完成
            </button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, markRaw, onBeforeUnmount, onMounted, ref, toRaw } from 'vue'
import QrcodeVue from 'qrcode.vue'
import { CheckIcon, FileTextIcon, RefreshIcon, XIcon } from '../utils/icons'
import type { FileItem } from '../utils/types/file'
import { createShare, generateShareLink, type ShareInfo } from '@/api/share'
import { notify } from '@/utils/notification'

interface Props {
  file: FileItem
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()

// 有效期与下载上限用分段按钮而不是原生 select：选项数量固定，
// 点一下就落到目标值，不必先展开下拉再找一遍。
const expireOptions = [
  { label: '永久', value: 0 },
  { label: '1 天', value: 1 },
  { label: '7 天', value: 7 },
  { label: '30 天', value: 30 },
  { label: '90 天', value: 90 },
]

const downloadOptions = [
  { label: '不限', value: 0 },
  { label: '1 次', value: 1 },
  { label: '10 次', value: 10 },
  { label: '50 次', value: 50 },
  { label: '100 次', value: 100 },
]

const usePassword = ref(false)
const password = ref('')
const expireDays = ref(7)
const maxDownloadCount = ref(0)
const creating = ref(false)
const shareCreated = ref(false)
const shareInfo = ref<ShareInfo | null>(null)
// 服务端返回的密码可能被省略，创建时提交的原始密码留一份兜底
const submittedPassword = ref('')

const shareUrl = computed(() => (shareInfo.value ? generateShareLink(shareInfo.value.share_id) : ''))
// file 是父组件 ref 里的深层响应式对象，直接读 icon 会拿到组件的 reactive 代理，
// Vue 会警告「Component that was made a reactive object」，所以取回原始组件再 markRaw
const fileIcon = computed(() => {
  const icon = props.file.icon
  return icon ? markRaw(toRaw(icon)) : FileTextIcon
})
const createdPassword = computed(() => shareInfo.value?.password || submittedPassword.value)

const expireHint = computed(() => (expireDays.value > 0 ? `${expireDays.value} 天后自动失效` : '创建后长期有效'))
const downloadLimitHint = computed(() =>
  maxDownloadCount.value > 0 ? `最多下载 ${maxDownloadCount.value} 次` : '不限制，可反复下载'
)

const createdExpireLabel = computed(() => {
  const expireAt = shareInfo.value?.expire_at
  return expireAt ? `至 ${expireAt.slice(0, 10)}` : '永久'
})

const createdDownloadLabel = computed(() => {
  const limit = shareInfo.value?.max_download_count
  return limit ? `最多 ${limit} 次` : '不限次数'
})

function segmentClass(active: boolean) {
  // 选中态与未选中态是互斥的两套类，不能叠加：Tailwind 里 bg-transparent 排在
  // bg-[#2a8aff] 之后，同优先级会把选中态的底色盖掉。
  const base = 'px-3 py-1.5 rounded-lg text-[0.8125rem] cursor-pointer border transition-colors duration-200 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-[#2a8aff]'
  return active
    ? `${base} bg-[#2a8aff] text-white border-transparent font-medium`
    : `${base} bg-transparent text-[#4b5563] dark:text-zinc-400 border-[#e5e7eb] dark:border-white/15 hover:bg-[rgba(42,138,255,0.08)] hover:text-[#2a8aff] dark:hover:text-blue-400`
}

function togglePassword() {
  usePassword.value = !usePassword.value
  if (usePassword.value && !password.value) {
    generateRandomPassword()
  }
}

function generateRandomPassword() {
  // 去掉 0/o/1/l/i 这类容易读错抄错的字符，密码是要靠人念给对方的
  const alphabet = 'abcdefghjkmnpqrstuvwxyz23456789'
  const randomBytes = new Uint8Array(6)
  crypto.getRandomValues(randomBytes)
  password.value = Array.from(randomBytes, (byte) => alphabet[byte % alphabet.length]).join('')
}

async function createShareLink() {
  if (!props.file.id) {
    notify.error('文件ID无效')
    return
  }

  if (usePassword.value && !password.value.trim()) {
    notify.warning('请输入访问密码')
    return
  }

  try {
    creating.value = true

    const request = {
      file_key: props.file.id,
      ...(usePassword.value ? { password: password.value.trim() } : {}),
      ...(expireDays.value > 0 ? { expire_days: expireDays.value } : {}),
      ...(maxDownloadCount.value > 0 ? { max_download_count: maxDownloadCount.value } : {}),
    }

    submittedPassword.value = request.password ?? ''
    shareInfo.value = await createShare(request)
    shareCreated.value = true
    notify.success('分享链接创建成功')
  } catch (err: any) {
    notify.error(err?.message || '创建分享链接失败')
  } finally {
    creating.value = false
  }
}

async function copyText(value: string, successMessage: string) {
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    notify.success(successMessage)
  } catch {
    // 非安全上下文（http、内嵌 webview）没有 clipboard API，退回临时 textarea
    try {
      const helper = document.createElement('textarea')
      helper.value = value
      helper.setAttribute('readonly', '')
      helper.style.position = 'fixed'
      helper.style.opacity = '0'
      document.body.appendChild(helper)
      helper.select()
      document.execCommand('copy')
      document.body.removeChild(helper)
      notify.success(successMessage)
    } catch {
      notify.error('复制失败，请手动选中复制')
    }
  }
}

function resetShare() {
  shareCreated.value = false
  shareInfo.value = null
  submittedPassword.value = ''
  usePassword.value = false
  password.value = ''
  expireDays.value = 7
  maxDownloadCount.value = 0
}

function requestClose() {
  emit('close')
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.stopPropagation()
    requestClose()
  }
}

onMounted(() => document.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => document.removeEventListener('keydown', handleKeydown))
</script>
