# WebDAV 组件

这是一套完整的 WebDAV 云存储界面组件，提供了美观的文件管理、上传、分享和设置功能。

## 组件结构

- `WebDAView.vue` - 主视图组件
- `SettingsModal.vue` - 设置弹窗
- `ShareLinkPopup.vue` - 分享链接弹窗
- `UploadModal.vue` - 上传文件弹窗
- `MobileView.vue` - 移动设备视图
- `icons.ts` - 图标组件
- `types/file.ts` - 类型定义

## 如何使用

### 基本导入

```vue
<template>
  <WebDAView />
</template>

<script setup lang="ts">
import { WebDAView } from '@/views/frontend/webdav'
</script>
```

### 使用单个组件

```vue
<template>
  <div>
    <button @click="showSettings = true">打开设置</button>
    <SettingsModal v-if="showSettings" @close="showSettings = false" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SettingsModal } from '@/views/frontend/webdav'

const showSettings = ref(false)
</script>
```

### 使用图标

```vue
<template>
  <button>
    <CloudIcon class="icon" />
    云同步
  </button>
</template>

<script setup lang="ts">
import { icons } from '@/views/frontend/webdav'
const { CloudIcon } = icons
</script>
```

## 自定义文件数据

你可以自定义传入的文件数据：

```vue
<template>
  <WebDAView :files="customFiles" :mobile-files="customMobileFiles" />
</template>

<script setup lang="ts">
import { WebDAView, type FileItem } from '@/views/frontend/webdav'

const customFiles: FileItem[] = [
  { name: "我的文档", type: "folder", size: "5 个项目", modified: "今天" },
  { name: "我的照片", type: "folder", size: "42 个项目", modified: "昨天" }
]

const customMobileFiles: FileItem[] = [
  { name: "我的文档", type: "folder", size: "5 个项目" },
  { name: "我的照片", type: "folder", size: "42 个项目" }
]
</script>
``` 