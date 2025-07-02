package dto

type ArticleInsertDTO struct {
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags"`
	WordNum      int      `json:"word_num"`
	CategoryId   int      `json:"category_id"`
	ThumbnailUrl string   `json:"thumbnail_url"`
	IsLocked     bool     `json:"is_locked"`
	LockPassword string   `json:"lock_password"`
}
