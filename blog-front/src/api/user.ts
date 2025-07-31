import { Article } from "@/types/Article";
import { Category } from "@/types/Category";
import { Page, PageResult } from "@/types/Page";
import { Tag } from "@/types/Tag";
import request from "@/api/axios";
import { UserLogin } from "@/types/User";
import { OverView } from "@/types/DashBoard";
import { Comment } from "@/types/Comment";

/**
 * 查询文章详情
 * @param id
 * @returns
 */
export const getArticleInfo = (id: String): Promise<Article<any>> => {
  return request.get("/article/" + id);
};

/**
 * 根据id查询文章标题
 */
export const getArticleTitleById = (id: number): Promise<string> => {
  return request.get("/article/title/" + id);
}

/**
 * 查询文章列表（分页查询）
 * @param data
 * @returns
 */
export const getArticleList = (
  data: Page
): Promise<PageResult<Article<any>>> => {
  return request.post("/article/list", data);
};

/**
 * 获取所有文章分类
 * @returns
 */
export const getArticleCategoryList = (): Promise<Category[]> => {
  return request.get("/article/category");
};

/**
 * 获取所有文章标签
 * @returns
 */
export const getArticleTagList = (): Promise<Tag[]> => {
  return request.get("/article/tag");
};

/**
 * 用户登录
 */
export const userLogin = (user: UserLogin): Promise<string> => {
  return request.post("/user/login", user);
};

/**
 * 用户校验
 */
export const userCheck = () => {
  return request.post("/user/check");
};

/**
 *  解密文章
 */
export const unLockArticle = (
  id: number,
  password: string
): Promise<Article<any>> => {
  return request.get(`/article/unlock/${id}/${password}`);
};

/**
 * 心跳
 */
export const heartBeat = () => {
  return request.get("/user/heart");
};

/**
 * 数据总览
 */
export const getOverview = (): Promise<OverView> => {
  return request.get("/article/overview");
};

/**
 * 获取所有标签和分类
 * @returns 返回包含所有标签和分类的数组，格式为{name, url, type, count}
 */
export const getAllTaxonomies = (): Promise<Array<{name: string, url: string, type: string, count: number}>> => {
  return request.get("/article/taxonomies");
};

/**
 * 获取指定标签或分类的关联文章
 * @param name 标签或分类名称
 * @param type 类型，可以是'tag'或'category'
 * @returns 返回关联的文章列表
 */
export const getArticlesByTaxonomy = (name: string, type: string): Promise<Array<{
  id: number,
  title: string,
  views: number,
  wordNum: number,
  createTime: string,
  updateTime: string
}>> => {
  return request.get(`/article/taxonomy/articles?name=${encodeURIComponent(name)}&type=${type}`);
};

/**
 * 用户评论
 */
export const addComment = (comment: Comment) => {
  return request.post("/comment", comment);
};

/**
 * 获取评论列表
 */
export const getCommentList = (
  articleId: number
): Promise<PageResult<Comment>> => {
  return request.get("/comment/" + articleId);
};