export interface Category {
  id?: number;
  name: string;
  slug: string;
  createTime?: Date;
  updateTime?: Date;
  tagIds?: number[];
}