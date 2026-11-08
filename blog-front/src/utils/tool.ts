import { SERVER_URL, OSS_URL } from "@/types/Constant";

export const getArticleBg = (url: string = "", articleId?: string | number): string => {
  if (!url || url.length === 0) {
    // 使用CDN随机图片，根据文章ID生成稳定图片
    const providers = [
      `https://picsum.photos/seed/${articleId || 'default'}/400/250`,
    ];
    
    // 根据文章ID选择图片源，保证同一文章始终显示相同图片
    if (articleId) {
      const hash = simpleHash(String(articleId));
      return providers[hash % providers.length];
    }
    
    return providers[0];
  } else if (url.includes("defaultArticleImg/")) {
    return `${OSS_URL}/${url}`;
  }
  // 拼接url
  return `${SERVER_URL}/${url}`;
};

// 简单的哈希函数，用于生成稳定但随机的图片
function simpleHash(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // 转换为32位整数
  }
  return Math.abs(hash);
}

export const plusDate = (date: Date, add: number): Date => {
  const result = new Date(date);
  result.setDate(result.getDate() + add);
  return result;
};

export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout> | undefined;

  return function (this: ThisParameterType<T>, ...args: Parameters<T>): void {
    const context = this;
    clearTimeout(timeout);
    timeout = setTimeout(() => {
      func.apply(context, args);
    }, wait);
  };
}

/**
 * 格式化日期
 * @param dateString 格式为2023-01-01T00:00:00.000Z
 * @returns 格式为2023/1/1 0:0:0
 */
export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()} ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}`;
}

/**
 * 把字节数格式化成人读的大小。
 *
 * 全应用只此一份实现：网盘列表、上传弹窗、分享列表和预览页曾经各写一套，
 * 保留位数还不一样（1.5 MB / 1.50 MB 混着出现）。要改显示格式就改这里。
 *
 * @param bytes 字节数；0 或非法值统一显示 "0 B"
 */
export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";

  const units = ["B", "KB", "MB", "GB", "TB"];
  // 1023 B 仍按字节显示；超出 TB 的极端值不再往上一级换算，避免出现 undefined 单位
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  if (unitIndex === 0) return `${Math.round(bytes)} B`;
  return `${(bytes / 1024 ** unitIndex).toFixed(1)} ${units[unitIndex]}`;
}
