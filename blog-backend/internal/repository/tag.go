package repository

import (
	"errors"

	"dh-blog/internal/model"
	"gorm.io/gorm"
)

// TagRepository 封装标签相关的数据库操作
type TagRepository struct {
	*GormRepository[model.Tag, int]
	db *gorm.DB
}

// NewTagRepository 创建标签仓库
func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{
		GormRepository: NewGormRepository[model.Tag, int](db),
		db:             db,
	}
}

// FindOrCreateBySlugs finds or creates tags by their slugs
func (r *TagRepository) FindOrCreateBySlugs(tx *gorm.DB, slugs []string) ([]*model.Tag, error) {
	var tags []*model.Tag
	for _, slug := range slugs {
		var tag model.Tag
		err := tx.Where("slug = ?", slug).First(&tag).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Tag not found, create it
				newTag := &model.Tag{Slug: slug, Name: slug} // You might want to derive name differently
				if err := tx.Create(newTag).Error; err != nil {
					return nil, err
				}
				tags = append(tags, newTag)
			} else {
				return nil, err
			}
		} else {
			tags = append(tags, &tag)
		}
	}
	return tags, nil
}
