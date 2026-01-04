<template>
  <div class="w-full min-h-screen flex flex-col bg-[#f8f9fa] animate-[fade-in_0.3s_ease] relative" :class="{ 'share-mode': shareMode }">
    <!-- 顶部导航栏 -->
    <div class="sticky top-0 z-10 backdrop-blur-[20px] bg-white/90 border-b border-black/5 shadow-[0_4px_20px_rgba(0,0,0,0.05)] py-2.5">
      <div class="max-w-[1400px] mx-auto flex justify-between items-center px-6 py-3 max-md:flex-col max-md:gap-3 max-md:px-3">
      <div class="flex-[2] max-md:w-full">
        <!-- 分享模式：显示Logo -->
        <template v-if="shareMode">
          <a href="/" class="no-underline">
            <span class="text-lg font-semibold text-[#333]">DH-Blog</span>
          </a>
        </template>
        <!-- 普通模式：显示面包屑 -->
        <template v-else>
          <div class="flex items-center gap-2 text-sm bg-[#f8f9fa]/90 px-4 py-2.5 rounded-[50px] shadow-[0_2px_8px_rgba(0,0,0,0.03)] backdrop-blur-[4px]">
            <HomeIcon class="cursor-pointer text-[#555] w-4 h-4 transition-all duration-200 hover:text-[var(--color-blue)] hover:scale-110" @click="handleNavigateToRoot" />
            <template v-if="pathSegments.length > 0">
              <ChevronRightIcon class="text-[#aaa] w-3 h-3" />
              <template v-for="(segment, index) in pathSegments" :key="index">
                <span
                  class="cursor-pointer text-[#555] font-medium px-2 py-0.5 rounded transition-all duration-200 hover:text-[var(--color-blue)] hover:bg-[rgba(56,161,219,0.1)]"
                  @click="handleNavigateToPathSegment(index)"
                >{{ segment.name }}</span>
                <ChevronRightIcon v-if="index < pathSegments.length - 1" class="text-[#aaa] w-3 h-3" />
              </template>
            </template>
            <span v-else class="cursor-pointer text-[#555] font-medium px-2 py-0.5 rounded transition-all duration-200 hover:text-[var(--color-blue)] hover:bg-[rgba(56,161,219,0.1)]" @click="handleNavigateToRoot">我的网盘</span>
          </div>
        </template>
      </div>

        <div class="flex-1 text-center max-md:w-full max-md:order-[-1]">
          <h2 class="text-lg font-semibold text-[#333] m-0 whitespace-nowrap overflow-hidden text-ellipsis max-w-[400px] max-md:max-w-full inline-block px-4 py-2 rounded-lg bg-white/80 backdrop-blur-[4px] shadow-[0_2px_8px_rgba(0,0,0,0.03)]">{{ currentFileName }}</h2>
        </div>

      <div class="flex-[2] flex justify-end gap-3 max-md:w-full max-md:justify-between">
        <!-- 返回按钮 -->
        <button class="flex items-center gap-2 rounded-lg cursor-pointer text-sm font-medium px-[18px] py-2.5 transition-all duration-300 bg-white/90 border border-[#eee] text-[#555] hover:-translate-y-0.5 hover:shadow-[0_4px_12px_rgba(0,0,0,0.08)] hover:bg-white hover:border-[#ddd]" @click="handleBack">
          <ArrowLeftIcon class="w-4 h-4" />
          {{ shareMode ? '返回首页' : '返回' }}
        </button>
        <!-- 下载按钮 -->
        <button v-if="!shareMode || canPreview" class="flex items-center gap-2 rounded-lg cursor-pointer text-sm font-medium px-[18px] py-2.5 transition-all duration-300 bg-gradient-to-br from-[#4facfe] to-[#00f2fe] text-white border-none shadow-[0_4px_10px_rgba(79,172,254,0.2)] hover:-translate-y-0.5 hover:shadow-[0_6px_15px_rgba(79,172,254,0.3)] hover:from-[#4facfe] hover:to-[#00c6fb]" @click="handleDownload">
          <DownloadIcon class="w-4 h-4" />
          下载
        </button>
      </div>
      </div>
    </div>

    <div class="flex-1 flex justify-center items-center overflow-auto p-8 max-md:p-4">
      <!-- ========== 分享模式特殊状态处理 ========== -->
      <template v-if="shareMode && !canPreview">
        <!-- 分享加载中 -->
        <div v-if="shareLoading" class="flex flex-col items-center justify-center h-full">
          <div class="w-12 h-12 border-3 border-[rgba(56,161,219,0.1)] border-t-[var(--color-blue)] rounded-full animate-spin mb-5"></div>
          <p class="text-[#666] text-base font-medium">加载中...</p>
        </div>

        <!-- 分享错误 -->
        <div v-else-if="shareError" class="flex flex-col items-center justify-center h-full max-w-[450px] bg-white p-10 rounded-2xl shadow-[0_10px_30px_rgba(0,0,0,0.08)]">
          <div class="w-[70px] h-[70px] rounded-full bg-linear-to-br from-[#ff4d4f] to-[#ff7875] text-white flex items-center justify-center text-[32px] font-bold mb-6 shadow-[0_8px_20px_rgba(255,77,79,0.2)]">!</div>
          <h3 class="text-[22px] text-[#333] mb-4 font-semibold">访问失败</h3>
          <p class="text-[#666] mb-7 text-center leading-relaxed text-[15px]">{{ shareError }}</p>
          <div class="flex gap-4">
            <a href="/" class="px-6 py-3 rounded-[10px] cursor-pointer text-[15px] font-medium transition-all duration-300 flex items-center justify-center gap-2 bg-linear-to-br from-[#4facfe] to-[#00f2fe] text-white border-none shadow-[0_4px_10px_rgba(79,172,254,0.3)] hover:-translate-y-0.5 hover:shadow-[0_8px_20px_rgba(79,172,254,0.4)] hover:from-[#4facfe] hover:to-[#00c6fb]">返回首页</a>
          </div>
        </div>

        <!-- 分享已过期 -->
        <div v-else-if="shareInfo?.is_expired" class="flex flex-col items-center justify-center h-full max-w-[450px] bg-white p-10 rounded-2xl shadow-[0_10px_30px_rgba(0,0,0,0.08)]">
          <div class="w-[70px] h-[70px] rounded-full bg-linear-to-br from-[#ff4d4f] to-[#ff7875] text-white flex items-center justify-center text-[32px] font-bold mb-6 shadow-[0_8px_20px_rgba(255,77,79,0.2)]">!</div>
          <h3 class="text-[22px] text-[#333] mb-4 font-semibold">分享已过期</h3>
          <p class="text-[#666] mb-7 text-center leading-relaxed text-[15px]">此分享链接已过期，无法访问</p>
          <div class="flex gap-4">
            <a href="/" class="px-6 py-3 rounded-[10px] cursor-pointer text-[15px] font-medium transition-all duration-300 flex items-center justify-center gap-2 bg-linear-to-br from-[#4facfe] to-[#00f2fe] text-white border-none shadow-[0_4px_10px_rgba(79,172,254,0.3)] hover:-translate-y-0.5 hover:shadow-[0_8px_20px_rgba(79,172,254,0.4)] hover:from-[#4facfe] hover:to-[#00c6fb]">返回首页</a>
          </div>
        </div>

        <!-- 需要密码验证 -->
        <div v-else-if="shareInfo?.has_password && !passwordVerified" class="flex justify-center items-center w-full p-5">
          <div class="bg-white rounded-[20px] px-10 py-12 shadow-[0_10px_40px_rgba(0,0,0,0.08)] max-w-[420px] w-full text-center max-md:px-6 max-md:mx-4">
            <div class="mb-6 flex justify-center items-center">
              <LockIcon class="w-12 h-12 p-4 bg-linear-to-br from-[rgba(79,172,254,0.1)] to-[rgba(0,242,254,0.1)] rounded-full text-[var(--color-blue)] box-content" />
            </div>
            <h3 class="text-[22px] font-semibold text-[#1a1a1a] mb-3">此分享需要密码访问</h3>
            <div class="flex flex-col gap-1.5 mb-8 p-4 bg-[#f8f9fa] rounded-xl">
              <span class="font-semibold text-[15px] text-[#333] break-all leading-relaxed">{{ shareInfo.file_name }}</span>
              <span class="text-[13px] text-[#888]">{{ formatFileSize(shareInfo.file_size) }}</span>
            </div>
            <div class="flex gap-2.5 mb-5">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                placeholder="请输入访问密码"
                class="flex-1 px-[18px] py-4 border-2 border-[#e8e8e8] rounded-xl text-[15px] outline-none transition-all duration-300 bg-[#fafafa] focus:border-[var(--color-blue)] focus:bg-white focus:shadow-[0_0_0_4px_rgba(79,172,254,0.1)] placeholder:text-[#aaa]"
                @keyup.enter="verifySharePasswordHandler"
              />
              <button class="p-3.5 bg-[#f0f0f0] border-2 border-transparent rounded-xl cursor-pointer transition-all duration-300 flex items-center justify-center hover:bg-[#e5e5e5]" @click="showPassword = !showPassword">
                <EyeIcon v-if="!showPassword" class="w-[22px] h-[22px] text-[#666]" />
                <EyeOffIcon v-else class="w-[22px] h-[22px] text-[#666]" />
              </button>
            </div>
            <button class="w-full py-4 bg-linear-to-br from-[#4facfe] to-[#00f2fe] text-white border-none rounded-xl text-base font-semibold cursor-pointer transition-all duration-300 shadow-[0_4px_15px_rgba(79,172,254,0.3)] hover:enabled:-translate-y-0.5 hover:enabled:shadow-[0_8px_25px_rgba(79,172,254,0.4)] active:enabled:translate-y-0 disabled:opacity-60 disabled:cursor-not-allowed" @click="verifySharePasswordHandler" :disabled="verifying">
              {{ verifying ? '验证中...' : '验证密码' }}
            </button>
          </div>
        </div>
      </template>

      <!-- ========== 统一预览区域（分享模式验证通过或普通模式） ========== -->
      <template v-else>
        <!-- 判断文件类型是否支持预览 -->
        <template v-if="currentSupportedPreviewType">
          <!-- 加载状态 -->
          <div v-if="isLoading" class="flex flex-col items-center justify-center h-full">
            <div class="w-12 h-12 border-3 border-[rgba(56,161,219,0.1)] border-t-[var(--color-blue)] rounded-full animate-spin mb-5"></div>
            <p class="text-[#666] text-base font-medium">正在加载预览...</p>
          </div>

          <!-- 错误状态 -->
          <div v-else-if="hasError" class="flex flex-col items-center justify-center h-full max-w-[450px] bg-white p-10 rounded-2xl shadow-[0_10px_30px_rgba(0,0,0,0.08)]">
            <div class="w-[70px] h-[70px] rounded-full bg-linear-to-br from-[#ff4d4f] to-[#ff7875] text-white flex items-center justify-center text-[32px] font-bold mb-6 shadow-[0_8px_20px_rgba(255,77,79,0.2)]">!</div>
            <h3 class="text-[22px] text-[#333] mb-4 font-semibold">预览失败</h3>
            <p class="text-[#666] mb-7 text-center leading-relaxed text-[15px]">{{ errorMessage }}</p>
            <div class="flex gap-4">
              <button class="px-6 py-3 rounded-[10px] cursor-pointer text-[15px] font-medium transition-all duration-300 flex items-center justify-center gap-2 bg-linear-to-br from-[#4facfe] to-[#00f2fe] text-white border-none shadow-[0_4px_10px_rgba(79,172,254,0.3)] hover:-translate-y-0.5 hover:shadow-[0_8px_20px_rgba(79,172,254,0.4)] hover:from-[#4facfe] hover:to-[#00c6fb]" @click="handleRetry">重试</button>
              <button class="px-6 py-3 rounded-[10px] cursor-pointer text-[15px] font-medium transition-all duration-300 flex items-center justify-center gap-2 bg-white text-[#555] border border-[#ddd] hover:bg-[#f9f9f9] hover:border-[#ccc] hover:-translate-y-0.5 hover:shadow-[0_4px_12px_rgba(0,0,0,0.05)]" @click="handleDownload">下载文件</button>
            </div>
          </div>

          <!-- 图片预览 -->
          <div v-else-if="currentFileType === 'image'" class="max-w-full max-h-full flex justify-center items-center bg-[#f5f5f5] rounded-xl overflow-hidden shadow-[0_10px_30px_rgba(0,0,0,0.08)]">
            <img :src="currentFileUrl" :alt="currentFileName" class="max-w-full max-h-[80vh] object-contain rounded-lg transition-transform duration-300 hover:scale-[1.02]" @load="onPreviewLoaded" @error="onPreviewError('图片加载失败')" />
          </div>

          <!-- 视频预览 -->
          <div v-else-if="currentFileType === 'video'" class="w-full h-full flex justify-center items-center bg-black rounded-xl overflow-hidden shadow-[0_10px_30px_rgba(0,0,0,0.1)]">
            <video controls :src="currentFileUrl" class="max-w-full max-h-[80vh] rounded-lg" @loadeddata="onPreviewLoaded" @error="onPreviewError('视频加载失败')">
              您的浏览器不支持视频播放
            </video>
          </div>

          <!-- 音频预览 -->
          <div v-else-if="currentFileType === 'audio'" class="w-full py-5 flex justify-center">
            <div class="flex items-center gap-[30px] bg-linear-to-br from-white to-[#f8f9fa] p-10 rounded-[20px] shadow-[0_10px_30px_rgba(0,0,0,0.08)] max-w-[650px] w-full max-md:flex-col max-md:p-[30px]">
              <div class="flex justify-center items-center w-[90px] h-[90px] bg-linear-to-br from-[#e3fdf5] to-[#ffe6fa] rounded-full shadow-[0_8px_20px_rgba(0,0,0,0.05)] max-md:mb-5">
                <MusicIcon class="w-[45px] h-[45px] text-[var(--color-blue)]" />
              </div>
              <div class="flex-1">
                <div class="text-lg font-semibold text-[#333] mb-5">{{ currentFileName }}</div>
                <audio controls :src="currentFileUrl" class="w-full h-10 outline-none" @loadeddata="onPreviewLoaded" @error="onPreviewError('音频加载失败')">
                  您的浏览器不支持音频播放
                </audio>
              </div>
            </div>
          </div>

          <!-- PDF预览 -->
          <div v-else-if="currentFileType === 'pdf'" class="w-full h-full bg-white rounded-xl overflow-hidden shadow-[0_10px_30px_rgba(0,0,0,0.08)]">
            <iframe :src="currentFileUrl" frameborder="0" class="w-full h-full border-none" @load="onPreviewLoaded" @error="onPreviewError('PDF加载失败')"></iframe>
          </div>

          <!-- 文本文件预览 -->
          <div v-else-if="currentFileType === 'text'" class="w-full h-full bg-white rounded-xl overflow-auto shadow-[0_10px_30px_rgba(0,0,0,0.08)] p-5">
            <div v-if="isMarkdown" class="markdown-content p-5 leading-relaxed text-[#333] font-sans">
              <div v-html="renderedMarkdown"></div>
            </div>
            <div v-else-if="isHtmlFile" class="w-full h-full flex flex-col">
              <div class="flex gap-3 px-5 py-2.5 bg-[#f9f9fa] border-b border-[#eee] max-md:flex-col max-md:gap-2 max-md:px-4 max-md:py-2">
                <button class="px-4 py-2 rounded-md text-sm font-medium cursor-pointer transition-all duration-200 flex items-center justify-center gap-2 border-none" :class="showHtmlPreview ? 'bg-[#3b82f6] text-white' : 'bg-[#e5e7eb] text-[#374151] hover:bg-[#d1d5db]'" @click="toggleHtmlPreview">
                  {{ showHtmlPreview ? '查看源码' : '预览HTML' }}
                </button>
                <button class="px-4 py-2 rounded-md text-sm font-medium cursor-pointer transition-all duration-200 flex items-center justify-center gap-2 bg-white border border-[#d1d5db] text-[#4b5563] hover:bg-[#f9fafb] hover:border-[#9ca3af]" @click="openHtmlInNewWindow" title="在新窗口中打开">
                  <span>打开新窗口</span>
                </button>
              </div>
              <div v-if="showHtmlPreview" class="flex-1 p-0">
                <iframe :src="htmlPreviewUrl" sandbox="allow-scripts allow-same-origin" class="w-full h-full min-h-[70vh] border-none bg-white"></iframe>
              </div>
              <div v-else class="code-content w-full h-full p-5 overflow-auto">
                <div v-html="highlightedCode"></div>
              </div>
            </div>
            <div v-else-if="isVueFile || isCodeFile" class="code-content w-full h-full p-5 overflow-auto">
              <div v-html="highlightedCode"></div>
            </div>
            <div v-else class="w-full h-full">
              <pre class="font-mono text-sm leading-relaxed whitespace-pre-wrap break-all text-[#333] p-5 m-0 overflow-auto max-h-[70vh]">{{ textContent }}</pre>
            </div>
          </div>
        </template>

        <!-- 不支持预览的文件类型 -->
        <div v-else class="flex justify-center items-center h-full">
          <div class="flex flex-col items-center justify-center bg-linear-to-br from-white to-[#f8f9fa] p-[50px] rounded-[20px] shadow-[0_10px_30px_rgba(0,0,0,0.08)] max-w-[440px] text-center max-md:p-[30px]">
            <div class="mb-[30px] bg-linear-to-br from-[#e3fdf5] to-[#ffe6fa] w-[110px] h-[110px] rounded-full flex justify-center items-center shadow-[0_8px_20px_rgba(0,0,0,0.05)]">
              <component :is="file.icon || FileIcon" class="w-[55px] h-[55px] text-[#999]" :class="getIconClass(currentFileType)" />
            </div>
            <div class="text-center">
              <h3 class="text-[22px] text-[#333] mb-4 font-semibold">无法预览此文件</h3>
              <p class="text-[#666] mb-7 leading-relaxed text-[15px]">该文件类型暂不支持在线预览，您可以下载后在本地查看。</p>
              <div class="flex flex-col gap-2.5 mb-7 bg-[rgba(245,247,250,0.5)] rounded-[10px] px-5 py-4 text-sm text-[#555] shadow-[0_2px_8px_rgba(0,0,0,0.03)]">
                <div class="flex justify-between items-center border-b border-dashed border-black/5 pb-2">
                  <span class="font-medium text-[#333]">文件名</span>
                  <span class="font-normal text-[#666] max-w-[250px] whitespace-nowrap overflow-hidden text-ellipsis">{{ currentFileName }}</span>
                </div>
                <div v-if="currentFileSize" class="flex justify-between items-center border-b border-dashed border-black/5 pb-2">
                  <span class="font-medium text-[#333]">大小</span>
                  <span class="font-normal text-[#666] max-w-[250px] whitespace-nowrap overflow-hidden text-ellipsis">{{ formatFileSize(currentFileSize) }}</span>
                </div>
                <div v-if="currentFileCreateTime" class="flex justify-between items-center">
                  <span class="font-medium text-[#333]">创建时间</span>
                  <span class="font-normal text-[#666] max-w-[250px] whitespace-nowrap overflow-hidden text-ellipsis">{{ formatDate(currentFileCreateTime) }}</span>
                </div>
              </div>
              <div class="flex justify-center mt-5">
                <button class="flex flex-col items-center no-underline transition-all duration-300 hover:-translate-y-[3px] group" @click="handleDownload">
                  <div class="w-[60px] h-[60px] bg-linear-to-br from-[#4facfe] to-[#00f2fe] rounded-full flex justify-center items-center mb-3 shadow-[0_5px_15px_rgba(79,172,254,0.3)] transition-all duration-300 group-hover:shadow-[0_8px_20px_rgba(79,172,254,0.4)] group-hover:from-[#4facfe] group-hover:to-[#00c6fb]">
                    <DownloadIcon class="w-7 h-7 text-white" />
                  </div>
                  <span class="text-[15px] font-medium text-[#666] transition-colors duration-300 group-hover:text-[#4facfe]">下载文件</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, inject, nextTick, onMounted, watch } from 'vue'
