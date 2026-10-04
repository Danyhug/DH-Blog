import router from '@/router/index'
import Icon from "@/components/Child/Icon.vue";

import { createApp } from 'vue'
import App from './App.vue'

import '@/assets/iconfont/iconfont.js'
import { MdEditor, MdPreview, MdCatalog, config as mdEditorConfig } from 'md-editor-v3';
import 'md-editor-v3/lib/style.css';
import 'element-plus/theme-chalk/dark/css-vars.css'
import { createPinia } from 'pinia';

import '@/assets/css/style.less'
import '@/assets/css/tailwind.css'

import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'

// 正文图片懒加载 + 异步解码：长文首屏不再和屏幕外的几十张图抢带宽，解码也不卡住滚动。
// md-editor 的图片插件是 markdown-it-image-figures，它自带这两个开关。
mdEditorConfig({
  markdownItPlugins: plugins => plugins.map(item =>
    item.type === 'image' ? { ...item, options: { ...item.options, lazy: true, async: true } } : item),
})

const app = createApp(App)
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

const pinia = createPinia()

app.component('MdEditor', MdEditor)
  .component('MdPreview', MdPreview)
  .component('MdCatalog', MdCatalog)
  .component('Icon', Icon)

app.use(router)
  .use(pinia)
  .use(ElementPlus, {
    locale: zhCn,
  })

app.mount('#app')
