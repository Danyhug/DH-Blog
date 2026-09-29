<template>
  <div class="absolute top-32 left-1/2 z-[25] w-[360px] bg-[rgba(255,255,255,0.95)] backdrop-blur-[24px] rounded-2xl shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25)] border border-[rgba(255,255,255,0.2)] [transform:translateX(-50%)_translateX(8rem)]">
    <div class="p-5">
      <div class="flex items-center justify-between mb-4">
        <h3 class="popup-title text-[1rem] text-[#111827] m-0">创建分享链接</h3>
        <button class="p-2 rounded-md cursor-pointer transition-[background-color] duration-200 ease-[ease] hover:bg-[rgba(156,163,175,0.1)]" @click="$emit('close')">
          <XIcon class="w-4 h-4" />
        </button>
      </div>

      <div class="flex flex-col gap-4">
        <div class="flex items-center gap-3 p-3 bg-[rgba(249,250,251,0.5)] rounded-lg">
          <div class="w-8 h-8 bg-[#dbeafe] rounded-lg flex items-center justify-center">
            <component :is="file.icon || FileTextIcon" class="w-4 h-4 text-[#2563eb]" />
          </div>
          <div class="flex-1">
            <p class="text-[0.875rem] font-medium text-[#111827] m-0 [word-break:break-all]">{{ file.name }}</p>
            <p class="text-[0.75rem] text-[#6b7280] mt-1 mb-0">{{ file.size }}</p>
          </div>
        </div>

        <!-- 分享设置 -->
        <template v-if="!shareCreated">
          <div class="flex items-center justify-between">
            <span class="text-[0.875rem] text-[#374151]">设置密码</span>
            <label class="toggle-switch">
              <input type="checkbox" v-model="usePassword" />
              <span class="toggle-slider"></span>
            </label>
          </div>

          <div v-if="usePassword" class="flex flex-col gap-2">
            <input
              type="text"
              v-model="password"
              placeholder="请输入访问密码"
              class="url-input"
              maxlength="32"
            />
          </div>

          <div class="flex items-center justify-between">
            <span class="text-[0.875rem] text-[#374151]">过期时间</span>
            <select v-model="expireDays" class="select-input">
              <option :value="0">永不过期</option>
              <option :value="1">1天</option>
              <option :value="7">7天</option>
              <option :value="30">30天</option>
              <option :value="90">90天</option>
            </select>
          </div>

          <div class="flex items-center justify-between">
            <span class="text-[0.875rem] text-[#374151]">下载次数限制</span>
            <select v-model="maxDownloadCount" class="select-input">
              <option :value="0">不限制</option>
              <option :value="1">1次</option>
              <option :value="10">10次</option>
              <option :value="50">50次</option>
              <option :value="100">100次</option>
            </select>
          </div>

          <button class="create-btn" @click="createShareLink" :disabled="creating">
            {{ creating ? '创建中...' : '创建分享链接' }}
          </button>
        </template>

        <!-- 分享链接展示 -->
        <template v-else>
          <div class="flex flex-col gap-2">
            <label class="text-[0.75rem] text-[#6b7280]">分享链接</label>
            <div class="flex gap-2">
              <input
                type="text"
                :value="shareUrl"
                readonly
                class="url-input"
                ref="urlInput"
              />
              <button class="copy-btn" @click="copyUrl">复制</button>
            </div>
          </div>

          <div v-if="shareInfo?.password" class="flex flex-col gap-2">
            <label class="text-[0.75rem] text-[#6b7280]">访问密码</label>
            <div class="flex gap-2">
              <input
                type="text"
                :value="displayPassword"
                readonly
                class="url-input"
              />
              <button class="copy-btn" @click="copyPassword">复制</button>
            </div>
          </div>

          <div class="p-3 bg-[#f0fdf4] rounded-lg border border-[#bbf7d0]">
            <p v-if="expireDays > 0" class="m-0 text-[0.75rem] text-[#166534] flex items-center gap-2">
              <span class="text-[1rem]">⏰</span> {{ expireDays }}天后过期
            </p>
            <p v-else class="m-0 text-[0.75rem] text-[#166534] flex items-center gap-2">
              <span class="text-[1rem]">✨</span> 永不过期
            </p>
            <!-- 原 `.share-tips p + p { margin-top: .5rem }`：p1/p2 互斥，DOM 里的第二个 p 恒为这一条 -->
            <p v-if="maxDownloadCount > 0" class="m-0 mt-2 text-[0.75rem] text-[#166534] flex items-center gap-2">
              <span class="text-[1rem]">📥</span> 最多下载{{ maxDownloadCount }}次
            </p>
          </div>

          <button class="create-btn secondary" @click="resetShare">
            创建新的分享
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { XIcon, FileTextIcon } from '../utils/icons'
import type { FileItem } from '../utils/types/file'
import { createShare, generateShareLink, type ShareInfo } from '@/api/share'
import { notify } from '@/utils/notification'