import type { FileItem } from '../utils/types/file'
import { getDownloadUrl } from '@/api/file'
import { SERVER_URL } from '@/types/Constant'
import {
  getShareInfo,
  verifySharePassword,
  getShareDownloadUrl,
  type PublicShareInfo
} from '@/api/share'
import {
  ArrowLeftIcon,
  DownloadIcon,
  MusicIcon,
  HomeIcon,
  ChevronRightIcon,
  FileIcon,
  LockIcon,
  EyeIcon,
  EyeOffIcon
} from '../utils/icons'
import axios from 'axios'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.css' // 引入GitHub样式的高亮CSS

// 定义路径段接口
interface PathSegment {
  id: string;
  name: string;
}

const props = defineProps<{
  file: FileItem
  // 分享模式相关
  shareMode?: boolean
  shareId?: string
}>()

const emits = defineEmits<{
  (e: 'close'): void
}>()

// 从父组件注入路径导航信息
const pathSegments = inject<PathSegment[]>('pathSegments', [])
const navigateToRoot = inject('navigateToRoot', () => {})
const navigateToPathSegment = inject('navigateToPathSegment', (_index: number) => {})

// 分享模式相关状态
const shareInfo = ref<PublicShareInfo | null>(null)
const shareLoading = ref(false)
const shareError = ref('')
const password = ref('')
const showPassword = ref(false)
const passwordVerified = ref(false)
const verifying = ref(false)
const downloadToken = ref('')
const shareBlobUrl = ref('') // 分享模式的 Blob URL
const shareStreamUrl = ref('') // 分享模式的流式传输 URL（音视频）

