import request from '@/api/axios'
import { Article } from '@/types/Article'
import { Category } from '@/types/Category'
import { Page, PageResult } from '@/types/Page'
import { Tag } from '@/types/Tag'
/**
 * 新增文章
 * @param data 
 * @returns 
 */
export const addArticle = (data: Article<any>): Promise<Article<any>> => {
  return request.post('/admin/article', data)
}

/**
 * 查询文章列表（分页查询）
 * @param data
 * @returns
 */
export const getArticleList = (data: Page): Promise<PageResult<Article<any>>> => {
  return request.post('/admin/article/list', data)
}

/**
 * 更新文章
 * @param data 
 * @returns 
 */
export const updateArticle = (data: Article<any>): Promise<Article<any>> => {
  return request.put('/admin/article', data)
}

// ********** 标签操作 **********

/**
 * 新增标签
 * @param data
 */
export const addTag = (data: Tag): Promise<Tag> => {
  return request.post('/admin/article/tag', data)
}

/**
 * 更新标签
 * @param data
 */
export const updateTag = (data: Tag): Promise<Tag> => {
  return request.put('/admin/article/tag', data)
}

/**
 * 删除标签
 * @param id
 */
export const deleteTag = (id: String): Promise<any> => {
  return request.delete('/admin/article/tag/' + id)
}

// ********** 分类操作 **********

/**
 * 新增分类
 * @param data
 */
export const addCategory = (data: Category): Promise<Category> => {
  return request.post('/admin/article/category', data)
}

/**
 * 更改分类
 * @param data
 */
export const updateCategory = (data: Category): Promise<Category> => {
  return request.put('/admin/article/category', data)
}

/**
 * 删除分类
 * @param id
 */
export const deleteCategory = (id: String): Promise<any> => {
  return request.delete('/admin/article/category/' + id)
}

// ********** 文件上传 **********
export const uploadFile = (data: FormData): Promise<String> => {
  return request.post('/admin/upload', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}
