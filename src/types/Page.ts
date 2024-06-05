export interface Page {
  pageSize: Number
  pageNum: Number
  category?: String
}

export interface PageResult<T> {
  total: Number
  list: T[]
}