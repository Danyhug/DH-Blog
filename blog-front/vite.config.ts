import { defineConfig } from 'vite'
import Vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { visualizer } from 'rollup-plugin-visualizer'

import path from 'path'

export default defineConfig({
  base: "./",

  plugins: [
    visualizer(), Vue(),
    AutoImport({ 
      resolvers: [ElementPlusResolver()], 
      imports: ['vue'] 
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],

  resolve: {
    alias: {
      "@": path.resolve(__dirname, './src')
    },
  },

  build: {
    sourcemap: false,
    target: ['es2020', 'edge88', 'firefox78', 'chrome87', 'safari14'], // 更新target配置，使用现代浏览器列表
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue', 'vue-router', 'pinia', 'element-plus', 'md-editor-v3']
        }
      }
    }
  }
})
