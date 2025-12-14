<template>
  <div class="file-preview-container" :class="{ 'share-mode': shareMode }">
    <!-- 顶部导航栏 -->
    <div class="preview-header">
      <div class="header-content">
      <div class="header-left">
        <!-- 分享模式：显示Logo -->
        <template v-if="shareMode">
          <a href="/" class="logo-link">
            <span class="logo-text">DH-Blog</span>
          </a>
        </template>
        <!-- 普通模式：显示面包屑 -->
        <template v-else>
          <div class="breadcrumb">
            <HomeIcon class="icon-sm" @click="handleNavigateToRoot" />
            <template v-if="pathSegments.length > 0">
              <ChevronRightIcon class="icon-xs" />
              <template v-for="(segment, index) in pathSegments" :key="index">
                <span
                  class="path-segment"
                  @click="handleNavigateToPathSegment(index)"
                >{{ segment.name }}</span>
                <ChevronRightIcon v-if="index < pathSegments.length - 1" class="icon-xs" />
              </template>
            </template>
            <span v-else class="path-segment" @click="handleNavigateToRoot">我的网盘</span>
          </div>
        </template>
      </div>

        <div class="header-center">
          <h2 class="file-title">{{ currentFileName }}</h2>
        </div>

      <div class="header-right">
        <!-- 分享模式：显示返回首页和下载按钮 -->
        <template v-if="shareMode">
          <a href="/" class="action-btn back-btn">
            <ArrowLeftIcon class="icon-sm" />
            返回首页
          </a>
          <button v-if="canPreview" class="action-btn download-btn" @click="downloadShareFile">
            <DownloadIcon class="icon-sm" />
            下载
          </button>
        </template>
        <!-- 普通模式：显示返回和下载按钮 -->
        <template v-else>
          <button class="action-btn back-btn" @click="$emit('close')">
            <ArrowLeftIcon class="icon-sm" />
            返回
          </button>
          <a class="action-btn download-btn" :href="fileUrl" download :title="'下载' + file.name">
            <DownloadIcon class="icon-sm" />
            下载
          </a>
        </template>
      </div>
      </div>
    </div>

    <div class="preview-content">
      <!-- ========== 分享模式特殊状态处理 ========== -->
      <template v-if="shareMode && !canPreview">
        <!-- 分享加载中 -->
        <div v-if="shareLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p>加载中...</p>
        </div>

        <!-- 分享错误 -->
        <div v-else-if="shareError" class="error-container">
          <div class="error-icon">!</div>
          <h3>访问失败</h3>
          <p>{{ shareError }}</p>
          <div class="error-actions">
            <a href="/" class="btn-primary">返回首页</a>
          </div>
        </div>

        <!-- 分享已过期 -->
        <div v-else-if="shareInfo?.is_expired" class="error-container">
          <div class="error-icon">!</div>
          <h3>分享已过期</h3>
          <p>此分享链接已过期，无法访问</p>
          <div class="error-actions">
            <a href="/" class="btn-primary">返回首页</a>
          </div>
        </div>

        <!-- 需要密码验证 -->
        <div v-else-if="shareInfo?.has_password && !passwordVerified" class="password-container">
          <div class="password-card">
            <div class="lock-icon">
              <LockIcon class="icon-lg" />
            </div>
            <h3>此分享需要密码访问</h3>
            <div class="file-info-brief">
              <span class="file-name">{{ shareInfo.file_name }}</span>
              <span class="file-size">{{ formatFileSize(shareInfo.file_size) }}</span>
            </div>
            <div class="password-input-group">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                placeholder="请输入访问密码"
                class="password-input"
                @keyup.enter="verifySharePasswordHandler"
              />
              <button class="toggle-password" @click="showPassword = !showPassword">
                <EyeIcon v-if="!showPassword" class="icon-sm" />
                <EyeOffIcon v-else class="icon-sm" />
              </button>
            </div>
            <button class="verify-btn" @click="verifySharePasswordHandler" :disabled="verifying">
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
          <div v-if="isLoading" class="loading-container">
            <div class="loading-spinner"></div>
            <p>正在加载预览...</p>
          </div>

          <!-- 错误状态 -->
          <div v-else-if="hasError" class="error-container">
            <div class="error-icon">!</div>
            <h3>预览失败</h3>
            <p>{{ errorMessage }}</p>
            <div class="error-actions" v-if="!shareMode">
              <button class="btn-primary" @click="retryPreview">重试</button>
              <button class="btn-outline" @click="downloadFile">下载文件</button>
            </div>
          </div>

          <!-- 图片预览 -->
          <div v-else-if="currentFileType === 'image'" class="image-preview">
            <img :src="currentFileUrl" :alt="currentFileName" @load="onPreviewLoaded" @error="onPreviewError('图片加载失败')" />
          </div>

          <!-- 视频预览 -->
          <div v-else-if="currentFileType === 'video'" class="video-preview">
            <video controls :src="currentFileUrl" @loadeddata="onPreviewLoaded" @error="onPreviewError('视频加载失败')">
              您的浏览器不支持视频播放
            </video>
          </div>

          <!-- 音频预览 -->
          <div v-else-if="currentFileType === 'audio'" class="audio-preview">
            <div class="audio-card">
              <div class="audio-icon-container">
                <MusicIcon class="audio-icon" />
              </div>
              <div class="audio-info">
                <div class="audio-name">{{ currentFileName }}</div>
                <audio controls :src="currentFileUrl" @loadeddata="onPreviewLoaded" @error="onPreviewError('音频加载失败')">
                  您的浏览器不支持音频播放
                </audio>
              </div>
            </div>
          </div>

          <!-- PDF预览 -->
          <div v-else-if="currentFileType === 'pdf'" class="pdf-preview">
            <iframe :src="currentFileUrl" frameborder="0" @load="onPreviewLoaded" @error="onPreviewError('PDF加载失败')"></iframe>
          </div>

          <!-- 文本文件预览 -->
          <div v-else-if="currentFileType === 'text'" class="text-preview">
            <div v-if="isMarkdown" class="markdown-content">
              <div v-html="renderedMarkdown"></div>
            </div>
            <div v-else-if="isHtmlFile" class="html-preview-container">
              <div class="preview-controls">
                <button class="preview-toggle-btn" @click="toggleHtmlPreview" :class="{ active: showHtmlPreview }">
                  {{ showHtmlPreview ? '查看源码' : '预览HTML' }}
                </button>
                <button class="open-new-window-btn" @click="openHtmlInNewWindow" title="在新窗口中打开">
                  <span>打开新窗口</span>
                </button>
              </div>
              <div v-if="showHtmlPreview" class="html-render">
                <iframe :src="htmlPreviewUrl" sandbox="allow-scripts allow-same-origin" class="html-iframe"></iframe>
              </div>
              <div v-else class="code-content">
                <div v-html="highlightedCode"></div>
              </div>
            </div>
            <div v-else-if="isVueFile || isCodeFile" class="code-content">
              <div v-html="highlightedCode"></div>
            </div>
            <div v-else class="text-content">
              <pre>{{ textContent }}</pre>
            </div>
          </div>
        </template>

        <!-- 不支持预览的文件类型 -->
        <div v-else class="unsupported-preview">
          <div class="unsupported-card">
            <div class="unsupported-icon">
              <component :is="file.icon || FileIcon" class="file-icon" :class="getIconClass(currentFileType)" />
            </div>
            <div class="unsupported-message">
              <h3>无法预览此文件</h3>
              <p>该文件类型暂不支持在线预览，您可以下载后在本地查看。</p>
              <div class="file-info">
                <div class="file-info-item">
                  <span class="label">文件名</span>
                  <span class="value">{{ currentFileName }}</span>
                </div>
                <div class="file-info-item" v-if="currentFileSize">
                  <span class="label">大小</span>
                  <span class="value">{{ formatFileSize(currentFileSize) }}</span>
                </div>
                <div class="file-info-item" v-if="!shareMode && file.originalFile && file.originalFile.createTime">
                  <span class="label">创建时间</span>
                  <span class="value">{{ formatDate(file.originalFile.createTime) }}</span>
                </div>
              </div>
              <div class="button-container">
                <template v-if="shareMode">
                  <button class="download-button" @click="downloadShareFile">
                    <div class="download-button-icon">
                      <DownloadIcon class="icon" />
                    </div>
                    <span class="download-button-text">下载文件</span>
                  </button>
                </template>
                <template v-else>
                  <a class="download-button" :href="fileUrl" download :title="'下载' + file.name">
                    <div class="download-button-icon">
                      <DownloadIcon class="icon" />
                    </div>
                    <span class="download-button-text">下载文件</span>
                  </a>
                </template>
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

