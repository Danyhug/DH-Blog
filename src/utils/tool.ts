import { SERVER_URL } from "@/types/Constant"

export const getArticleBg = (url: string = ''): string => {
  if (!url || url.length === 0) {
    // 随机图片
    return `${SERVER_URL}/article/image/random?${Math.random()}`
  }
  // 拼接url
  return `${SERVER_URL}/upload/${url}`
}
