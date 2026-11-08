# DH-Blog 前端

DH-Blog 的后台与前台界面。完整的项目说明、架构与部署方式见[根目录 README](../README.md)，
这里只记录前端自己的东西。

## 技术栈

Vue 3 + TypeScript + Vite + Element Plus + Tailwind CSS v4（CSS-first 配置，没有 `tailwind.config.js`）。
包管理器是 **bun**，不要用 npm / pnpm / yarn。

## 常用命令

```bash
bun install
bun run dev              # 开发服务器，接口指向 http://localhost:2233/api（见 .env）
bun run build:type-check # vue-tsc 类型检查 + 构建；前端唯一的自动化验证手段
bun run build            # 只构建
bunx knip                # 未使用的文件 / 导出 / 依赖，CI 同步执行
```

## 目录

- `src/views/frontend/` — 博客、网盘（`webdav/`）、影院
- `src/views/backend/` — 后台各页面，按功能分目录
- `src/components/` — 组件，`backend/` 与 `frontend/` 对应两侧
- `src/api/` — 接口封装，统一走 `api/axios.ts` 的实例，返回值已是拆包后的 `data`
- `src/store/index.ts` — 全部 Pinia store 集中在这里
- `public/pet/` — 桌宠素材包，运行期动态加载，不经过 Vite 打包

## 几个约定

- 路由是 **hash 模式**，地址形如 `/#/admin/dashboard`。
- Vue API 与 Element Plus 组件自动导入；`auto-imports.d.ts` / `components.d.ts` 是生成物但已入库，改完要一起提交。
- 不写自定义 CSS，一律用 Tailwind 工具类 + Element Plus 组件。
- `typescript` 装的是 `typescript-native-bridge`（tsgo 后端），`vue-tsc` 能直接跑；运行时看到 `▎ TNB ACTIVE` 说明生效。
- `vue-tsc` 钉在 3.3.13，升级前先看根目录 `CLAUDE.md` 的「已知的坑」。
