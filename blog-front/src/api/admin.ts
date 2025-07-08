import request from '@/api/axios'
import { Article } from '@/types/Article'
import { Category } from '@/types/Category'
import { Page, PageResult } from '@/types/Page'
import { Tag } from '@/types/Tag'
import { IpStat } from '@/types/IpStat'
import { Comment } from "@/types/Comment";
import { SystemConfig } from '@/types/SystemConfig';

/**
 * 查询文章详情
 * @param id
 * @returns
 */
export const getArticleInfo = (id: String): Promise<Article<any>> => {
  return request.get('/admin/article/' + id)
}

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
  return request.post('/admin/tag', data)
}

/**
 * 更新标签
 * @param data
 */
export const updateTag = (data: Tag): Promise<Tag> => {
  return request.put('/admin/tag', data)
}

/**
 * 删除标签
 * @param id
 */
export const deleteTag = (id: String): Promise<any> => {
  return request.delete('/admin/tag' + id)
}

// ********** 分类操作 **********

/**
 * 新增分类
 * @param data
 */
export const addCategory = (data: Category): Promise<Category> => {
  return request.post('/admin/category', data)
}

/**
 * 更改分类
 * @param data
 */
export const updateCategory = (data: Category): Promise<Category> => {
  return request.put('/admin/category', data)
}

/**
 * 删除分类
 * @param id
 */
export const deleteCategory = (id: String): Promise<any> => {
  return request.delete('/admin/category/' + id)
}

/**
 * 通过分类id查询标签列表
 * @param data 
 * @returns 
 */
export const getTagListByCategoryId = (categoryId: number): Promise<Tag[]> => {
  return request.get(`/admin/category/${categoryId}/tags`)
}

// ********** 文件上传 **********
export const uploadFile = (data: FormData): Promise<String> => {
  return request.post('/admin/upload/blog', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

/**
 * 数据总览获取访问记录
 * @param page 
 * @param pageSize 
 * @param startDate 2024-11-11
 * @param endDate 
 * @returns 
 */
export const getOverviewLog = (page: number, pageSize: number, startDate: string, endDate: string): Promise<PageResult<IpStat>> => {
  return request.get(`/admin/log/overview/visitLog?page=${page}&pageSize=${pageSize}&startDate=${startDate}&endDate=${endDate}`)
};

/**
 * 获取所有评论
 * @param pageNum 
 * @param pageSize 
 * @returns 
 */
export const getAllComment = (pageNum: number, pageSize: number): Promise<PageResult<Comment>> => {
  return request.get(`/admin/comment/${pageSize}/${pageNum}`);
}

/**
 * 编辑评论
 */
export const editComment = (comment: Comment): Promise<string> => {
  return request.put('/admin/comment', comment)
}

/**
 * 回复评论
 */
export const replyComment = (content: string, isPublic: boolean, parentId: number, articleId: number): Promise<string> => {
  return request.post('/admin/comment/reply', { content, isPublic, parentId, articleId })
}

/**
 * 删除评论
 */
export const deleteComment = (id: number): Promise<string> => {
  return request.delete(`/admin/comment/${id}`)
}

/**
 * 封禁ip
 */
export const postBanIp = (ip: string, status: number): Promise<string> => {
  return request.post(`/admin/ip/ban/${ip}/${status}`)
}


// ********** 系统配置 **********
export const getSystemConfig = (): Promise<SystemConfig> => {
  return request.get('/admin/config')
}

export const updateSystemConfig = (data: SystemConfig): Promise<any> => {
  return request.put('/admin/config', data)
}
