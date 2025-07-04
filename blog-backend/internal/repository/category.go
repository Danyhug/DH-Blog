package repository

import (
	"errors"
	"fmt"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	DB *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{DB: db}
}

// CreateCategory 创建新分类
func (r *CategoryRepository) CreateCategory(category *model.Category) error {
	err := r.DB.Create(category).Error
	if err != nil {
		// 检查是否是唯一索引冲突错误
		return fmt.Errorf("创建分类失败: %w", err)
	}
	return nil
}

// GetCategoryByID 根据 ID 获取分类
func (r *CategoryRepository) GetCategoryByID(id uint) (model.Category, error) {
	var category model.Category
	err := r.DB.First(&category, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Category{}, fmt.Errorf("查询分类失败: %w", errs.ErrNotFound)
		}
		return model.Category{}, fmt.Errorf("数据库查询分类失败: %w", err)
	}
	return category, nil
}

// GetCategoryBySlug 根据 Slug 获取分类
func (r *CategoryRepository) GetCategoryBySlug(slug string) (model.Category, error) {
	var category model.Category
	err := r.DB.Where("slug = ?", slug).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Category{}, fmt.Errorf("查询分类失败: %w", errs.ErrNotFound)
		}
		return model.Category{}, fmt.Errorf("数据库查询分类失败: %w", err)
	}
	return category, nil
}

// UpdateCategory 更新分类
func (r *CategoryRepository) UpdateCategory(category *model.Category) error {
	err := r.DB.Save(category).Error
	if err != nil {
		return fmt.Errorf("更新分类失败: %w", err)
	}
	return nil
}

// DeleteCategory 删除分类
func (r *CategoryRepository) DeleteCategory(id uint) error {
	err := r.DB.Delete(&model.Category{}, id).Error
	if err != nil {
		return fmt.Errorf("删除分类失败: %w", err)
	}
	return nil
}

// GetAllCategories 获取所有分类
func (r *CategoryRepository) GetAllCategories() ([]model.Category, error) {
	var categories []model.Category
	err := r.DB.Find(&categories).Error
	if err != nil {
		return nil, fmt.Errorf("获取所有分类失败: %w", err)
	}
	return categories, nil
}

// SaveCategoryDefaultTags 保存分类默认标签关联
func (r *CategoryRepository) SaveCategoryDefaultTags(categoryID uint, tagIDs []uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 删除旧的关联
		if err := tx.Where("category_id = ?", categoryID).Delete(&model.CategoryDefaultTags{}).Error; err != nil {
			return fmt.Errorf("删除旧的分类标签关联失败: %w", err)
		}

		// 插入新的关联
		for _, tagID := range tagIDs {
			association := model.CategoryDefaultTags{
				CategoryID: categoryID,
				TagID:      tagID,
			}
			if err := tx.Create(&association).Error; err != nil {
				return fmt.Errorf("创建新的分类标签关联失败: %w", err)
			}
		}
		return nil
	})
}

// GetCategoryDefaultTagsByID 根据分类ID获取默认标签ID列表
func (r *CategoryRepository) GetCategoryDefaultTagsByID(categoryID uint) ([]uint, error) {
	var associations []model.CategoryDefaultTags
	if err := r.DB.Where("category_id = ?", categoryID).Find(&associations).Error; err != nil {
		return nil, fmt.Errorf("查询分类默认标签失败: %w", err)
	}

	var tagIDs []uint
	for _, assoc := range associations {
		tagIDs = append(tagIDs, assoc.TagID)
	}
	return tagIDs, nil
}
