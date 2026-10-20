export interface Comment {
  id?: number;
  articleId: number;
  author: string;
  email: string;
  content: string;
  isPublic: boolean;
  createTime?: Date | String;
  parentId: number | null;
  ua?: string;
  isAdmin?: boolean;
  children?: Comment[]; // 子评论
}

export interface ArticleCommentGroup {
  articleId: number;
  articleTitle: string;
  commentCount: number;
  latestCommentTime: string;
  children: Comment[];
}

/**
 * 文章详情页的评论分页结果。
 *
 * 分页切的是顶层评论，所以这里有两个总数：
 * - total：这篇文章的全部公开评论数（含回复），用来显示「N 条评论」；
 * - rootTotal：顶层评论数，用来算总页数。
 */
export interface CommentPageResult {
  total: number;
  rootTotal: number;
  list: Comment[];
}
