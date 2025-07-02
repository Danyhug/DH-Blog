package dto

type ArticlePageDTO struct {
	PageNum    int `json:"page_num"`
	PageSize   int `json:"page_size"`
	CategoryId int `json:"category_id"`
}
