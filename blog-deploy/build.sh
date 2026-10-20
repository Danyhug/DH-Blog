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
# embed 目录先占个位：go:embed all:dist 在没有任何文件时会直接报错，前端构建期间
# 如果有人同时跑 go build 也不至于失败。真正的内容由第 3 步的 prepare-assets.sh 铺进去。
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"

# 2. 构建前端
echo "构建前端..."
# --frozen-lockfile：依赖与 bun.lock 不一致时直接失败，而不是悄悄装出另一套版本
(cd "$FRONTEND_DIR" && bun install --frozen-lockfile && bun run build -- --mode production)

# 3. 嵌入前端到后端（顺带把文本资源预压缩成 .gz，只留压缩版）
echo "嵌入前端到后端..."
# 后端对接受 gzip 的浏览器直接返回 .gz（internal/frontend/embed.go 的 serveAssets），
# 首屏 JS/CSS 传输量降到约 1/3；原始文件同时删掉，否则二进制里每份 JS 存两遍。
./prepare-assets.sh "$FRONTEND_DIR/dist" "$EMBED_DIR"

# 4. 多平台构建
build_for_platform() {
    local os=$1
    local arch=$2
    local ext=$3
    local output="$BUILD_DIR/${BINARY_NAME}${ext}"
    echo "构建 $os/$arch ..."
    # nomsgpack: 排除 gin v1.12 无条件引入的 msgpack 绑定及 ugorji codec，省约 5MB
    # CGO_ENABLED=0: SQLite 用的是纯 Go 驱动，关掉 cgo 后本机 darwin 产物也和交叉编译的一样是静态链接
    # （set -e 下失败会直接退出，不需要再判断 $?）
    (cd "$BACKEND_DIR" && CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -tags nomsgpack -trimpath -ldflags="-s -w" -o "$output" ./cmd/blog-backend) \
        || { echo "❌ $os/$arch 构建失败"; exit 1; }
    echo "✅ $output"
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

# 嵌入目录(EMBED_DIR)保留不删：internal/frontend 的 //go:embed 需要它，删掉后本地 go build / go test 会失败

echo "🎉 全部构建完成！最终产物："
echo "- 后端二进制文件: $BUILD_DIR/"
ls -la "$BUILD_DIR/"