// 加载和错误状态
const isLoading = ref(true)
const hasError = ref(false)
const errorMessage = ref('')
const textContent = ref('') // 添加textContent变量

// 使用ref存储渲染后的Markdown内容
const renderedMarkdown = ref('')

// 高亮代码
const highlightedCode = ref('')

// 根据文件扩展名获取语言
function getLanguage(fileName: string): string {
  const extension = fileName.split('.').pop()?.toLowerCase();
  if (!extension) return 'plaintext';
  
  // 映射常见的文件扩展名到highlight.js支持的语言
  const langMap: Record<string, string> = {
    js: 'javascript',
    ts: 'typescript',
    html: 'html',
    htm: 'html',
    vue: 'vue',
    css: 'css',
    xml: 'xml',
    json: 'json',
    py: 'python',
    java: 'java',
    c: 'c',
    cpp: 'cpp',
    go: 'go',
    php: 'php',
    rb: 'ruby',
    sh: 'bash',
    sql: 'sql',
    yaml: 'yaml',
    yml: 'yaml',
    md: 'markdown',
    markdown: 'markdown',
    jsx: 'javascript',
    tsx: 'typescript'
  };
  
  return langMap[extension] || 'plaintext';
}

// 计算属性：是否支持预览的文件类型
const isSupportedPreviewType = computed(() => {
  const supportedTypes = ['image', 'video', 'audio', 'pdf', 'text'];
  return supportedTypes.includes(props.file.type);
});

