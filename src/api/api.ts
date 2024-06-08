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
export const getArticleInfo = (id: String): Promise<Article<any>> => {
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

// ********** 标签操作 **********

/**
 * 新增标签
 * @param data
 */
export const addTag = (data: Tag): Promise<Tag> => {
  return request.post('/article/tag', data)
}

/**
 * 更新标签
 * @param data
 */
export const updateTag = (data: Tag): Promise<Tag> => {
  return request.put('/article/tag', data)
}

/**
 * 删除标签
 * @param id
 */
export const deleteTag = (id: String): Promise<any> => {
  return request.delete('/article/tag/' + id)
}

// ********** 分类操作 **********

/**
 * 新增分类
 * @param data
 */
export const addCategory = (data: Category): Promise<Category> => {
  return request.post('/article/category', data)
}

/**
 * 更改分类
 * @param data
 */
export const updateCategory = (data: Category): Promise<Category> => {
  return request.put('/article/category', data)
}

/**
 * 删除分类
 * @param id
 */
export const deleteCategory = (id: String): Promise<any> => {
  return request.delete('/article/category/' + id)
}

// ********** 文件上传 **********
export const uploadFile = (data: FormData): Promise<String> => {
  return request.post('/article/upload', data)
}
