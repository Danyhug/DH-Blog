package model

// PageRequest 分页请求参数
type PageRequest struct {
	PageNum  int `json:"pageNum" form:"pageNum" binding:"min=0"`   // 当前页码
	PageSize int `json:"pageSize" form:"pageSize" binding:"min=0"` // 每页大小
}

// PageResult 分页结果
type PageResult[T any] struct {
	Total    int64 `json:"total"`    // 总记录数
	Page     int   `json:"page"`     // 当前页码
	PageSize int   `json:"pageSize"` // 每页大小
	List     []T   `json:"list"`     // 数据列表
}

// HasMore 是否还有更多数据
func (p *PageResult[T]) HasMore() bool {
	return p.Total > int64(p.Page*p.PageSize)
}

// TotalPages 总页数
func (p *PageResult[T]) TotalPages() int {
	if p.Total == 0 {
		return 0
	}

	totalPages := int(p.Total) / p.PageSize
	if int(p.Total)%p.PageSize > 0 {
		totalPages++
	}
	return totalPages
}
