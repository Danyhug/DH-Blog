import router from '@/router/index'
import Icon from "@/components/Child/Icon.vue";

import { createApp } from 'vue'
import App from './App.vue'

import '@/assets/iconfont/iconfont.js'
import { MdEditor, MdPreview } from 'md-editor-v3';
import 'md-editor-v3/lib/style.css';
import { createPinia } from 'pinia';

const app = createApp(App)
const pinia = createPinia()

app.component('MdEditor', MdEditor)
  .component('MdPreview', MdPreview)
  .component('Icon', Icon)

app.use(router)
  .use(pinia)

app.mount('#app')
