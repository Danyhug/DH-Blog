import router from '@/router/index'
import Icon from "@/components/Child/Icon.vue";

import { createApp } from 'vue'
import App from './App.vue'

import '@/assets/iconfont/iconfont.js'
// MdEditor 只在后台发布页用，由 PublishView 自己引入：它带着 CodeMirror（约 600KB），全局注册会让每位博客访客都下载一遍
import { MdPreview, MdCatalog, config as mdEditorConfig } from 'md-editor-v3';
import 'md-editor-v3/lib/style.css';
import 'element-plus/theme-chalk/dark/css-vars.css'
import { createPinia } from 'pinia';

import '@/assets/css/style.less'
import '@/assets/css/tailwind.css'

// Element Plus 组件与图标都由 vite.config.ts 里的解析器按用到的才引入；
// 不再 app.use(ElementPlus)：整包安装会把全部组件（压缩前约 1.7MB）打进每个页面都要加载的 vendor。
// 整包安装顺带做的两件事在这里补上：全局中文语言包，以及让命令式弹窗拿到应用上下文。
import { ElMessageBox, ElNotification, provideGlobalConfig } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'

// 正文图片懒加载 + 异步解码：长文首屏不再和屏幕外的几十张图抢带宽，解码也不卡住滚动。
// md-editor 的图片插件是 markdown-it-image-figures，它自带这两个开关。
mdEditorConfig({
  markdownItPlugins: plugins => plugins.map(item =>
    item.type === 'image' ? { ...item, options: { ...item.options, lazy: true, async: true } } : item),
})

const app = createApp(App)
// global = true：ElMessageBox / ElNotification 这类不在组件树里的弹窗也读这份配置，按钮才是「确定 / 取消」
provideGlobalConfig({ locale: zhCn }, app, true)

const pinia = createPinia()

app.component('MdPreview', MdPreview)
  .component('MdCatalog', MdCatalog)
  .component('Icon', Icon)

app.use(router)
  .use(pinia)
  // 安装时记下 app 的上下文，弹窗内容里的全局组件（如 Icon）才能解析
  .use(ElMessageBox)
  .use(ElNotification)

app.mount('#app')
