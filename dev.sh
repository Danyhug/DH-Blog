#!/usr/bin/env bash
# 本地开发环境一键启动：后端 air 热重载 + 前端 Vite HMR
# 用法: ./dev.sh
# 前端改代码即时生效；后端改 Go 源码后自动重新编译并重启（约 1-2 秒）。
# 访问 http://localhost:5173 实时预览整站。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AIR="$ROOT/.tools/bin/air"
PORT=2233

# Go 编译缓存放到项目内（~ 与 ~/Library 在部分受限环境下不可写）
export GOCACHE="$ROOT/.tools/gocache"

# 端口被占用说明有旧的后端进程在跑（比如手动启动的 dhblog_dev），先停掉再接管
if lsof -ti :$PORT -sTCP:LISTEN >/dev/null 2>&1; then
  echo "端口 $PORT 被旧后端进程占用，先停掉……"
  lsof -ti :$PORT -sTCP:LISTEN | xargs kill 2>/dev/null || true
  sleep 1
fi

# 后端热重载：air 监听 blog-backend 源码，编译并重启
cd "$ROOT/blog-backend"
"$AIR" &
AIR_PID=$!

# 前端 HMR；退出时把 air 一起停掉
trap 'kill "$AIR_PID" 2>/dev/null || true' EXIT INT TERM
cd "$ROOT/blog-front"
bun run dev
