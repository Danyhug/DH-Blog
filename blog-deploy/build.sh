#!/bin/bash

set -e

# 变量
BINARY_NAME=dhblog
BACKEND_DIR="../blog-backend"
FRONTEND_DIR="../blog-front"
DEPLOY_DIR="$(pwd)"
BUILD_DIR="$DEPLOY_DIR/build"
EMBED_DIR="$BACKEND_DIR/internal/frontend/dist"

# 1. 准备目录
[ ! -d "$BUILD_DIR" ] && mkdir -p "$BUILD_DIR"
[ -d "$EMBED_DIR" ] && rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"

# 2. 构建前端
echo "构建前端..."
(cd "$FRONTEND_DIR" && bun run build -- --mode production)

# 3. 嵌入前端到后端
echo "嵌入前端到后端..."
cp -r "$FRONTEND_DIR/dist"/* "$EMBED_DIR/"

# 4. 多平台构建
build_for_platform() {
    local os=$1
    local arch=$2
    local ext=$3
    local output="$BUILD_DIR/${BINARY_NAME}${ext}"
    echo "构建 $os/$arch ..."
    # nomsgpack: 排除 gin v1.12 无条件引入的 msgpack 绑定及 ugorji codec，省约 5MB
    (cd "$BACKEND_DIR" && GOOS=$os GOARCH=$arch go build -tags nomsgpack -trimpath -ldflags="-s -w" -o "$output" ./cmd/blog-backend)
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

# 清理前端构建目录（保留嵌入到后端的文件，供编译时使用）
if [ -d "$FRONTEND_DIR/dist" ]; then
    rm -rf "$FRONTEND_DIR/dist"
    echo "✅ 已清理前端构建目录: $FRONTEND_DIR/dist"
fi

# 注意：嵌入目录(EMBED_DIR)在编译后清理，避免//go:embed找不到文件
# 清理嵌入目录将在构建完成后进行

echo "🎉 全部构建完成！最终产物："
echo "- 后端二进制文件: $BUILD_DIR/"
ls -la "$BUILD_DIR/"
