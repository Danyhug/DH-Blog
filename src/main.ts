import { createApp } from 'vue'
import App from './App.vue'
import router from '@/router/index'

import '@/assets/css/style.css'
import VMdPreview from '@kangc/v-md-editor/lib/preview';
import '@kangc/v-md-editor/lib/style/preview.css';
import githubTheme from '@kangc/v-md-editor/lib/theme/github.js';
import '@kangc/v-md-editor/lib/theme/style/github.css';

import '@/assets/iconfont/iconfont.js'

import hljs from 'highlight.js';

VMdPreview.use(githubTheme, {
  Hljs: hljs,
});

const app = createApp(App)

app.use(VMdPreview);
app.use(router)

app.mount('#app')