// ========== 统一计算属性（分享模式和普通模式共用） ==========

// 分享文件类型（根据文件名扩展名判断）
const shareFileType = computed(() => {
  if (!shareInfo.value) return 'file'
  const name = shareInfo.value.file_name.toLowerCase()
  const ext = name.split('.').pop() || ''

  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico']
  const videoExts = ['mp4', 'webm', 'ogg', 'mov', 'avi', 'mkv']
  const audioExts = ['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a']
  const pdfExts = ['pdf']
  const textExts = ['txt', 'log', 'md', 'markdown', 'json', 'xml', 'html', 'htm', 'css', 'js', 'ts', 'vue', 'py', 'java', 'c', 'cpp', 'go', 'php', 'rb', 'sh', 'sql', 'yaml', 'yml', 'ini', 'conf', 'cfg', 'env']

  if (imageExts.includes(ext)) return 'image'
  if (videoExts.includes(ext)) return 'video'
  if (audioExts.includes(ext)) return 'audio'
  if (pdfExts.includes(ext)) return 'pdf'
  if (textExts.includes(ext)) return 'text'
  return 'file'
})

// 是否可以进入预览模式（分享模式需要验证通过）
const canPreview = computed(() => {
  if (props.shareMode) {
    return passwordVerified.value && !shareError.value && !shareInfo.value?.is_expired
  }
  return true
})

