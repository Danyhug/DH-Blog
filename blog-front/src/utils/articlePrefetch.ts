import { getArticleInfo } from '@/api/user'
import { useUserStore } from '@/store'
import { Article } from '@/types/Article'
import { formatDate, getArticleBg } from '@/utils/tool'

// ArticleView only mounts after the chunk loads and the previous view's leave transition ends,
// so starting the request from the router lets it run in parallel with both instead of after them.
// A single slot is enough: only one article navigation is in flight at a time.
let pending: { id: string; at: number; promise: Promise<Article<any>> } | null = null
// An aborted navigation leaves its request here; don't hand that out much later as fresh data.
const PENDING_TTL = 10_000

export const articleHeaderInfo = (article: Article<any>) => ({
  id: article.id || 0,
  title: article.title || '',
  created: article.createTime ? formatDate(article.createTime) : '',
  wordNum: article.wordNum || 0,
  timConSum: article.wordNum ? (article.wordNum / 400 + 0.5).toFixed(0) : '0',
  thumbnailUrl: getArticleBg(article.thumbnailUrl, article.id),
  tags: article.tags ?? [],
})

export const prefetchArticle = (id: string) => {
  if (pending?.id !== id || Date.now() - pending.at > PENDING_TTL) {
    const promise = getArticleInfo(id)
    // ArticleView handles the rejection; this only keeps an unclaimed failure from being reported as unhandled.
    promise.catch(() => {})
    pending = { id, at: Date.now(), promise }
  }

  // The list already carries title/date/word count/cover, so the banner and sidebar can show them
  // right away instead of a skeleton until the detail request returns (tags arrive with it).
  const store = useUserStore()
  const listed = store.articleList.find(item => String(item.id) === id)
  if (listed && String(store.homeHeaderInfo.id) !== id) {
    store.homeHeaderInfo = articleHeaderInfo(listed)
  }
}

export const takeArticle = (id: string) => {
  const hit = pending?.id === id && Date.now() - pending.at <= PENDING_TTL ? pending.promise : getArticleInfo(id)
  pending = null
  return hit
}
