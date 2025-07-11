package repository

import (
	"context"
	"fmt"

	"dh-blog/internal/model"
	"github.com/sirupsen/logrus"
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

// GetAllTagNames 获取所有标签的名称列表
func (r *TagRepository) GetAllTagNames(ctx context.Context) ([]string, error) {
	var tagNames []string

	// 使用原始SQL查询直接获取标签名称列表
	err := r.db.WithContext(ctx).
		Model(&model.Tag{}).
		Select("name").
		Order("name").
		Pluck("name", &tagNames).Error

	if err != nil {
		logrus.Errorf("获取所有标签名称失败: %v", err)
		return nil, fmt.Errorf("获取所有标签名称失败: %w", err)
	}

	return tagNames, nil
}

// GetAllTagNamesWithCache 获取所有标签的名称列表（带缓存）
// 在实际项目中，可以添加缓存机制来优化性能
func (r *TagRepository) GetAllTagNamesWithCache(ctx context.Context) ([]string, error) {
	// TODO: 实现缓存机制
	// 这里可以添加Redis或内存缓存，定期刷新
	// 简单起见，目前直接调用数据库查询
	return r.GetAllTagNames(ctx)
}
