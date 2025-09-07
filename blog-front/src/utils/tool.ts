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
