package dto

type ArticleUpdateDTO struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	CategoryId   int      `json:"category_id"`
	WordNum      int      `json:"word_num"`
	Tags         []string `json:"tags"`
	ThumbnailUrl string   `json:"thumbnail_url"`
	IsLocked     bool     `json:"is_locked"`
	LockPassword string   `json:"lock_password"`
}
