# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览

DH-Blog 是一个个人内容管理系统（博客 + 个人网盘 + WebDAV + AI 搜索网关）。最终产物是**单个自包含可执行文件**：Vue 前端通过 `go:embed` 打进 Go 后端，运行时在可执行文件同级自建 `data/` 目录（SQLite、config.yaml、上传文件）。

- `blog-backend/` — Go 1.26 + Gin + GORM(SQLite) + Viper，模块名 `dh-blog`
- `blog-front/` — Vue 3 + TS + Vite + Element Plus + Tailwind v4，包管理器是 **bun**
- `blog-deploy/` — 跨平台打包脚本 + 本地开发运行目录
- `docs/` — 中文设计文档，**写在实现之前**，不代表已落地

## 常用命令

### 后端（`blog-backend/`）

```bash
go test ./...                                      # 全部测试
go test ./internal/modules/article -run TestXxx -v # 单个测试
go vet ./...
go build ./cmd/blog-backend
```

**编译前必须先准备嵌入目录**：`internal/frontend/dist/` 被 gitignore，而 `embed.go` 声明了 `//go:embed all:dist`。全新 checkout 直接编译会失败：

```
internal/frontend/embed.go:17:12: pattern all:dist: no matching files found
```

放任意占位文件即可解除（该错误只影响 `internal/frontend` 及其下游 `router` / `app` / `cmd`，其余包的测试不受影响）：

```bash
mkdir -p blog-backend/internal/frontend/dist && \
  printf '<!doctype html>' > blog-backend/internal/frontend/dist/index.html
```

本仓库的开发编译约定是把产物输出到 `blog-deploy/backend/`，该目录下的 `data/dhblog.db` 是开发数据库。改完后端要编译验证，但**不要运行**编译产物：

```bash
cd blog-backend && go build -o ../blog-deploy/backend/dhblog_dev ./cmd/blog-backend
```

`blog-backend/.air.toml` 已过期（把 `../blog-deploy/backend` 当成源码目录，那里没有 Go 文件），air 热重载不可用。

### 前端（`blog-front/`）

```bash
bun install
bun run dev              # Vite dev server；.env 指向 http://localhost:2233/api
bun run build:type-check # vue-tsc 类型检查 + 构建，前端唯一的自动化验证手段
bun run build
```

统一用 **bun**，不要用 npm/pnpm。前端没有测试框架，也没有 ESLint/Prettier 配置。

### 打包发布

```bash
cd blog-deploy && ./build.sh
# 构建前端 → 拷进 internal/frontend/dist → 交叉编译 darwin-arm64 / windows-amd64 / linux-amd64
```

## 架构

### 组合根：`internal/app/registry.go`

后端只有一个装配点。`moduleRegistrations` 切片同时决定三件事：

1. 路由注册顺序
2. `AutoMigrate` 的模型清单（`app.SchemaModels()` 汇总各模块的 `MigrationModels()`）
3. 模块之间的依赖装配

`buildContext` 是惰性 + 记忆化的 DI 容器（`ctx.article()`、`ctx.system()`…），因此路由顺序不必等于依赖顺序。新增模块 = 在 `moduleRegistrations` 追加一项 + 模块内实现 `MigrationModels()` 与 `RegisterRoutes()`；`registry_test.go` 硬编码了模块名和顺序，需要同步更新。

后台组件的启停由 `buildContext.starts()` / `shutdowns()` 收集，`App.Start()` / `App.Shutdown()` 各执行一次。

### 垂直模块 `internal/modules/<name>/`

每个模块自带 `model.go` / `repository.go` / `service.go` / `handler*.go` / `module.go`，对外只暴露 `module.go` 里的类型。

**跨模块依赖一律由使用方定义窄接口**，实现方不需要知情，真正的接线只发生在 `registry.go`。例如 `article.AIService`、`article.CommentCounter`、`article.TagTaskScheduler`、`system.StorageRuntime`、`system.CommentPolicy`、`webdav.FileService`、`webdav.UserAuthenticator`。不要跨模块直接 import 对方的 repository / service。

`internal/platform/` 是不含业务逻辑的外部适配层：`platform/ai`（OpenAI 兼容的 chat completions）、`platform/search`（Brave / Tavily / Exa / Firecrawl 归一成一套 `Provider` 接口；配额、选路、落库属于 aigateway 模块）。

### 路由与鉴权面

`router.Module` 只有 `RegisterRoutes(*router.Routes)` 一个方法。`Routes` 提供四种挂载面：

| 挂载面 | 前缀 | 鉴权 |
|---|---|---|
| `PublicAPI` | `/api` | 无 |
| `AdminAPI` | `/api/admin` | JWT 中间件 |
| `AuthenticatedAPI(path)` | 任意 | JWT（files / share 用 `/api/files`） |
| `Engine` | 任意 | 自行处理：aigateway 用自己的 API Key，share 公开链接不鉴权，webdav 用 Basic Auth |

JWT 从 `Authorization` 头（`Bearer ` 前缀可选）或 `?token=` 读取。全局中间件顺序：CORS(`*`) → `IPMiddleware`（异步写访问日志 + IP 封禁拦截，跳过 `heartbeat` 与 `gateway` 两类资源）→ `ValidLoginMiddleware`（软校验，只写 `isLogin`）。

