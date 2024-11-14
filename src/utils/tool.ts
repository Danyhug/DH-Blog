import { SERVER_URL, OSS_URL } from "@/types/Constant";

export const getArticleBg = (url: string = ""): string => {
  if (!url || url.length === 0) {
    // 随机图片
    return `${SERVER_URL}/article/image/random?${Math.random()}`;
  } else if (url.includes("defaultArticleImg/")) {
    return `${OSS_URL}/${url}`;
  }
  // 拼接url
  return `${SERVER_URL}/articleUpload/${url}`;
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
