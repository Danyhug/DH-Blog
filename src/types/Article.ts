export interface Article<T> {
  id?: number;
  title: string;
  content: string;
  categoryId: number;
  categoryName?: string;
  createTime?: string;
  updateTime?: string;
  tags?: T[];
  views?: number;
  wordNum?: number; // TypeScript中没有byte类型，通常使用number代替
  thumbnailUrl?: string;
}