前端静态资源在所有模块之后注册：`NoRoute` 把非 `/api`、非 WebDAV 前缀的请求回落到 `index.html`，并注入 `window.__SERVER_CONFIG__.SERVER_URL = "/api"`。

### 数据与配置

- 迁移只有 GORM `AutoMigrate`，没有迁移文件。表结构变更 = 改 struct + 确认它在模块的 `MigrationModels()` 里。
- 共享基类 `model.BaseModel`（`id` / `createTime` / `updateTime` / 软删除）；时间字段用 `model.JSONTime`，JSON 固定输出 `"2006-01-02 15:04:05"`。
- **两套配置，不要混**：
  - `data/config.yaml`（Viper，重启生效）只放部署相关的启动参数：监听端口、DB 路径、JWT 密钥与有效期、WebDAV 开关、网关运行参数。给 `config.Config` 加字段必须同步 `config.Init()` 里的 `v.SetDefault`，否则不会落盘；默认值集合一旦变化，启动时会把旧文件备份成 `config_backup_<时间戳>.yaml` 再重写。
  - `system_settings` 表（system 模块，带缓存，热生效）放运行期可改的内容：站点信息、AI endpoint/key/model/提示词、文件存储路径、分片大小、评论开关。默认值见 `system/defaults.go` 的 `DefaultSettings()`；改默认提示词时要用 `legacyPromptDefault` 登记旧值，`ensureDefaults` 只在库里存的仍是旧默认值时才升级。
- SQLite 以 WAL 模式打开。测试用 `sqlite.Open(":memory:")` + 模块自己的 `MigrationModels()`（参考 `files/test_helpers_test.go`），测试文件与实现同目录。

### 响应约定

所有 JSON 走 `response.AjaxResult{code,msg,data}`，**`code == 1` 才是成功，HTTP 状态码通常仍为 200**（`FailWithCode` 例外，401/403 走真实状态码）。前端 axios 拦截器据此拆包：成功直接返回 `data`，否则 `notify.error(msg)` 并 reject；401 清 token 跳登录（并发去重），403 置 `isBan` 跳错误页。

### 后台任务

`internal/task` 是通用调度器：5 个 worker、队列 100、单任务 30s 超时、失败最多重试 10 次间隔 5s。业务不下沉到 task 包 —— article 模块通过 `RegisterTagGenerationHandler` / `RegisterSummaryGenerationHandler` 把 handler 注册进来，task 只负责排队和重试。

### AI 网关

`internal/modules/aigateway` 自成体系：自己的下游 API Key、按供应商的月度配额与费用上限、`auto` 选路策略（配额余量 0.7 + 负载均衡 0.3 加权，含熔断器与出站限速）、单次请求最多一次 fallback、原生透传端点（`/api/gateway/v1/{tavily,brave,exa,firecrawl}/...`）、以及 `/api/gateway/v1/mcp` 的 MCP Server。完整契约见 `docs/AI网关设计.md`。表名统一 `ai_gateway_*`。

### 前端

- 路由是 **hash 模式**（`createWebHashHistory`），URL 形如 `/#/admin/dashboard`；`/admin` 与 `/webdav` 在 `beforeEach` 里调 `userCheck()` 鉴权。
- `src/api/*.ts` 只用 `api/axios.ts` 的实例，所以 api 函数的返回值已经是拆包后的 `data`。
- Pinia store 全部集中在 `src/store/index.ts`。
- Vue API 与 Element Plus 组件自动导入；`auto-imports.d.ts` / `components.d.ts` 是生成物但**已纳入版本管理**，变更后要一起提交。
- Tailwind v4 走 `@tailwindcss/vite`，CSS-first 配置（`src/assets/css/tailwind.css` 的 `@import "tailwindcss"`），没有 `tailwind.config.js`。

## 工作约定

- 用中文对话。提交信息用 Conventional Commits，scope 取模块名（`feat(gateway):`、`fix(files):`、`refactor(article):`），描述可用中文。
- 改动尽量最小，不顺手重构无关代码。
- 后端：改完必须编译验证（见上），不要运行编译产物；面向用户的错误信息用中文。注释语言跟随所在文件 —— 新代码倾向用英文解释**为什么**（见 `registry.go`、`policy.go`、`middleware/ip.go`），老代码是中文。
- 前端：不写自定义 CSS，一律用 Tailwind + Element Plus 组件；除非明确要求，不要改样式。
- 不熟悉的库或框架先查官方文档（context7 或搜索）再动手，不要凭记忆写 API。
- 不要提交 `data/`、`dhblog.db`、上传内容、编译产物、`internal/frontend/dist/`。

## 已知的坑

- `docs/` 里的设计文档是**实现前**写的，可能尚未落地。以代码为准（`2026-08-10-文章摘要优化与批量生成设计.md` 已实现，并按最终实现回写过一次）。
- CI（`.github/workflows/build.yml`）用 pnpm 且依赖 `blog-front/pnpm-lock.yaml`，但仓库里只有 `bun.lock`，前端安装步骤与本地工具链不一致。
- `blog-front/README.md` 是旧版遗留（写的是 SpringBoot + MySQL），以根 `README.md` 为准。
- 首次启动时若 `users` 表为空，程序会在 **stdin** 交互式索要管理员用户名和密码，非交互环境下会卡住。
