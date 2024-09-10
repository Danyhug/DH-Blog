export interface Page {
  pageSize: number
  pageNum: number
  total: number
  categoryId?: string
}

export interface PageResult<T> {
  total: number
  list: T[]
}