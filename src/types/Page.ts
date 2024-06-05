export interface Page {
  pageSize: Number
  pageNum: Number
  categoryId?: String
}

export interface PageResult<T> {
  total: Number
  list: T[]
}