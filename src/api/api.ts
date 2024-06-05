import { Article } from '@/types/Article'
import { Category } from '@/types/Category'
import { Page, PageResult } from '@/types/Page'
import { Tag } from '@/types/Tag'
import request from '@/api/axios'

/**
 * 新增文章
 * @param data 
 * @returns 
 */
export const addArticle = (data: Article<any>): Promise<Article<any>> => {
  return request.post('/article', data)
}

/**
 * 更新文章
 * @param data 
 * @returns 
 */
export const updateArticle = (data: Article<any>): Promise<Article<any>> => {
  return request.put('/article', data)
}

/**
 * 查询文章详情
 * @param id
 * @returns
 */
export const getArticleInfo = (id: String): Promise<PageResult<Article<any>>> => {
  return request.get('/article/' + id)
}

/**
 * 查询文章列表（分页查询）
 * @param data
 * @returns
 */
export const getArticleList = (data: Page): Promise<PageResult<Article<any>>> => {
  return request.post('/article/list', data)
}

/**
 * 获取所有文章分类
 * @returns
 */
export const getArticleCategoryList = (): Promise<Category[]> => {
  return request.get('/article/category')
}

/**
 * 获取所有文章标签
 * @returns
 */
export const getArticleTagList = (): Promise<Tag[]> => {
  return request.get('/article/tag')
}