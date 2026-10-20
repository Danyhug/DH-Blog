#!/bin/bash
# 把前端构建产物铺进 go:embed 目录，并把文本资源预压缩成 .gz。
#
# 只保留 .gz 的原始文件是故意的：同一份 JS 同时嵌进二进制两遍（原始 + .gz）会让产物
# 白白大出 4 MB，而 /assets/ 只会发其中一份。后端 internal/frontend/embed.go 的
# serveAssets 对不接受 gzip 的客户端现场解压，所以删掉原始文件不影响兼容性。
#
# 用法：./prepare-assets.sh [前端 dist 目录] [embed 目录]
# 本地 build.sh 和 CI 都调这里，免得两边各写一套、不知不觉跑偏。

set -e

FRONTEND_DIST="${1:-../blog-front/dist}"
EMBED_DIR="${2:-../blog-backend/internal/frontend/dist}"

if [ ! -d "$FRONTEND_DIST" ]; then
    echo "❌ 前端构建产物不存在：$FRONTEND_DIST"
    exit 1
fi

# 先清空再拷：embed 目录是生成物，留着上一次的旧 chunk 会让它们被打进二进制
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
cp -r "$FRONTEND_DIST"/. "$EMBED_DIR"/

# 只压 assets/ 和 pet/ —— 这两处走 embed.go 的 serveAssets，它认识 .gz 并且能现场解压。
# dist 根目录的 vite.svg / manifest.json 由 serveCachedFile 直接 fs.ReadFile，删了源文件
# 会 404，所以留原样（加起来才几十 KB，不值得为它绕一圈）。
# 图片、音频本身已压缩，不在列表里；-n 不写入文件名和时间戳，产物可复现；
# 不加 -k，压缩后直接删掉原始文件（见文件头）
echo "预压缩文本资源 ..."
gzip_dirs=("$EMBED_DIR/assets")
if [ -d "$EMBED_DIR/pet" ]; then
    gzip_dirs+=("$EMBED_DIR/pet")
fi
find "${gzip_dirs[@]}" -type f \( -name '*.js' -o -name '*.mjs' -o -name '*.css' -o -name '*.svg' -o -name '*.json' \) \
    -exec gzip -9 -n {} +

echo "✅ 嵌入目录就绪：$EMBED_DIR（$(du -sh "$EMBED_DIR" | cut -f1)）"
