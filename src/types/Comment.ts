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
