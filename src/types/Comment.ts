export interface Comment {
  id?: number;
  articleId: number;
  author: string;
  email: string;
  content: string;
  public: boolean;
  createTime?: Date;
  parentId: number | null;
  ua?: string;
  admin?: boolean;
}
