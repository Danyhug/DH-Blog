import { SERVER_URL, OSS_URL } from "@/types/Constant";

export const getArticleBg = (url: string = ""): string => {
  if (!url || url.length === 0) {
    // 随机图片
    return `${SERVER_URL}/upload/image/random?${Math.random()}`;
  } else if (url.includes("defaultArticleImg/")) {
    return `${OSS_URL}/${url}`;
  }
  // 拼接url
  return `${SERVER_URL}/${url}`;
};

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
  return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()} ${date.getHours()}:${date.getMinutes()}`;
}
