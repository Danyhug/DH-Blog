export interface Article<T> {
  id?: number;
  title: string;
  content: string;
  categoryId: number;
  publishDate?: Date;
  updateDate?: Date;
  tags?: T[];
  views?: number;
  wordNum?: number; // TypeScript中没有byte类型，通常使用number代替
}