// 定义属性
interface Props {
  file: FileItem
}

// 发出事件
const emit = defineEmits(['close'])

const props = defineProps<Props>()

// 状态
const usePassword = ref(false)
const password = ref('')
const displayPassword = ref('') // 用于显示的原始密码
const expireDays = ref(7)
const maxDownloadCount = ref(0)
const urlInput = ref<HTMLInputElement | null>(null)
const creating = ref(false)
const shareCreated = ref(false)
const shareInfo = ref<ShareInfo | null>(null)

// 计算属性
const shareUrl = computed(() => {
  if (shareInfo.value) {
    return generateShareLink(shareInfo.value.share_id)
  }
  return ''
})

// 创建分享链接
async function createShareLink() {
  if (!props.file.id) {
    notify.error('文件ID无效')
    return
  }

  if (usePassword.value && !password.value) {
    notify.warning('请输入访问密码')
    return
  }

  try {
    creating.value = true

    const data: any = {
      file_key: props.file.id
    }

    if (usePassword.value && password.value) {
      data.password = password.value
      displayPassword.value = password.value // 保存原始密码用于显示
    }

    if (expireDays.value > 0) {
      data.expire_days = expireDays.value
    }

    if (maxDownloadCount.value > 0) {
      data.max_download_count = maxDownloadCount.value
    }

    shareInfo.value = await createShare(data)
    shareCreated.value = true
    notify.success('分享链接创建成功')
  } catch (err: any) {
    notify.error(err.message || '创建分享链接失败')
  } finally {
    creating.value = false
  }
}

// 复制链接
function copyUrl() {
  if (urlInput.value) {
    urlInput.value.select()
    try {
      navigator.clipboard.writeText(shareUrl.value).then(() => {
        notify.success('链接已复制到剪贴板')
      })
    } catch (err) {
      document.execCommand('copy')
      notify.success('链接已复制到剪贴板')
    }
  }
}

// 复制密码
function copyPassword() {
  try {
    navigator.clipboard.writeText(displayPassword.value).then(() => {
      notify.success('密码已复制到剪贴板')
    })
  } catch (err) {
    notify.error('复制失败')
  }
}

// 重置分享表单
function resetShare() {
  shareCreated.value = false
  shareInfo.value = null
  usePassword.value = false
  password.value = ''
  displayPassword.value = ''
  expireDays.value = 7
  maxDownloadCount.value = 0
}
</script>

<style scoped>
/* 字重：style.less 全局规则是无层级的，压过工具类 */
.popup-title {
  font-weight: 600;
}

/* 重复 ≥2 次且带状态伪类的表单控件类，内联会撑长 class 串 */
.url-input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background: #f9fafb;
}

.url-input:focus {
  outline: none;
  border-color: #2563eb;
}

.copy-btn {
  background: #2563eb;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.2s;
  white-space: nowrap;
}

.copy-btn:hover {
  background: #1d4ed8;
}

/* 纯 CSS 状态机：input:checked + ... 兄弟选择器无工具类写法 */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  transition: 0.4s;
  border-radius: 24px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.4s;
  border-radius: 50%;
}

input:checked + .toggle-slider {
  background-color: #3b82f6;
}

input:checked + .toggle-slider:before {
  transform: translateX(20px);
}

.select-input {
  padding: 0.375rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background: #f9fafb;
  cursor: pointer;
}

.select-input:focus {
  outline: none;
  border-color: #2563eb;
}

.create-btn {
  width: 100%;
  padding: 0.75rem 1rem;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-top: 0.5rem;
}

.create-btn:hover:not(:disabled) {
  background: #1d4ed8;
}

.create-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.create-btn.secondary {
  background: #6b7280;
}

.create-btn.secondary:hover {
  background: #4b5563;
}
</style>
