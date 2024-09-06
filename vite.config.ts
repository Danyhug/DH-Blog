import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { visualizer } from 'rollup-plugin-visualizer'
import path from 'path'

export default defineConfig({
  base: "./",
  esbuild: {
    drop: ['console', 'debugger'], // 删除 debugger
  },

  plugins: [
    visualizer(), vue(),
    AutoImport({ resolvers: [ElementPlusResolver()], }),
    Components({
      resolvers: [ElementPlusResolver()],
    })
  ],

  resolve: {
    alias: {
      "@": path.resolve(__dirname, './src')
    },
  },
  build: {
    sourcemap: false,
  }
})
