#!/bin/bash

set -e

# 变量
BINARY_NAME=dhblog
BACKEND_DIR="../blog-backend"
FRONTEND_DIR="../blog-front"
DEPLOY_DIR="$(pwd)/build"
EMBED_DIR="$BACKEND_DIR/internal/frontend/dist"

# 1. 准备目录
rm -rf "$DEPLOY_DIR" "$EMBED_DIR"
mkdir -p "$DEPLOY_DIR" "$EMBED_DIR"

# 2. 构建前端
echo "构建前端..."
(cd "$FRONTEND_DIR" && pnpm build)

# 3. 嵌入前端到后端
echo "准备后端嵌入前端文件..."
cp -r "$FRONTEND_DIR/dist"/* "$EMBED_DIR/"

# 4. 多平台构建
build_for_platform() {
    local os=$1
    local arch=$2
    local ext=$3
    local output="$DEPLOY_DIR/${BINARY_NAME}${ext}"
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
rm -rf "$EMBED_DIR"

echo "全部构建完成！产物在 $DEPLOY_DIR" 