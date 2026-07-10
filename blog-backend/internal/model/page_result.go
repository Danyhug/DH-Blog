package model

// PageRequest 分页请求参数
type PageRequest struct {
	PageNum  int `json:"pageNum" form:"pageNum" binding:"min=0"`   // 当前页码
	PageSize int `json:"pageSize" form:"pageSize" binding:"min=0"` // 每页大小
}
