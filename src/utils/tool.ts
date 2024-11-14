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