// 统一的文件名
const currentFileName = computed(() => {
  if (props.shareMode) {
    return shareInfo.value?.file_name || '文件分享'
  }
  return props.file.name
})

// 统一的文件类型
const currentFileType = computed(() => {
  if (props.shareMode) {
    return shareFileType.value
  }
  return props.file.type
})

// 统一的文件 URL
const currentFileUrl = computed(() => {
  if (props.shareMode) {
    // 分享模式：音视频使用流式 URL，其他使用 Blob URL
    if (isStreamableMedia.value && shareStreamUrl.value) {
      return shareStreamUrl.value
    }
    return shareBlobUrl.value
  }
  // 普通模式：音视频使用 preview URL 支持流式传输
  if (isStreamableMedia.value && props.file.id) {
    return getDownloadUrl(props.file.id, true) // preview=true
  }
  return fileUrl.value
})

// 统一的文件大小
const currentFileSize = computed(() => {
  if (props.shareMode) {
    return shareInfo.value?.file_size || 0
  }
  return props.file.originalFile?.size || props.file.size || 0
})

// 统一的文件创建时间
const currentFileCreateTime = computed(() => {
  if (props.shareMode) return ''
  return props.file.originalFile?.createTime || ''
})

// ========== 统一的操作方法 ==========

// 统一的返回操作
const handleBack = () => {
  if (props.shareMode) {
    window.location.href = '/'
  } else {
    emits('close')
  }
}

