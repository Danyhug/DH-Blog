package repository

import (
	"fmt"

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

// FindOrCreateByNames finds or creates tags by their names
func (r *TagRepository) FindOrCreateByNames(tx *gorm.DB, names []string) ([]*model.Tag, error) {
	if len(names) == 0 {
		return []*model.Tag{}, nil
	}

	// 先一次性查找所有已存在的标签
	var existingTags []*model.Tag
	if err := tx.Where("name IN ?", names).Find(&existingTags).Error; err != nil {
		return nil, fmt.Errorf("查找现有标签失败: %w", err)
	}

	// 创建一个map来快速查找已存在的标签
	existingTagMap := make(map[string]*model.Tag)
	for _, tag := range existingTags {
		existingTagMap[tag.Name] = tag
	}

	// 收集需要创建的新标签
	var newTagNames []string
	for _, name := range names {
		if _, exists := existingTagMap[name]; !exists {
			newTagNames = append(newTagNames, name)
		}
	}

	// 创建新标签（如果有的话）
	if len(newTagNames) > 0 {
		var newTags []*model.Tag
		for _, name := range newTagNames {
			// 不设置ID字段，让数据库自动增长
			newTags = append(newTags, &model.Tag{Name: name})
		}

		// 批量创建新标签
		if err := tx.Create(&newTags).Error; err != nil {
			return nil, fmt.Errorf("创建新标签失败: %w", err)
		}

		// 将新创建的标签添加到结果中
		existingTags = append(existingTags, newTags...)
	}

	// 按照原始顺序排列结果
	var result []*model.Tag
	resultMap := make(map[string]*model.Tag)
	for _, tag := range existingTags {
		resultMap[tag.Name] = tag
	}

	for _, name := range names {
		if tag, exists := resultMap[name]; exists {
			result = append(result, tag)
		}
	}

	return result, nil
}
