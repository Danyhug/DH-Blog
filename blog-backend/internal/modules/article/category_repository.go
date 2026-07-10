package article

import "gorm.io/gorm"

type CategoryRepository struct {
	*gormRepository[Category, int]
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{gormRepository: newGormRepository[Category, int](db)}
}

func (r *CategoryRepository) GetCategoryDefaultTags(categoryID int) ([]Tag, error) {
	return r.getCategoryDefaultTags(r.db, categoryID)
}

func (r *CategoryRepository) getCategoryDefaultTags(db *gorm.DB, categoryID int) ([]Tag, error) {
	var tags []Tag
	err := db.Table("tags").
		Joins("JOIN tag_relations ON tags.id = tag_relations.tag_id").
		Where("tag_relations.related_id = ? AND tag_relations.relation_type = ?", categoryID, "category").
		Find(&tags).Error
	return tags, err
}

func (r *CategoryRepository) SaveCategoryDefaultTags(categoryID int, tagIDs []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("related_id = ? AND relation_type = ?", categoryID, "category").Delete(&TagRelation{}).Error; err != nil {
			return err
		}
		if len(tagIDs) == 0 {
			return nil
		}
		relations := make([]TagRelation, 0, len(tagIDs))
		for _, tagID := range tagIDs {
			relations = append(relations, TagRelation{TagID: tagID, RelatedID: categoryID, RelationType: "category"})
		}
		return tx.Create(&relations).Error
	})
}
