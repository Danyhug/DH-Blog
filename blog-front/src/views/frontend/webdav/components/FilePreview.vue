<template>
  <div class="file-preview-container">
    <!-- 地址栏 -->
    <div class="browser-header">
      <div class="header-left">
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
      </div>
      <div class="header-right">
        <button class="back-btn" @click="$emit('close')">
          <ArrowLeftIcon class="icon-sm" />
          返回
        </button>
      </div>
    </div>
    
    <div class="preview-header">
      <div class="header-left">
        <div class="file-title">{{ file.name }}</div>
      </div>
      <div class="header-right">
        <button class="download-btn" @click="downloadFile">
          <DownloadIcon class="icon-sm" />
          下载
        </button>
      </div>
    </div>
    
    <div class="preview-content">
      <!-- 加载状态 -->
      <div v-if="isLoading" class="loading-container">
        <div class="loading-spinner"></div>
        <p>加载中...</p>
      </div>
      
      <!-- 错误状态 -->
      <div v-else-if="hasError" class="error-container">
        <div class="error-icon">!</div>
        <h3>预览失败</h3>
        <p>{{ errorMessage }}</p>
        <button class="btn-primary" @click="retryPreview">重试</button>
        <button class="btn-outline" @click="downloadFile">下载文件</button>
      </div>
      
      <!-- 图片预览 -->
      <div v-else-if="file.type === 'image'" class="image-preview">
        <img 
          :src="fileUrl" 
          :alt="file.name" 
          @load="onPreviewLoaded" 
          @error="onPreviewError('图片加载失败')" 
        />
      </div>
      
      <!-- 视频预览 -->
      <div v-else-if="file.type === 'video'" class="video-preview">
        <video 
          controls 
          :src="fileUrl"
          @loadeddata="onPreviewLoaded"
          @error="onPreviewError('视频加载失败')"
        >
          您的浏览器不支持视频播放
        </video>
      </div>
      
      <!-- 音频预览 -->
      <div v-else-if="file.type === 'audio'" class="audio-preview">
        <div class="audio-player">
          <MusicIcon class="audio-icon" />
          <div class="audio-info">
            <div class="audio-name">{{ file.name }}</div>
            <audio 
              controls 
              :src="fileUrl"
              @loadeddata="onPreviewLoaded"
              @error="onPreviewError('音频加载失败')"
            >
              您的浏览器不支持音频播放
            </audio>
          </div>
        </div>
      </div>
      
      <!-- PDF预览 -->
      <div v-else-if="file.type === 'pdf'" class="pdf-preview">
        <iframe 
          :src="fileUrl" 
          frameborder="0"
          @load="onPreviewLoaded"
          @error="onPreviewError('PDF加载失败')"
        ></iframe>
      </div>
      
      <!-- 不支持预览的文件类型 -->
      <div v-else class="unsupported-preview">
        <div class="unsupported-icon">
          <component :is="file.icon || FileIcon" class="file-icon" :class="getIconClass(file.type)" />
        </div>
        <div class="unsupported-message">
          <h3>无法预览此文件</h3>
          <p>该文件类型不支持在线预览，请下载后查看。</p>
          <button class="btn-primary" @click="downloadFile">下载文件</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, ref, onMounted } from 'vue'
import axios from 'axios'
import type { FileItem } from '../utils/types/file'
import { getDownloadUrl } from '@/api/file'
import { SERVER_URL } from '@/types/Constant'
import {
  ArrowLeftIcon,
  DownloadIcon,
  MusicIcon,
  HomeIcon,
  ChevronRightIcon,
  FileIcon
} from '../utils/icons'

// 定义路径段接口
interface PathSegment {
  id: string;
  name: string;
}

const props = defineProps<{
  file: FileItem
}>()

const emits = defineEmits<{
  (e: 'close'): void
}>()

// 从父组件注入路径导航信息
const pathSegments = inject<PathSegment[]>('pathSegments', [])
const navigateToRoot = inject('navigateToRoot', () => {})
const navigateToPathSegment = inject('navigateToPathSegment', (index: number) => {})

// 加载和错误状态
const isLoading = ref(true)
const hasError = ref(false)
const errorMessage = ref('')
const fileContent = ref<Blob | null>(null)

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
  console.log('FilePreview - 文件对象:', props.file);
  console.log('FilePreview - 文件ID:', props.file.id);
  
  if (props.file.id) {
    const url = getDownloadUrl(props.file.id);
    console.log('FilePreview - 生成的下载URL:', url);
    return url;
  }
  console.log('FilePreview - 文件ID不存在，无法生成下载URL');
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

  try {
    isLoading.value = true;
    hasError.value = false;
    
    const token = localStorage.getItem('token') || '';
    const url = `${SERVER_URL}/files/download/${props.file.id}`;
    
    console.log('FilePreview - 开始请求文件内容:', url);
    
    const response = await axios.get(url, {
      headers: {
        Authorization: token
      },
      responseType: 'blob'
    });
    
    console.log('FilePreview - 文件内容请求成功:', response);
    
    // 创建Blob URL
    fileContent.value = response.data;
    const blobUrl = URL.createObjectURL(response.data);
    
    // 根据文件类型处理预览
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
    
    onPreviewLoaded();
  } catch (error) {
    console.error('FilePreview - 请求文件内容失败:', error);
    onPreviewError('请求文件内容失败');
  }
}

// 预览加载成功
const onPreviewLoaded = () => {
  console.log('预览加载成功:', props.file.name, '文件ID:', props.file.id);
  isLoading.value = false;
  hasError.value = false;
}

