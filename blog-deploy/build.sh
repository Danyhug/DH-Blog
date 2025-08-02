#!/bin/bash

set -e

# 变量
BINARY_NAME=dhblog
BACKEND_DIR="../blog-backend"
FRONTEND_DIR="../blog-front"
DEPLOY_DIR="$(pwd)"
EMBED_DIR="$BACKEND_DIR/internal/frontend/dist"

# 1. 准备目录
rm -rf "$EMBED_DIR" "$DEPLOY_DIR/backend"
mkdir -p "$EMBED_DIR" "$DEPLOY_DIR/backend"

# 2. 构建前端
echo "构建前端..."
(cd "$FRONTEND_DIR" && pnpm build)

# 3. 嵌入前端到后端
echo "嵌入前端到后端..."
cp -r "$FRONTEND_DIR/dist"/* "$EMBED_DIR/"

# 4. 多平台构建
build_for_platform() {
    local os=$1
    local arch=$2
    local ext=$3
    local output="$DEPLOY_DIR/backend/${BINARY_NAME}${ext}"
    echo "构建 $os/$arch ..."
    (cd "$BACKEND_DIR" && GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -o "$output" ./cmd/blog-backend)
    if [ $? -eq 0 ]; then
        echo "✅ $output"
    else
        echo "❌ $os/$arch 构建失败"
        exit 1
    fi
}

build_for_platform "darwin" "arm64" "-darwin-arm64"
build_for_platform "windows" "amd64" "-windows-amd64.exe"
build_for_platform "linux" "amd64" "-linux-amd64"

# 5. 清理中间文件
echo "清理中间文件..."

# 清理前端构建目录（保留嵌入到后端的文件）
if [ -d "$FRONTEND_DIR/dist" ]; then
    rm -rf "$FRONTEND_DIR/dist"
    echo "✅ 已清理前端构建目录: $FRONTEND_DIR/dist"
fi

# 清理嵌入目录（构建产物已包含在二进制中）
if [ -d "$EMBED_DIR" ]; then
    rm -rf "$EMBED_DIR"
    echo "✅ 已清理嵌入目录: $EMBED_DIR"
fi

# 清理Go构建缓存（可选，加快后续构建速度）
# (cd "$BACKEND_DIR" && go clean -cache -testcache 2>/dev/null || true)

echo "🎉 全部构建完成！最终产物："
echo "- 前端文件已嵌入到: $EMBED_DIR"
echo "- 后端二进制文件: $DEPLOY_DIR/backend/"
ls -la "$DEPLOY_DIR/backend/"