// 统一的下载操作
const handleDownload = () => {
  if (props.shareMode) {
    downloadShareFile()
  } else {
    downloadFile()
  }
}

// 统一的重试操作
const handleRetry = () => {
  if (props.shareMode) {
    fetchShareFileContent()
  } else {
    retryPreview()
  }
}

// 统一的是否支持预览
const currentSupportedPreviewType = computed(() => {
  const supportedTypes = ['image', 'video', 'audio', 'pdf', 'text']
  return supportedTypes.includes(currentFileType.value)
})

// 是否为流媒体类型（音视频）- 使用直接 URL 流式传输
const isStreamableMedia = computed(() => {
  return ['video', 'audio'].includes(currentFileType.value)
})

// 添加isMarkdown和isCodeFile计算属性
const isMarkdown = computed(() => {
  const name = currentFileName.value;
  if (!name) return false;
  const extension = name.split('.').pop()?.toLowerCase();
  return extension === 'md' || extension === 'markdown';
});

const isCodeFile = computed(() => {
  const name = currentFileName.value;
  if (!name) return false;
  const extension = name.split('.').pop()?.toLowerCase();
  return ['js', 'ts', 'html', 'css', 'xml', 'json', 'py', 'java', 'c', 'cpp', 'go', 'php', 'rb', 'sh', 'sql', 'yaml', 'yml'].includes(extension || '');
});

// 添加isHtmlFile计算属性
const isHtmlFile = computed(() => {
  const name = currentFileName.value;
  if (!name) return false;
  const extension = name.split('.').pop()?.toLowerCase();
  return extension === 'html' || extension === 'htm';
});

// 添加isVueFile计算属性
const isVueFile = computed(() => {
  const name = currentFileName.value;
  if (!name) return false;
  const extension = name.split('.').pop()?.toLowerCase();
  return extension === 'vue';
});

// 监听textContent变化，处理代码高亮
watch([textContent, currentFileName], async ([newContent, fileName]) => {
  if (!newContent || !fileName) return;

  // 处理Markdown
  if (isMarkdown.value) {
    try {
      const parsed = await marked.parse(newContent);
      renderedMarkdown.value = DOMPurify.sanitize(parsed);
    } catch (error) {
      console.error('Markdown渲染失败:', error);
      renderedMarkdown.value = `<p class="error">Markdown渲染失败: ${error}</p>`;
    }
    return;
  }

  // 处理代码文件，包括HTML和Vue
  if (isHtmlFile.value || isVueFile.value || isCodeFile.value) {
    try {
      const language = getLanguage(fileName);
      const highlighted = hljs.highlight(newContent, { language }).value;
      highlightedCode.value = `<pre class="hljs"><code>${highlighted}</code></pre>`;
    } catch (error) {
      console.error('代码高亮失败:', error);
      highlightedCode.value = `<pre class="error">${newContent}</pre>`;
    }
  }
}, { immediate: true });

// 控制HTML预览的状态
const showHtmlPreview = ref(false);
const htmlPreviewUrl = ref('');

// 处理根目录导航
const handleNavigateToRoot = () => {
  // 先关闭预览
  emits('close')
  // 然后导航到根目录
  nextTick(() => {
    navigateToRoot()
  })
}

// 处理路径段导航
const handleNavigateToPathSegment = (index: number) => {
  // 先关闭预览
  emits('close')
  // 然后导航到指定路径段
  nextTick(() => {
    navigateToPathSegment(index)
  })
}

// 获取文件URL
const fileUrl = computed(() => {
  if (props.file.id) {
    const url = getDownloadUrl(props.file.id);
    return url;
  }
  return '';
})

// 下载文件
const downloadFile = () => {
  if (props.file.id) {
    const downloadUrl = getDownloadUrl(props.file.id);
    window.open(downloadUrl, '_blank');
  }
}

