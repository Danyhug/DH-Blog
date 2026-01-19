import request from '@/api/axios'
import { SERVER_URL } from '@/types/Constant'
import { Article } from '@/types/Article'
import { Category } from '@/types/Category'
import { Page, PageResult } from '@/types/Page'
import { Tag } from '@/types/Tag'
import { IpStat } from '@/types/IpStat'
import { Comment } from "@/types/Comment";
import { SystemConfig, BlogConfig, EmailConfig, AIConfig, StorageConfig } from '@/types/SystemConfig';

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
 * 获取访问统计数据
 * @returns 访问统计数据
 */
export const getVisitStatistics = (): Promise<{todayVisits: number, weekVisits: number, monthVisits: number, totalVisits: number}> => {
  return request.get('/admin/log/stats/visits');
};

/**
 * 获取月度访问统计数据
 * @param year 年份，可选，默认为当前年份
 * @returns 月度访问统计数据
 */
export const getMonthlyVisitStats = (year?: number): Promise<{month: number, visit_count: number}[]> => {
  const url = year ? `/admin/log/stats/monthly?year=${year}` : '/admin/log/stats/monthly';
  return request.get(url);
};

/**
 * 获取每日访问统计数据
 * @param days 天数，可选，默认为30天
 * @returns 每日访问统计数据
 */
export const getDailyVisitStats = (days?: number): Promise<{date: string, visit_count: number}[]> => {
  const url = days ? `/admin/log/stats/daily-chart?days=${days}` : '/admin/log/stats/daily-chart';
  return request.get(url);
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
// 获取所有配置
export const getSystemConfig = (): Promise<SystemConfig> => {
  return request.get('/admin/config')
}

// 更新所有配置
export const updateSystemConfig = (data: SystemConfig): Promise<any> => {
  return request.put('/admin/config', data)
}

// ********** 博客基本配置 **********
export const getBlogConfig = (): Promise<BlogConfig> => {
  return request.get('/admin/config/blog')
}

export const updateBlogConfig = (data: BlogConfig): Promise<any> => {
  return request.put('/admin/config/blog', data)
}

// ********** 邮件配置 **********
export const getEmailConfig = (): Promise<EmailConfig> => {
  return request.get('/admin/config/email')
}

export const updateEmailConfig = (data: EmailConfig): Promise<any> => {
  return request.put('/admin/config/email', data)
}

// ********** AI配置 **********
export const getAIConfig = (): Promise<AIConfig> => {
  return request.get('/admin/config/ai')
}

export const updateAIConfig = (data: AIConfig): Promise<any> => {
  return request.put('/admin/config/ai', data)
}

/**
 * 获取预定义的AI提示词标签
 */
export const getAIPromptTags = (): Promise<{ label: string, prompt: string }[]> => {
  return request.get('/admin/config/ai/prompts');
}

// ********** 存储配置 **********
export const getStorageConfig = (): Promise<StorageConfig> => {
  return request.get('/admin/config/storage')
}

export const updateStorageConfig = (data: StorageConfig): Promise<any> => {
  return request.put('/admin/config/storage', data)
}

// 备份目录信息
export interface BackupDirInfo {
  name: string
  is_protected: boolean
}

// 获取可备份的目录列表
export const getBackupDirs = (): Promise<BackupDirInfo[]> => {
  return request.get('/admin/config/backup/dirs')
}

// 获取数据备份下载URL
// mode: 'full' - 备份数据库和整个WebDAV目录
// dirs: 逗号分隔的目录名列表 - 备份数据库和指定目录
// 默认（无参数）：备份数据库和固定目录
export const getBackupUrl = (options?: { mode?: 'full', dirs?: string[] }): string => {
  const token = localStorage.getItem("token") || "";
  const tokenParam = token.startsWith("Bearer ") ? token.substring(7) : token;

  let url = `${SERVER_URL}/admin/config/backup?token=${tokenParam}`;
  if (options?.mode === 'full') {
    url += '&mode=full';
  } else if (options?.dirs && options.dirs.length > 0) {
    url += `&dirs=${encodeURIComponent(options.dirs.join(','))}`;
  }
  return url;
}

// ********** 文件存储路径配置（兼容旧版） **********
export const getStoragePath = (): Promise<{path: string}> => {
  return request.get('/admin/config/storage-path')
}

export const updateStoragePath = (path: string): Promise<any> => {
  return request.put('/admin/config/storage-path', {path})
}

// ********** 系统配置项管理 **********
// 获取所有系统配置项
export const getSystemSettings = (): Promise<any[]> => {
  return request.get('/admin/system-setting/list');
}
// 新增系统配置项
export const addSystemSetting = (data: { settingKey: string, settingValue: string, configType?: string }): Promise<any> => {
  return request.post('/admin/system-setting', data);
}
// 更新系统配置项
export const updateSystemSetting = (data: { id?: number, settingKey: string, settingValue: string, configType?: string }): Promise<any> => {
  return request.put('/admin/system-setting', data);
}
// 删除系统配置项
export const deleteSystemSetting = (id: number): Promise<any> => {
  return request.delete(`/admin/system-setting/${id}`);
}

/**
 * 使用AI为文章生成标签
 * @param articleId 文章ID
 * @returns 生成的标签列表
 */
export const generateAITags = (articleId: number): Promise<Tag[]> => {
  return request.post(`/admin/article/${articleId}/generate-tags`);
}
