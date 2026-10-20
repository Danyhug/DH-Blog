#!/bin/bash
# 把桌宠素材包的 PNG 贴图转成无损 WebP。
#
# 素材包是照搬上游的（见 blog-front/public/pet/README.md），430 张贴图里有 7 套配色
# 是同一形状的换色版，PNG 编码下共 11.1 MB。无损 WebP 能压到 7.8 MB，而且像素完全一致
# （cwebp -lossless 是可逆编码），所以这不是"降质换体积"。
#
# 用法：./webp-pet.sh [素材包目录]
# 默认处理 ../blog-front/public/pet。脚本幂等：已经转换过的文件不会再动。
# 依赖 cwebp：macOS 用 brew install webp，Debian/Ubuntu 用 apt install webp。

set -e

PACK_DIR="${1:-../blog-front/public/pet}"

if [ ! -d "$PACK_DIR" ]; then
    echo "❌ 找不到素材包目录：$PACK_DIR"
    exit 1
fi

if ! command -v cwebp >/dev/null 2>&1; then
    echo "❌ 需要 cwebp：macOS 执行 brew install webp，Debian/Ubuntu 执行 apt install webp"
    exit 1
fi

echo "转换 PNG → 无损 WebP ..."
# -z 6 是压缩强度（0-9，越高压得越小、越慢），-m 6 是编码方法；贴图有 alpha，
# 无损模式会原样保留 alpha 通道
find "$PACK_DIR" -type f -name '*.png' -print0 \
    | xargs -0 -P 8 -I{} sh -c 'cwebp -lossless -z 6 -m 6 -quiet "$1" -o "${1%.png}.webp" && rm "$1"' _ {}

# 素材包里只有三处拼贴图文件名：figure.js 的两行模板字符串，model.json 的 20 个
# outfits src，figure.json 的 11 个 thumbs。都是 ".png" 紧跟引号或反引号的形式，
# 而 canvas.toDataURL('image/png') 那种 MIME 是 "/png"，不会被误伤。
echo "改写素材包里的引用 ..."
for manifest in "$PACK_DIR/whale/figure.js" "$PACK_DIR/whale/figure.json" "$PACK_DIR/whale/model.json"; do
    [ -f "$manifest" ] || continue
    perl -pi -e 's/\.png(?=["`])/.webp/g' "$manifest"
done

echo "✅ 完成，当前体积："
du -sh "$PACK_DIR"
