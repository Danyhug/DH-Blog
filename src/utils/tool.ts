import { SERVER_URL } from "@/types/Constant"

export const getArticleBg = (url: string): string => {
  if (url.length === 0) {
    // 随机图片
    return ''
  }
  // 拼接url
  return `${SERVER_URL}/url`
}
