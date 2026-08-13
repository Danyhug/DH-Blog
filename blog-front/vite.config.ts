import { defineConfig } from 'vite'
import Vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { visualizer } from 'rollup-plugin-visualizer'
import tailwindcss from '@tailwindcss/vite'

import path from 'path'

export default defineConfig({
  base: "./",

  plugins: [
    tailwindcss(),
    ...(process.env.ANALYZE === 'true' ? [visualizer()] : []),
    Vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
      imports: ['vue'],
      dts: 'auto-imports.d.ts'
    }),
    Components({
      resolvers: [ElementPlusResolver()],
      dts: 'components.d.ts'
    }),
  ],

  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, './src')
    },
  },

  build: {
    sourcemap: false,
    outDir: 'dist', // Ensure the output directory is 'dist'
    emptyOutDir: true, // 构建前清空输出目录
    target: ['es2020', 'edge88', 'firefox78', 'chrome87', 'safari14'], // 更新target配置，使用现代浏览器列表
    rolldownOptions: {
      output: {
        // Vite 8 起改用 Rolldown，对象写法的 manualChunks 已移除，等价配置是 codeSplitting.groups。
        // 依赖默认递归归组，因此这几个包的传递依赖同样会落到 vendor。
        codeSplitting: {
          groups: [
            {
              name: 'vendor',
              test: /node_modules[\\/](vue-router|vue|pinia|element-plus|md-editor-v3)[\\/]/
            }
          ]
        }
      }
    }
  }
})
