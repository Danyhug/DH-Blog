package dto

import (
	"time"
)

type ArticleDetailDTO struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    Category  `json:"category"`
	PublishDate time.Time `json:"publish_date"`
	UpdateTime  time.Time `json:"update_time"`
	Views       int       `json:"views"`
	WordNum     byte      `json:"word_num"`
}
