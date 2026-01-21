# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DH-Blog is a full-stack blog system with a Go backend and Vue 3 frontend. The project uses a monorepo structure with three main directories:
- `blog-backend/` - Go backend with Gin framework
- `blog-front/` - Vue 3 + TypeScript frontend
- `blog-deploy/` - Unified deployment scripts and built binaries

## Build & Development Commands

### Backend (blog-backend/)
```bash
# Development with hot reload (requires air)
cd blog-backend && air

# Build production binary
cd blog-backend && go build -ldflags="-s -w" -o dhblog ./cmd/blog-backend
```

### Frontend (blog-front/)
```bash
# Install dependencies
cd blog-front && bun install

# Development server
cd blog-front && bun run dev

# Production build
cd blog-front && bun run build

# Build with type checking
cd blog-front && bun run build:type-check
```

### Full Build (blog-deploy/)
```bash
# Build for all platforms (darwin-arm64, windows-amd64, linux-amd64)
cd blog-deploy && ./build.sh
```

This embeds the frontend into the backend binary, producing a single self-contained executable.

## Architecture

### Backend Structure
The backend follows a layered architecture with manual dependency injection:

```
internal/
├── wire/app.go      # Dependency injection, wires all components together
├── config/          # Configuration loading (Viper, YAML)
├── router/          # Gin route definitions
├── handler/         # HTTP handlers (controllers)
├── service/         # Business logic
├── repository/      # Data access layer (GORM)
├── model/           # Domain models and database entities
├── middleware/      # JWT auth, IP tracking
├── dhcache/         # In-memory caching layer
├── task/            # Background task processing (AI tasks)
├── frontend/        # Embedded frontend static files (go:embed)
└── response/        # Standardized HTTP response helpers
```

Key patterns:
- `wire/app.go` manually creates and injects all dependencies (no code generation)
- Repository layer uses GORM with SQLite as default database
- JWT-based authentication with middleware protection for admin routes
- Frontend is embedded into the binary using `//go:embed`

### Frontend Structure
```
src/
├── api/             # Axios API client modules
├── views/           # Vue page components
├── components/      # Reusable Vue components
├── router/          # Vue Router configuration
├── store/           # Pinia state management
├── types/           # TypeScript type definitions
└── utils/           # Helper utilities
```

Key dependencies:
- Element Plus for UI components (auto-imported)
- md-editor-v3 for Markdown editing
- vue-echarts for charts
- marked + DOMPurify for Markdown rendering

### API Routes
- Public API: `/api/*` - Article listing, viewing, comments, user login
- Admin API: `/api/admin/*` - Protected by JWT middleware
- File API: `/api/files/*` - File management with chunk upload support
- Static files: `/api/uploads/*` - Uploaded content

### Configuration
Configuration is in `data/config.yaml` (auto-created on first run):
- Server port defaults to 2233
- SQLite database at `data/dhblog.db`
- JWT secret auto-generated
- Supports WebDAV for file uploads

### First Run
On first startup, if no admin user exists, the application prompts for username/password via stdin to create the admin account.

## 注意事项
- 不了解的框架或库，不要直接使用，先通过`use context7`官方文档或搜索相关资料，确认后再使用；
- 后端：每次在文件进行更改时，必须进行编译测试，编译后端文件到`/Users/danyhug/GolandProjects/DH-Blog/blog-deploy/backend` 中，每完成一个任务必须编译一次查看是否成功，所有任务完成后删除编译文件；编译成功后不要运行！
- 数据库在 `/Users/danyhug/GolandProjects/DH-Blog/blog-deploy/backend/data/dhblog.db`
- 前端：除非我要求，否则不要改任何样式；`npm`命令统一改为使用`bun`；不要写任何css样式，全部用tailwindcss框架和element-plus组件库
- 更改要尽量最小规模更新，不要更改无关代码，用中文对话
