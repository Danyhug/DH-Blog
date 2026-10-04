import { defineConfig } from 'vite'
import Vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import * as ElementPlusIcons from '@element-plus/icons-vue'
import { visualizer } from 'rollup-plugin-visualizer'
import tailwindcss from '@tailwindcss/vite'

import path from 'path'

// 模板里直接写 <Lock /> 这类 Element Plus 图标时按需引入，取代原来在 main.ts 里全局注册全部 293 个图标。
// src/components 下的同名组件（如 Comment、View、Loading）优先于解析器，不会被图标顶掉。
const elementPlusIconNames = new Set(Object.keys(ElementPlusIcons))
const ElementPlusIconsResolver = (name: string) =>
  elementPlusIconNames.has(name) ? { name, from: '@element-plus/icons-vue' } : undefined

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
      resolvers: [ElementPlusResolver(), ElementPlusIconsResolver],
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
        // md-editor-v3 的 MdEditor.mjs 要排除在外：CodeMirror 只经由它引入，归进 vendor 会被递归带进首屏；
        // 排除后它跟着唯一使用它的后台发布页单独成块。
        codeSplitting: {
          groups: [
            {
              name: 'vendor',
              test: /node_modules[\\/](vue-router|vue|pinia|element-plus|md-editor-v3(?![\\/]lib[\\/]es[\\/]MdEditor\.mjs))[\\/]/
            }
          ]
        }
      }
    }
  }
})
