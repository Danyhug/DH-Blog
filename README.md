<div align="center">

# DH-Blog

**AI 时代的个人中枢 —— 博客、网盘、WebDAV 与 AI 搜索网关，一个二进制文件全部装下**

*Your personal hub for the AI era: a blog, your files, and a search gateway for your AI agents — all in a single binary.*

[Go](https://go.dev/) · [Vue 3](https://vuejs.org/) · [SQLite](https://www.sqlite.org/) · [MCP](https://modelcontextprotocol.io/)

</div>

---

## 这是什么

DH-Blog 是一个自托管的个人内容中枢。它把四件事装进一个可执行文件：

| | |
| --- | --- |
| 📝 **博客** | Markdown 写作、AI 生成标签与摘要、加密文章、评论、知识星图 |
| 📦 **个人网盘** | 文件管理、分片秒传、分享链接，WebDAV 协议全平台挂载 |
| 🧠 **AI 搜索网关** | 统一调度 4 家搜索服务，为你的 AI Agent 提供联网检索能力 |
| 🚀 **单文件部署** | 前端嵌入后端，一个二进制跑起来就是完整站点，SQLite 存一切 |

运行后在同级目录自建 `data/`（数据库、配置、上传文件），备份就是拷走一个目录。

## 为什么是「AI 时代的中枢」

内容平台真正的价值不在展示，而在**让 AI 替你使用它们**。DH-Blog 内置的 AI 搜索网关是这个想法的落点：

- 你在后台配置 Brave / Tavily / Exa / Firecrawl 的密钥和配额
- 网关签发独立 API Key，分发给每一个 Agent（Claude Code、自建机器人……）
- Agent 只需要**一个入口**，选哪家、排队限速、配额控制、熔断回退全部由网关代劳
- 支持 MCP 协议，Claude Code 这类客户端一条命令接入，直接取代内置的 `WebSearch`

你的博客积累知识，你的网盘沉淀文件，你的网关让所有 Agent 有一个统一的联网出口 —— 这是属于你自己的 AI 基础设施。

## 特性总览

### 🧠 AI 搜索网关（核心）

- **统一搜索 API**：一套请求契约调度 Brave / Tavily / Exa / Firecrawl 四家服务，`provider=auto` 时自动选路
- **智能调度**：能力硬过滤 → 熔断剔除 → 配额过滤 → 负载均衡（剩余配额 × 0.7 + 权重 × 0.3）或按优先级，失败自动回退一次
- **原生透传端点**：`/tavily/search`、`/brave/web/search` 等，现有 SDK 改个 base URL 即可接入
- **MCP Server**：`/api/gateway/v1/mcp`，Claude Code 可直接挂载，工具参数按你的 Key 权限动态生成
- **密钥与计费治理**：每供应商多密钥轮换、失效自动停用、上游用量每小时同步、按月配额与美元费用双上限
- **可观测**：全量请求流水、缓存命中率、成功率、供应商配额进度的可视化看板
- **结果缓存 + 限流**：15 分钟缓存省配额；每 Key QPM / 月配额、每供应商出站令牌桶

### 📝 博客

- Markdown 写作（md-editor-v3），代码高亮、目录、图片上传
- **AI 自动生成标签与摘要**，后台可一键批量补全历史文章摘要
- 分类层级、标签云、浏览量统计
- 单篇密码加密文章、评论审核、知识星图可视化

### 📦 个人网盘与 WebDAV

- 目录树、上传下载、重命名、移动，大文件分片断点续传
- WebDAV 协议支持，Windows / macOS / 手机文件管理器直接挂载
- 文件分享链接（有效期、可选密码）

### 🛡️ 运维与安全

- 访问统计与图表（ECharts）、IP 封禁、操作事件日志
- JWT 认证、bcrypt 密码、网关 Key 与博客鉴权完全隔离
- 数据一键备份恢复；配置与运行期设置分离，改提示词、AI 参数无需重启

## 快速开始

### 直接运行

在 [Release](../../releases) 下载对应平台的可执行文件（macOS ARM / Windows amd64 / Linux amd64），运行：

```bash
./dhblog-darwin-arm64
```

首次启动会在终端引导你创建管理员账号（非交互环境下请先保证能输入 stdin），随后自动生成 `data/` 目录。

访问 <http://localhost:2233>，后台入口 `/admin`。

> 部署细节见 [blog-deploy/README.md](blog-deploy/README.md)

### 从源码构建

```bash
# 前端产物需要先占位，否则 go:embed 会失败
mkdir -p blog-backend/internal/frontend/dist
printf '<!doctype html>' > blog-backend/internal/frontend/dist/index.html

# 一键打包：构建前端 → 嵌入后端 → 交叉编译三平台
cd blog-deploy && ./build.sh
```

产物输出到 `blog-deploy/build/`。

### 接入你的 Agent（MCP）

后台「AI 网关 → 接入密钥」页创建一把 Key，然后：

```bash
claude mcp add --transport http dh-search https://你的域名/api/gateway/v1/mcp \
  --header "Authorization: Bearer gw_live_你的Key" --scope user
```

建议同时在 Claude Code 配置中 `"deny": ["WebSearch"]`，让内置搜索完全让位于你的网关。

也可以直接走 HTTP API：

```bash
curl -X POST https://你的域名/api/gateway/v1/search \
  -H "Authorization: Bearer gw_live_你的Key" \
  -H "Content-Type: application/json" \
  -d '{"query": "Go 1.26 新特性", "max_results": 5, "freshness": "month"}'
```

## 技术栈

| 层 | 选型 |
| --- | --- |
| 后端 | Go 1.26 · Gin · GORM · SQLite（WAL）· Viper |
| 前端 | Vue 3 · TypeScript · Vite · Element Plus · Tailwind v4 · Pinia（构建用 bun） |
| AI | OpenAI 兼容 Chat Completions（标签/摘要）· 搜索网关（Brave / Tavily / Exa / Firecrawl）· MCP |
| 打包 | `go:embed` 前端 → 单二进制，交叉编译三平台 |

### 架构

```
┌────────────────────────────────────────────┐
│              单可执行文件                    │
│  ┌──────────────┐  ┌─────────────────────┐  │
│  │  前端 Vue 3  │  │  后端 Go (Gin)      │  │
│  │  博客/网盘/   │→ │  article · files    │  │
│  │  后台/网关    │  │  webdav · share     │  │
│  │  (go:embed)  │  │  system · aigateway │  │
│  └──────────────┘  └──────────┬──────────┘  │
│                       SQLite  │  data/      │
└───────────────────────────────┼─────────────┘
                                ▼
        ┌───────────────────────────────────────┐
        │  platform/search 适配层                │
        │  Brave · Tavily · Exa · Firecrawl     │
        │  令牌桶限速 · 熔断器 · 用量同步        │
        └───────────────────────────────────────┘
```

后端按垂直模块组织（`internal/modules/*`），跨模块依赖一律由使用方定义窄接口，在组合根 `internal/app/registry.go` 统一装配。

## 项目结构

```
DH-Blog/
├── blog-backend/   # Go 后端（Gin + GORM + SQLite）
│   └── internal/
│       ├── app/         # 组合根（依赖装配、路由注册、迁移清单）
│       ├── modules/     # 垂直业务模块：article/files/webdav/share/
│       │                #   comment/system/aigateway/...
│       ├── platform/    # 外部适配层：ai（LLM）、search（搜索网关上游）
│       └── task/        # 通用异步任务调度器（AI 标签/摘要生成）
├── blog-front/     # Vue 3 前端（bun 管理依赖）
├── blog-deploy/    # 打包脚本与本地开发运行目录
└── docs/           # 设计文档（含 AI 网关完整契约）
```

## 开发

```bash
# 后端：编译验证（产物输出到 blog-deploy/backend/，不运行）
cd blog-backend && go test ./... && go build -o ../blog-deploy/backend/dhblog_dev ./cmd/blog-backend

# 前端：开发服务器（.env 指向 http://localhost:2233/api）
cd blog-front && bun install && bun run dev
```

- 前端唯一的自动化验证是 `bun run build:type-check`（vue-tsc + 构建）
- 数据库迁移使用 GORM `AutoMigrate`，表结构变更 = 改 struct 并确认在模块的 `MigrationModels()` 中
- 提交规范：Conventional Commits（`feat(gateway):`、`fix(files):` 等）

## 文档

- [AI 网关设计](docs/AI网关设计.md) —— 网关完整契约：选路策略、API Key 治理、MCP Server、上游用量同步、费用上限
- [文章摘要优化与批量生成设计](docs/2026-08-10-文章摘要优化与批量生成设计.md)
- [数据库设计文档](docs/数据库设计文档.md)