// 加载和错误状态
const isLoading = ref(true)
const hasError = ref(false)
const errorMessage = ref('')
const fileContent = ref<Blob | null>(null)
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
    return shareBlobUrl.value // 使用 Blob URL
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

// 统一的是否支持预览
const currentSupportedPreviewType = computed(() => {
  const supportedTypes = ['image', 'video', 'audio', 'pdf', 'text']
  return supportedTypes.includes(currentFileType.value)
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

  try {
    isLoading.value = true;
    hasError.value = false;
    
    const url = `/files/download/${props.file.id}`;
    
    // 直接使用axios而不是request实例，因为我们需要原始响应
    const response = await axios.create({
      baseURL: SERVER_URL,
      headers: {
        Authorization: localStorage.getItem("token") || ""
      }
    }).get(url, {
      responseType: props.file.type === 'text' ? 'text' : 'blob'
    });
    
    // 根据文件类型处理预览
    if (props.file.type === 'text') {
      // 文本文件直接显示内容
      textContent.value = response.data;
    } else {
      // 创建Blob URL
      fileContent.value = response.data;
      const blobUrl = URL.createObjectURL(response.data);
      
      if (props.file.type === 'image') {
        const imgElement = document.querySelector('.image-preview img') as HTMLImageElement;
        if (imgElement) {
          imgElement.src = blobUrl;
        }
      } else if (props.file.type === 'video') {
        const videoElement = document.querySelector('.video-preview video') as HTMLVideoElement;
        if (videoElement) {
          videoElement.src = blobUrl;
        }
      } else if (props.file.type === 'audio') {
        const audioElement = document.querySelector('.audio-preview audio') as HTMLAudioElement;
        if (audioElement) {
          audioElement.src = blobUrl;
        }
      } else if (props.file.type === 'pdf') {
        const iframeElement = document.querySelector('.pdf-preview iframe') as HTMLIFrameElement;
        if (iframeElement) {
          iframeElement.src = blobUrl;
        }
      }
    }
    
    onPreviewLoaded();
  } catch (error) {
    onPreviewError('请求文件内容失败');
  }
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

  try {
    isLoading.value = true
    hasError.value = false

    const url = getShareDownloadUrl(props.shareId, downloadToken.value)
    const fileType = currentFileType.value

    const response = await axios.get(url, {
      responseType: fileType === 'text' ? 'text' : 'blob'
    })

    if (fileType === 'text') {
      textContent.value = response.data
    } else {
      // 对于图片/视频/音频/PDF，创建 Blob URL
      fileContent.value = response.data
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
.file-preview-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #f8f9fa;
  animation: fade-in 0.3s ease;
  position: relative;
  
  .preview-header {
    position: sticky;
    top: 0;
    z-index: 10;
    backdrop-filter: blur(20px);
    background-color: rgba(255, 255, 255, 0.9);
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.05);
    padding: 10px 0;
    
    .header-content {
      max-width: 1400px;
      margin: 0 auto;
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 24px;
    }
    
    .header-left {
      flex: 2;
      
      .breadcrumb {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        background-color: rgba(248, 249, 250, 0.9);
        padding: 10px 16px;
        border-radius: 50px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.03);
        backdrop-filter: blur(4px);
        
        .icon-sm {
          cursor: pointer;
          color: #555;
          width: 16px;
          height: 16px;
          transition: all 0.2s ease;
          
          &:hover {
            color: var(--color-blue);
            transform: scale(1.1);
          }
        }
        
        .icon-xs {
          color: #aaa;
          width: 12px;
          height: 12px;
        }
        
        .path-segment {
          cursor: pointer;
          color: #555;
          font-weight: 500;
          padding: 2px 8px;
          border-radius: 4px;
          transition: all 0.2s ease;
          
          &:hover {
            color: var(--color-blue);
            background-color: rgba(56, 161, 219, 0.1);
            text-decoration: none;
          }
        }
      }
    }
    
    .header-center {
      flex: 1;
      text-align: center;
      
      .file-title {
        font-size: 18px;
        font-weight: 600;
        color: #333;
        margin: 0;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 400px;
        display: inline-block;
        padding: 8px 16px;
        border-radius: 8px;
        background-color: rgba(255, 255, 255, 0.8);
        backdrop-filter: blur(4px);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.03);
      }
    }
    
    .header-right {
      flex: 2;
      display: flex;
      justify-content: flex-end;
      gap: 12px;
      
      .action-btn {
        display: flex;
        align-items: center;
        gap: 8px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 14px;
        font-weight: 500;
        padding: 10px 18px;
        transition: all 0.3s ease;
        
        .icon-sm {
          width: 16px;
          height: 16px;
        }
        
        &:hover {
          transform: translateY(-2px);
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
        }
      }
      
      .back-btn {
        background: rgba(255, 255, 255, 0.9);
        border: 1px solid #eee;
        color: #555;
        
        &:hover {
          background-color: #fff;
          border-color: #ddd;
        }
      }
      
      .download-btn {
        background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        color: white;
        border: none;
        box-shadow: 0 4px 10px rgba(79, 172, 254, 0.2);
        
        &:hover {
          background: linear-gradient(135deg, #4facfe 0%, #00c6fb 100%);
          box-shadow: 0 6px 15px rgba(79, 172, 254, 0.3);
        }
      }
    }
  }
  
  .preview-content {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    overflow: auto;
    padding: 32px;
    
    // 加载状态
    .loading-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100%;
      
      .loading-spinner {
        width: 48px;
        height: 48px;
        border: 3px solid rgba(56, 161, 219, 0.1);
        border-top: 3px solid var(--color-blue);
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 20px;
      }
      
      p {
        color: #666;
        font-size: 16px;
        font-weight: 500;
      }
    }
    
    // 错误状态
    .error-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100%;
      max-width: 450px;
      background-color: white;
      padding: 40px;
      border-radius: 16px;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
      
      .error-icon {
        width: 70px;
        height: 70px;
        border-radius: 50%;
        background: linear-gradient(135deg, #ff4d4f 0%, #ff7875 100%);
        color: white;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 32px;
        font-weight: bold;
        margin-bottom: 24px;
        box-shadow: 0 8px 20px rgba(255, 77, 79, 0.2);
      }
      
      h3 {
        font-size: 22px;
        color: #333;
        margin-bottom: 16px;
        font-weight: 600;
      }
      
      p {
        color: #666;
        margin-bottom: 28px;
        text-align: center;
        line-height: 1.6;
        font-size: 15px;
      }
      
      .error-actions {
        display: flex;
        gap: 16px;
        
        button {
          padding: 12px 24px;
          border-radius: 10px;
          cursor: pointer;
          font-size: 15px;
          font-weight: 500;
          transition: all 0.3s ease;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8px;
          
          &.btn-primary {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            border: none;
            box-shadow: 0 4px 10px rgba(79, 172, 254, 0.3);
            
            &:hover {
              transform: translateY(-2px);
              box-shadow: 0 8px 20px rgba(79, 172, 254, 0.4);
              background: linear-gradient(135deg, #4facfe 0%, #00c6fb 100%);
            }
            
            &:active {
              transform: translateY(-1px);
              box-shadow: 0 4px 8px rgba(79, 172, 254, 0.4);
            }
          }
          
          &.btn-outline {
            background-color: white;
            color: #555;
            border: 1px solid #ddd;
            
            &:hover {
              background-color: #f9f9f9;
              border-color: #ccc;
              transform: translateY(-2px);
              box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
            }
          }
        }
      }
    }
    
    // 图片预览
    .image-preview {
      max-width: 100%;
      max-height: 100%;
      display: flex;
      justify-content: center;
      align-items: center;
      background-color: #f5f5f5;
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
      
      img {
        max-width: 100%;
        max-height: 80vh;
        object-fit: contain;
        border-radius: 8px;
        transition: transform 0.3s ease;
        
        &:hover {
          transform: scale(1.02);
        }
      }
    }
    
    // 视频预览
    .video-preview {
      width: 100%;
      height: 100%;
      display: flex;
      justify-content: center;
      align-items: center;
      background-color: #000;
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
      
      video {
        max-width: 100%;
        max-height: 80vh;
        border-radius: 8px;
      }
    }
    
    // 音频预览
    .audio-preview {
      width: 100%;
      padding: 20px;
      display: flex;
      justify-content: center;
      
      .audio-card {
        display: flex;
        align-items: center;
        gap: 30px;
        background: linear-gradient(145deg, #fff, #f8f9fa);
        padding: 40px;
        border-radius: 20px;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
        max-width: 650px;
        width: 100%;
        
        .audio-icon-container {
          display: flex;
          justify-content: center;
          align-items: center;
          width: 90px;
          height: 90px;
          background: linear-gradient(135deg, #e3fdf5 0%, #ffe6fa 100%);
          border-radius: 50%;
          box-shadow: 0 8px 20px rgba(0, 0, 0, 0.05);
          
          .audio-icon {
            width: 45px;
            height: 45px;
            color: var(--color-blue);
          }
        }
        
        .audio-info {
          flex: 1;
          
          .audio-name {
            font-size: 18px;
            font-weight: 600;
            color: #333;
            margin-bottom: 20px;
          }
          
          audio {
            width: 100%;
            height: 40px;
            outline: none;
          }
        }
      }
    }
    
    // PDF预览
    .pdf-preview {
      width: 100%;
      height: 100%;
      background-color: white;
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
      
      iframe {
        width: 100%;
        height: 100%;
        border: none;
      }
    }

    // 文本文件预览
    .text-preview {
      width: 100%;
      height: 100%;
      background-color: white;
      border-radius: 12px;
      overflow: auto;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
      padding: 20px;
      
      .text-content {
        width: 100%;
        height: 100%;
        
        pre {
          font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
          font-size: 14px;
          line-height: 1.6;
          white-space: pre-wrap;
          word-break: break-all;
          color: #333;
          padding: 20px;
          margin: 0;
          overflow: auto;
          max-height: 70vh;
          
          &.code-content {
            background-color: #f5f5f5;
            border-radius: 8px;
            border: 1px solid #eee;
            padding: 16px;
          }
        }
      }
      
      .markdown-content {
        padding: 20px;
        line-height: 1.6;
        color: #333;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
        
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

      .html-preview-container {
        width: 100%;
        height: 100%;
        display: flex;
        flex-direction: column;
        
        .preview-controls {
          display: flex;
          gap: 12px;
          padding: 10px 20px;
          background-color: #f9f9fa;
          border-bottom: 1px solid #eee;
          
          button {
            padding: 8px 16px;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            border: none;
            
            &.preview-toggle-btn {
              background-color: #e5e7eb;
              color: #374151;
              
              &:hover {
                background-color: #d1d5db;
              }
              
              &.active {
                background-color: #3b82f6;
                color: white;
              }
            }
            
            &.open-new-window-btn {
              background-color: white;
              border: 1px solid #d1d5db;
              color: #4b5563;
              
              &:hover {
                background-color: #f9fafb;
                border-color: #9ca3af;
              }
            }
          }
        }
        
        .html-render {
          flex: 1;
          padding: 0;
          
          .html-iframe {
            width: 100%;
            height: 100%;
            min-height: 70vh;
            border: none;
            background-color: white;
          }
        }
      }
    }
    
    // 不支持预览的文件类型
    .unsupported-preview {
      display: flex;
      justify-content: center;
      align-items: center;
      height: 100%;
      
      .unsupported-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        background: linear-gradient(145deg, #fff, #f8f9fa);
        padding: 50px;
        border-radius: 20px;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
        max-width: 440px;
        text-align: center;
        
        .unsupported-icon {
          margin-bottom: 30px;
          background: linear-gradient(135deg, #e3fdf5 0%, #ffe6fa 100%);
          width: 110px;
          height: 110px;
          border-radius: 50%;
          display: flex;
          justify-content: center;
          align-items: center;
          box-shadow: 0 8px 20px rgba(0, 0, 0, 0.05);
          
          .file-icon {
            width: 55px;
            height: 55px;
            color: #999;
            
            &.image-icon { color: var(--color-blue); }
            &.video-icon { color: #f50; }
            &.audio-icon { color: #73d13d; }
            &.code-icon { color: #722ed1; }
            &.pdf-icon { color: #f5222d; }
            &.archive-icon { color: #fa8c16; }
            &.spreadsheet-icon { color: #52c41a; }
            &.presentation-icon { color: #eb2f96; }
          }
        }
        
        .unsupported-message {
          h3 {
            font-size: 22px;
            color: #333;
            margin-bottom: 16px;
            font-weight: 600;
          }
          
          p {
            color: #666;
            margin-bottom: 28px;
            line-height: 1.6;
            font-size: 15px;
          }
          
          .file-info {
            display: flex;
            flex-direction: column;
            gap: 10px;
            margin-bottom: 28px;
            background-color: rgba(245, 247, 250, 0.5);
            border-radius: 10px;
            padding: 16px 20px;
            font-size: 14px;
            color: #555;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.03);
            
            .file-info-item {
              display: flex;
              justify-content: space-between;
              align-items: center;
              border-bottom: 1px dashed rgba(0, 0, 0, 0.05);
              padding-bottom: 8px;
              
              &:last-child {
                border-bottom: none;
                padding-bottom: 0;
              }
              
              .label {
                font-weight: 500;
                color: #333;
              }
              
              .value {
                font-weight: 400;
                color: #666;
                max-width: 250px;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
              }
            }
          }
          
          .button-container {
            display: flex;
            justify-content: center;
            margin-top: 20px;
          }
          
          .download-button {
            display: flex;
            flex-direction: column;
            align-items: center;
            text-decoration: none;
            transition: all 0.3s ease;
            
            &:hover {
              transform: translateY(-3px);
              
              .download-button-icon {
                box-shadow: 0 8px 20px rgba(79, 172, 254, 0.4);
                background: linear-gradient(135deg, #4facfe 0%, #00c6fb 100%);
              }
              
              .download-button-text {
                color: #4facfe;
              }
            }
            
            .download-button-icon {
              width: 60px;
              height: 60px;
              background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
              border-radius: 50%;
              display: flex;
              justify-content: center;
              align-items: center;
              margin-bottom: 12px;
              box-shadow: 0 5px 15px rgba(79, 172, 254, 0.3);
              transition: all 0.3s ease;
              
              .icon {
                width: 28px;
                height: 28px;
                color: white;
              }
            }
            
            .download-button-text {
              font-size: 15px;
              font-weight: 500;
              color: #666;
              transition: color 0.3s ease;
            }
          }
          
          button {
            display: none; /* 隐藏旧按钮 */
          }
        }
      }
    }
  }
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* 响应式调整 */
@media screen and (max-width: 768px) {
  .file-preview-container {
    .preview-header {
      .header-content {
        flex-direction: column;
        gap: 12px;
        padding: 12px;
        
        .header-left {
          width: 100%;
        }
        
        .header-center {
          width: 100%;
          order: -1;
          
          .file-title {
            max-width: 100%;
          }
        }
        
        .header-right {
          width: 100%;
          justify-content: space-between;
        }
      }
    }
    
    .preview-content {
      padding: 16px;
      
      .audio-preview .audio-card {
        flex-direction: column;
        padding: 30px;
        
        .audio-icon-container {
          margin-bottom: 20px;
        }
      }

      .html-preview-container .preview-controls {
        flex-direction: column;
        gap: 8px;
        padding: 8px 16px;
      }
      
      .unsupported-preview .unsupported-card {
        padding: 30px;
      }
    }
  }
}

.code-content {
  width: 100%;
  height: 100%;
  padding: 20px;
  overflow: auto;

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

// 分享模式 logo 样式
.logo-link {
  text-decoration: none;

  .logo-text {
    font-size: 18px;
    font-weight: 600;
    color: #333;
  }
}

// 密码验证容器
.password-container {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;

  .password-card {
    background: white;
    border-radius: 16px;
    padding: 40px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
    max-width: 400px;
    width: 100%;
    text-align: center;

    .lock-icon {
      margin-bottom: 20px;

      .icon-lg {
        width: 60px;
        height: 60px;
        color: var(--color-blue, #4facfe);
      }
    }

    h3 {
      font-size: 20px;
      color: #333;
      margin-bottom: 16px;
    }

    .file-info-brief {
      display: flex;
      flex-direction: column;
      gap: 4px;
      margin-bottom: 24px;
      color: #666;
      font-size: 14px;

      .file-name {
        font-weight: 500;
        color: #333;
        word-break: break-all;
      }
    }

    .password-input-group {
      display: flex;
      gap: 8px;
      margin-bottom: 16px;

      .password-input {
        flex: 1;
        padding: 14px 16px;
        border: 1px solid #d9d9d9;
        border-radius: 10px;
        font-size: 15px;
        outline: none;
        transition: border-color 0.3s;

        &:focus {
          border-color: var(--color-blue, #4facfe);
        }
      }

      .toggle-password {
        padding: 12px;
        background: #f5f5f5;
        border: none;
        border-radius: 10px;
        cursor: pointer;
        transition: background-color 0.3s;

        &:hover {
          background: #e8e8e8;
        }

        .icon-sm {
          width: 20px;
          height: 20px;
          color: #666;
        }
      }
    }

    .verify-btn {
      width: 100%;
      padding: 14px;
      background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
      color: white;
      border: none;
      border-radius: 10px;
      font-size: 16px;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.3s ease;
      box-shadow: 0 4px 10px rgba(79, 172, 254, 0.2);

      &:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 8px 20px rgba(79, 172, 254, 0.4);
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }
    }
  }
}

// 分享模式响应式
@media screen and (max-width: 768px) {
  .password-container .password-card {
    padding: 24px;
    margin: 0 16px;
  }
}
</style> 