// 直接请求文件内容
const fetchFileContent = async () => {
  if (!props.file.id) {
    onPreviewError('文件ID不存在，无法获取文件内容');
    return;
  }

  // 检查文件类型是否支持预览
  if (!isSupportedPreviewType.value) {
    // 不支持预览的文件类型，直接显示不支持预览界面
    isLoading.value = false;
    return;
  }

  // 音视频使用直接 URL 流式传输，不需要 axios 请求
  // 直接显示播放器，让浏览器处理流式加载
  if (isStreamableMedia.value) {
    isLoading.value = false;
    return;
  }

  // 图片/PDF 使用直接 URL，直接显示元素让浏览器加载
  if (props.file.type === 'image' || props.file.type === 'pdf') {
    isLoading.value = false;
    return;
  }

  // 文本文件需要获取内容
  if (props.file.type === 'text') {
    try {
      isLoading.value = true;
      hasError.value = false;

      const url = `/files/download/${props.file.id}`;

      const response = await axios.create({
        baseURL: SERVER_URL,
        headers: {
          Authorization: localStorage.getItem("token") || ""
        }
      }).get(url, {
        responseType: 'text'
      });

      textContent.value = response.data;
      onPreviewLoaded();
    } catch (error) {
      onPreviewError('请求文件内容失败');
    }
    return;
  }

  // 其他类型，结束加载状态
  isLoading.value = false;
}

// 预览加载成功
const onPreviewLoaded = () => {
  isLoading.value = false;
  hasError.value = false;
}

// 预览加载失败
const onPreviewError = (message: string) => {
  isLoading.value = false;
  hasError.value = true;
  errorMessage.value = message || '文件预览失败，请尝试下载后查看';
}

// 重试预览
const retryPreview = () => {
  isLoading.value = true;
  hasError.value = false;
  errorMessage.value = '';
  
  // 尝试直接请求文件内容
  fetchFileContent();
}

// 获取图标类名
const getIconClass = (fileType: string) => {
  const typeClassMap = {
    'image': 'image-icon',
    'video': 'video-icon',
    'audio': 'audio-icon',
    'code': 'code-icon',
    'pdf': 'pdf-icon',
    'archive': 'archive-icon',
    'spreadsheet': 'spreadsheet-icon',
    'presentation': 'presentation-icon'
  }
  
  return typeClassMap[fileType as keyof typeof typeClassMap] || ''
}

// 格式化文件大小
const formatFileSize = (bytes: number | string) => {
  if (typeof bytes === 'string') {
    // 尝试将字符串转换为数字
    bytes = parseInt(bytes, 10);
    if (isNaN(bytes)) return 'Unknown';
  }
  
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 格式化日期
const formatDate = (timestamp: string) => {
  if (!timestamp) return '未知';
  
  try {
    // 尝试解析日期，支持多种格式
    const date = new Date(timestamp);
    
    // 检查日期是否有效
    if (isNaN(date.getTime())) {
      return '未知';
    }
    
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    
    return `${year}-${month}-${day} ${hours}:${minutes}`;
  } catch (error) {
    return '未知';
  }
}

// 添加切换HTML预览的函数
function toggleHtmlPreview() {
  if (!showHtmlPreview.value) {
    // 创建HTML预览URL
    const htmlBlob = new Blob([textContent.value], { type: 'text/html' });
    htmlPreviewUrl.value = URL.createObjectURL(htmlBlob);
  }
  showHtmlPreview.value = !showHtmlPreview.value;
}

// 在新窗口中打开HTML
function openHtmlInNewWindow() {
  const htmlBlob = new Blob([textContent.value], { type: 'text/html' });
  const url = URL.createObjectURL(htmlBlob);
  window.open(url, '_blank');
}

// ========== 分享模式相关方法 ==========

// 获取分享信息
const fetchShareInfo = async () => {
  if (!props.shareId) {
    shareError.value = '无效的分享链接'
    return
  }

  try {
    shareLoading.value = true
    shareError.value = ''
    shareInfo.value = await getShareInfo(props.shareId)

    // 如果没有密码且未过期，自动获取下载令牌
    if (!shareInfo.value.has_password && !shareInfo.value.is_expired) {
      const result = await verifySharePassword(props.shareId, '')
      if (result.valid && result.download_token) {
        downloadToken.value = result.download_token
        passwordVerified.value = true
        // 开始加载预览
        fetchShareFileContent()
      }
    }
  } catch (e: any) {
    shareError.value = e.message || '分享不存在或已失效'
  } finally {
    shareLoading.value = false
  }
}

// 验证分享密码
const verifySharePasswordHandler = async () => {
  if (!password.value) {
    ElMessage.warning('请输入密码')
    return
  }

  try {
    verifying.value = true
    const result = await verifySharePassword(props.shareId!, password.value)
    if (result.valid && result.download_token) {
      downloadToken.value = result.download_token
      passwordVerified.value = true
      ElMessage.success('密码验证成功')
      // 开始加载预览
      fetchShareFileContent()
    } else {
      ElMessage.error('密码错误')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '密码错误')
  } finally {
    verifying.value = false
  }
}

// 获取分享文件内容进行预览
const fetchShareFileContent = async () => {
  if (!downloadToken.value || !props.shareId) return

  // 检查是否支持预览
  if (!currentSupportedPreviewType.value) {
    isLoading.value = false
    return
  }

  const fileType = currentFileType.value

  // 音视频使用直接 URL 流式传输
  if (['video', 'audio'].includes(fileType)) {
    // 设置流式传输 URL，带 preview=true 参数
    shareStreamUrl.value = getShareDownloadUrl(props.shareId, downloadToken.value, true)
    // 重新获取令牌用于下载（因为令牌是一次性的）
    try {
      const result = await verifySharePassword(props.shareId, password.value || '')
      if (result.valid && result.download_token) {
        downloadToken.value = result.download_token
      }
    } catch (e) {
      // 忽略错误，下载时会重新获取
    }
    // 直接显示播放器，让浏览器处理流式加载
    isLoading.value = false
    return
  }

  try {
    isLoading.value = true
    hasError.value = false

    const url = getShareDownloadUrl(props.shareId, downloadToken.value)

    const response = await axios.get(url, {
      responseType: fileType === 'text' ? 'text' : 'blob'
    })

    if (fileType === 'text') {
      textContent.value = response.data
    } else {
      // 对于图片/PDF，创建 Blob URL
      shareBlobUrl.value = URL.createObjectURL(response.data)
    }

    // 重新获取令牌用于下载（因为令牌是一次性的）
    try {
      const result = await verifySharePassword(props.shareId, password.value || '')
      if (result.valid && result.download_token) {
        downloadToken.value = result.download_token
      }
    } catch (e) {
      // 忽略错误，下载时会重新获取
    }

    onPreviewLoaded()
  } catch (error) {
    onPreviewError('请求文件内容失败')
  }
}

// 下载分享文件
const downloadShareFile = async () => {
  if (!downloadToken.value) {
    // 需要重新获取令牌
    try {
      const result = await verifySharePassword(props.shareId!, password.value || '')
      if (result.valid && result.download_token) {
        downloadToken.value = result.download_token
      } else {
        ElMessage.error('获取下载令牌失败，请刷新页面重试')
        return
      }
    } catch (e: any) {
      ElMessage.error(e.message || '获取下载令牌失败')
      return
    }
  }

  const url = getShareDownloadUrl(props.shareId!, downloadToken.value)
  const link = document.createElement('a')
  link.href = url
  link.download = shareInfo.value?.file_name || 'download'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)

  // 令牌是一次性的，下载后清空
  downloadToken.value = ''
}

