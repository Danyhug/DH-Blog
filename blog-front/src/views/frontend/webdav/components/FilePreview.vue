<template>
  <div class="file-preview-container">
    <!-- 顶部导航栏 -->
    <div class="preview-header">
      <div class="header-content">
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
        
        <div class="header-center">
          <h2 class="file-title">{{ file.name }}</h2>
        </div>
        
        <div class="header-right">
          <button class="action-btn back-btn" @click="$emit('close')">
            <ArrowLeftIcon class="icon-sm" />
            返回
          </button>
          <button class="action-btn download-btn" @click="downloadFile">
            <DownloadIcon class="icon-sm" />
            下载
          </button>
        </div>
      </div>
    </div>
    
    <div class="preview-content">
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
        <div class="error-actions">
          <button class="btn-primary" @click="retryPreview">重试</button>
          <button class="btn-outline" @click="downloadFile">下载文件</button>
        </div>
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
        <div class="audio-card">
          <div class="audio-icon-container">
            <MusicIcon class="audio-icon" />
          </div>
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
        <div class="unsupported-card">
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
  </div>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, ref, onMounted } from 'vue'
import request from '@/api/axios'
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
import axios from 'axios'

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
      responseType: 'blob'
    });
    
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

// 组件挂载时尝试加载预览
onMounted(() => {
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
  background-color: #fafafa;
  animation: fade-in 0.3s ease;
  position: relative;
  
  .preview-header {
    position: sticky;
    top: 0;
    z-index: 10;
    backdrop-filter: blur(10px);
    background-color: rgba(255, 255, 255, 0.85);
    border-radius: 0 0 16px 16px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
    padding: 8px 0;
    
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
        background-color: rgba(248, 249, 250, 0.7);
        padding: 10px 16px;
        border-radius: 50px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
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
        padding: 6px 12px;
        border-radius: 8px;
        background-color: rgba(255, 255, 255, 0.7);
        backdrop-filter: blur(4px);
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.03);
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
        gap: 6px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 14px;
        font-weight: 500;
        padding: 10px 16px;
        transition: all 0.3s ease;
        
        .icon-sm {
          width: 16px;
          height: 16px;
        }
        
        &:hover {
          transform: translateY(-2px);
          box-shadow: 0 4px 8px rgba(0, 0, 0, 0.08);
        }
      }
      
      .back-btn {
        background: rgba(255, 255, 255, 0.8);
        border: 1px solid #eee;
        color: #555;
        
        &:hover {
          background-color: #fff;
          border-color: #ddd;
        }
      }
      
      .download-btn {
        background-color: var(--color-blue);
        color: white;
        border: none;
        
        &:hover {
          background-color: #2c91c8;
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
    padding: 24px;
    
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
      padding: 36px;
      border-radius: 16px;
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
      
      .error-icon {
        width: 64px;
        height: 64px;
        border-radius: 50%;
        background-color: #ff4d4f;
        color: white;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 32px;
        font-weight: bold;
        margin-bottom: 20px;
        box-shadow: 0 4px 12px rgba(255, 77, 79, 0.3);
      }
      
      h3 {
        font-size: 20px;
        color: #333;
        margin-bottom: 12px;
        font-weight: 600;
      }
      
      p {
        color: #666;
        margin-bottom: 24px;
        text-align: center;
        line-height: 1.6;
      }
      
      .error-actions {
        display: flex;
        gap: 12px;
        
        button {
          padding: 10px 18px;
          border-radius: 8px;
          cursor: pointer;
          font-size: 14px;
          font-weight: 500;
          transition: all 0.2s ease;
          
          &.btn-primary {
            background-color: var(--color-blue);
            color: white;
            border: none;
            
            &:hover {
              background-color: #2c91c8;
              transform: translateY(-2px);
              box-shadow: 0 4px 8px rgba(56, 161, 219, 0.2);
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
              box-shadow: 0 4px 8px rgba(0, 0, 0, 0.05);
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
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
      
      img {
        max-width: 100%;
        max-height: 80vh;
        object-fit: contain;
        border-radius: 8px;
        transition: transform 0.3s ease;
        
        &:hover {
          transform: scale(1.01);
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
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
      
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
        gap: 24px;
        background-color: white;
        padding: 30px;
        border-radius: 16px;
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
        max-width: 600px;
        width: 100%;
        
        .audio-icon-container {
          display: flex;
          justify-content: center;
          align-items: center;
          width: 80px;
          height: 80px;
          background: linear-gradient(135deg, #e3fdf5 0%, #ffe6fa 100%);
          border-radius: 50%;
          box-shadow: 0 4px 10px rgba(0, 0, 0, 0.05);
          
          .audio-icon {
            width: 40px;
            height: 40px;
            color: var(--color-blue);
          }
        }
        
        .audio-info {
          flex: 1;
          
          .audio-name {
            font-size: 16px;
            font-weight: 600;
            color: #333;
            margin-bottom: 16px;
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
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
      
      iframe {
        width: 100%;
        height: 100%;
        border: none;
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
        background-color: white;
        padding: 40px;
        border-radius: 16px;
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
        max-width: 400px;
        text-align: center;
        
        .unsupported-icon {
          margin-bottom: 24px;
          background: linear-gradient(135deg, #e3fdf5 0%, #ffe6fa 100%);
          width: 100px;
          height: 100px;
          border-radius: 50%;
          display: flex;
          justify-content: center;
          align-items: center;
          box-shadow: 0 4px 10px rgba(0, 0, 0, 0.05);
          
          .file-icon {
            width: 50px;
            height: 50px;
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
            font-size: 20px;
            color: #333;
            margin-bottom: 12px;
            font-weight: 600;
          }
          
          p {
            color: #666;
            margin-bottom: 24px;
            line-height: 1.6;
          }
          
          button {
            background-color: var(--color-blue);
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.2s ease;
            
            &:hover {
              background-color: #2c91c8;
              transform: translateY(-2px);
              box-shadow: 0 4px 8px rgba(56, 161, 219, 0.2);
            }
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
        gap: 10px;
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
          justify-content: center;
        }
      }
    }
    
    .preview-content {
      padding: 12px;
      
      .audio-preview .audio-card {
        flex-direction: column;
        padding: 20px;
        
        .audio-icon-container {
          margin-bottom: 20px;
        }
      }
    }
  }
}
</style> 