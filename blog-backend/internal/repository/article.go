package repository

import (
	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type ArticleRepository struct {
	DB *gorm.DB
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{DB: db}
}

// GetArticleById 根据id获取文章信息
func (r *ArticleRepository) GetArticleById(id int) (data model.Article, err error) {
	tx := r.DB.Where(&model.Article{ID: id}).Find(&data)
	return data, tx.Error
}
