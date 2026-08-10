export interface Article<T> {
  id?: number;
  title: string;
  content: string;
  summary?: string; // 首页展示用的摘要，列表接口只返回摘要不返回正文
  categoryId: number;
  categoryName?: string;
  createTime?: string;
  updateTime?: string;
  tags?: T[];
  views?: number;
  wordNum?: number; // TypeScript中没有byte类型，通常使用number代替
  thumbnailUrl?: string;
  isLocked?: boolean;
  canAccess?: boolean;
  lockPassword?: string;
}
