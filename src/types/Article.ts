export interface Article {
  id?: number;
  title: string;
  content: string;
  categoryId: number;
  publishDate?: Date;
  updateDate?: Date;
  tags?: string[];
  views?: number;
  wordNum?: number; // TypeScript中没有byte类型，通常使用number代替
}