// 预览加载失败
const onPreviewError = (message: string) => {
  console.error('预览加载失败:', props.file.name, '文件ID:', props.file.id, '错误信息:', message);
  console.error('预览URL:', fileUrl.value);
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

// 组件挂载时尝试加载预览
onMounted(() => {
  console.log('FilePreview组件挂载 - 文件信息:', props.file);
  // 尝试直接请求文件内容
  fetchFileContent();
})
</script>

<style lang="less" scoped>
.file-preview-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #ffffff;
  animation: fade-in 0.3s ease;
  
  .browser-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    flex-shrink: 0;
    padding: 0 0 10px 0;
    
    .header-left {
      .breadcrumb {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        background-color: #f8f9fa;
        padding: 10px 16px;
        border-radius: 50px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
        
        .icon-sm {
          cursor: pointer;
          color: #666;
          
          &:hover {
            color: #2a8aff;
          }
        }
        
        .path-segment {
          cursor: pointer;
          color: #666;
          font-weight: 500;
          padding: 2px 8px;
          border-radius: 4px;
          transition: all 0.2s ease;
          
          &:hover {
            color: #2a8aff;
            background-color: rgba(42, 138, 255, 0.1);
            text-decoration: none;
          }
        }
        
        .icon-xs {
          color: #aaa;
        }
      }
    }
    
    .header-right {
      .back-btn {
        display: flex;
        align-items: center;
        gap: 5px;
        background: none;
        border: 1px solid #ddd;
        color: #666;
        cursor: pointer;
        font-size: 14px;
        padding: 8px 12px;
        border-radius: 4px;
        
        &:hover {
          background-color: #f5f5f5;
        }
        
        .icon-sm {
          width: 16px;
          height: 16px;
        }
      }
    }
  }
  
  .preview-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 0;
    border-bottom: 1px solid #eee;
    margin-bottom: 20px;
    
    .header-left {
      .file-title {
        font-size: 18px;
        font-weight: 600;
        color: #333;
      }
    }
    
    .header-right {
      .download-btn {
        display: flex;
        align-items: center;
        gap: 5px;
        background-color: #2a8aff;
        color: white;
        border: none;
        cursor: pointer;
        font-size: 14px;
        padding: 8px 15px;
        border-radius: 4px;
        
        &:hover {
          background-color: #1a7aef;
        }
        
        .icon-sm {
          width: 16px;
          height: 16px;
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
    
    // 加载状态
    .loading-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100%;
      
      .loading-spinner {
        width: 40px;
        height: 40px;
        border: 3px solid #f3f3f3;
        border-top: 3px solid #2a8aff;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 15px;
      }
      
      p {
        color: #666;
        font-size: 16px;
      }
    }
    
    // 错误状态
    .error-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100%;
      
      .error-icon {
        width: 60px;
        height: 60px;
        border-radius: 50%;
        background-color: #ff4d4f;
        color: white;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 32px;
        font-weight: bold;
        margin-bottom: 15px;
      }
      
      h3 {
        font-size: 18px;
        color: #333;
        margin-bottom: 10px;
      }
      
      p {
        color: #666;
        margin-bottom: 20px;
        text-align: center;
      }
      
      button {
        margin: 5px;
        padding: 8px 15px;
        border-radius: 4px;
        cursor: pointer;
        font-size: 14px;
        
        &.btn-primary {
          background-color: #2a8aff;
          color: white;
          border: none;
          
          &:hover {
            background-color: #1a7aef;
          }
        }
        
        &.btn-outline {
          background-color: white;
          color: #666;
          border: 1px solid #ddd;
          
          &:hover {
            background-color: #f5f5f5;
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
      
      img {
        max-width: 100%;
        max-height: 80vh;
        object-fit: contain;
      }
    }
    
    // 视频预览
    .video-preview {
      width: 100%;
      height: 100%;
      display: flex;
      justify-content: center;
      align-items: center;
      
      video {
        max-width: 100%;
        max-height: 80vh;
      }
    }
    
    // 音频预览
    .audio-preview {
      width: 100%;
      padding: 20px;
      
      .audio-player {
        display: flex;
        align-items: center;
        gap: 20px;
        background-color: #f8f9fa;
        padding: 20px;
        border-radius: 10px;
        
        .audio-icon {
          width: 60px;
          height: 60px;
          color: #2a8aff;
        }
        
        .audio-info {
          flex: 1;
          
          .audio-name {
            font-size: 16px;
            font-weight: 500;
            color: #333;
            margin-bottom: 10px;
          }
          
          audio {
            width: 100%;
          }
        }
      }
    }
    
    // PDF预览
    .pdf-preview {
      width: 100%;
      height: 100%;
      
      iframe {
        width: 100%;
        height: 100%;
        border: none;
      }
    }
    
    // 不支持预览的文件类型
    .unsupported-preview {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 40px;
      text-align: center;
      
      .unsupported-icon {
        margin-bottom: 20px;
        
        .file-icon {
          width: 80px;
          height: 80px;
          color: #999;
          
          &.image-icon { color: #36c; }
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
          font-size: 18px;
          color: #333;
          margin-bottom: 10px;
        }
        
        p {
          color: #666;
          margin-bottom: 20px;
        }
        
        button {
          background-color: #2a8aff;
          color: white;
          border: none;
          padding: 8px 15px;
          border-radius: 4px;
          cursor: pointer;
          font-size: 14px;
          
          &:hover {
            background-color: #1a7aef;
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
</style> 