// 组件挂载时尝试加载预览
onMounted(() => {
  if (props.shareMode) {
    // 分享模式：先获取分享信息
    fetchShareInfo()
  } else if (isSupportedPreviewType.value) {
    // 普通模式：直接加载文件内容
    fetchFileContent();
  } else {
    // 对于不支持预览的文件类型，直接结束加载状态
    isLoading.value = false;
  }
})
</script>

<style lang="less" scoped>
/* 动画 */
@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* Markdown 内容样式 - 因为是动态生成的 HTML 需要保留 */
.markdown-content {
  h1, h2, h3, h4, h5, h6 {
    margin-top: 24px;
    margin-bottom: 16px;
    font-weight: 600;
    line-height: 1.25;
  }

  h1 {
    font-size: 2em;
    border-bottom: 1px solid #eaecef;
    padding-bottom: 0.3em;
  }

  h2 {
    font-size: 1.5em;
    border-bottom: 1px solid #eaecef;
    padding-bottom: 0.3em;
  }

  h3 {
    font-size: 1.25em;
  }

  p, blockquote, ul, ol, dl, table, pre {
    margin-top: 0;
    margin-bottom: 16px;
  }

  blockquote {
    padding: 0 1em;
    color: #6a737d;
    border-left: 0.25em solid #dfe2e5;
  }

  code {
    padding: 0.2em 0.4em;
    margin: 0;
    font-size: 85%;
    background-color: rgba(27, 31, 35, 0.05);
    border-radius: 3px;
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  }

  pre {
    padding: 16px;
    overflow: auto;
    font-size: 85%;
    line-height: 1.45;
    background-color: #f6f8fa;
    border-radius: 3px;

    code {
      padding: 0;
      margin: 0;
      font-size: 100%;
      background-color: transparent;
      border-radius: 0;
    }
  }

  table {
    display: block;
    width: 100%;
    overflow: auto;
    border-spacing: 0;
    border-collapse: collapse;

    th, td {
      padding: 6px 13px;
      border: 1px solid #dfe2e5;
    }

    tr {
      background-color: #fff;
      border-top: 1px solid #c6cbd1;

      &:nth-child(2n) {
        background-color: #f6f8fa;
      }
    }
  }

  img {
    max-width: 100%;
    box-sizing: content-box;
    background-color: #fff;
  }

  .error {
    color: #e53e3e;
    padding: 10px;
    background-color: #fff5f5;
    border-left: 4px solid #e53e3e;
    margin-bottom: 16px;
  }
}

/* 代码高亮样式 */
.code-content {
  pre.hljs {
    margin: 0;
    padding: 16px;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
    max-height: 70vh;
    overflow: auto;
  }

  pre.error {
    background-color: #fff5f5;
    color: #e53e3e;
    padding: 16px;
    border-radius: 8px;
    border-left: 4px solid #e53e3e;
  }
}

/* 文件图标颜色类 */
.image-icon { color: var(--color-blue); }
.video-icon { color: #f50; }
.audio-icon { color: #73d13d; }
.code-icon { color: #722ed1; }
.pdf-icon { color: #f5222d; }
.archive-icon { color: #fa8c16; }
.spreadsheet-icon { color: #52c41a; }
.presentation-icon { color: #eb2f96; }
</style> 