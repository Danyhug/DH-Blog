import router from '@/router/index'
import Icon from "@/components/Child/Icon.vue";

import { createApp } from 'vue'
import App from './App.vue'

import '@/assets/iconfont/iconfont.js'
import { MdEditor, MdPreview, MdCatalog } from 'md-editor-v3';
import 'md-editor-v3/lib/style.css';
import { createPinia } from 'pinia';

import '@/assets/css/style.less'

import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'

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
