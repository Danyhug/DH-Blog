package model

// PageResult 通用分页结果结构体
type PageResult[T any] struct {
	List  []T   `json:"list"`  // 数据列表
	Total int64 `json:"total"` // 总记录数
	Page  int   `json:"page"`  // 当前页码
	Size  int   `json:"size"`  // 每页大小
}
