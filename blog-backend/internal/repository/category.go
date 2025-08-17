package repository

import (
	"dh-blog/internal/model"

	"gorm.io/gorm"
)

// CategoryRepository 定义分类仓库
type CategoryRepository struct {
	*GormRepository[model.Category, int]
}

// NewCategoryRepository 创建新的分类仓库
func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{
		GormRepository: NewGormRepository[model.Category, int](db),
	}
}

// GetCategoryDefaultTags 获取分类的默认标签
func (r *CategoryRepository) GetCategoryDefaultTags(categoryID int) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Table("tags").
		Joins("JOIN tag_relations ON tags.id = tag_relations.tag_id").
		Where("tag_relations.related_id = ? AND tag_relations.relation_type = ?", categoryID, "category").
		Find(&tags).Error
	return tags, err
}

// SaveCategoryDefaultTags 保存分类的默认标签
func (r *CategoryRepository) SaveCategoryDefaultTags(categoryID int, tagIDs []int) error {
	// 使用事务确保操作的原子性
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先删除旧的关联
		if err := tx.Where("related_id = ? AND relation_type = ?", categoryID, "category").Delete(&model.TagRelation{}).Error; err != nil {
			return err
		}

		// 如果没有新的标签要关联，直接返回
		if len(tagIDs) == 0 {
			return nil
		}

		// 创建新的关联
		var relations []model.TagRelation
		for _, tagID := range tagIDs {
			relations = append(relations, model.TagRelation{
				TagID:        tagID,
				RelatedID:    categoryID,
				RelationType: "category",
			})
		}

		return tx.Create(&relations).Error
	})
}
