package repository

import (
	"errors"
	"fmt"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type TagRepository struct {
	DB *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{DB: db}
}

// CreateTag 创建新标签
func (r *TagRepository) CreateTag(tag *model.Tag) error {
	err := r.DB.Create(tag).Error
	if err != nil {
		// 检查是否是唯一索引冲突错误
		// 具体的错误码可能因数据库类型而异，这里简化处理
		return fmt.Errorf("创建标签失败: %w", err)
	}
	return nil
}

// GetTagByID 根据 ID 获取标签
func (r *TagRepository) GetTagByID(id uint) (model.Tag, error) {
	var tag model.Tag
	err := r.DB.First(&tag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Tag{}, fmt.Errorf("查询标签失败: %w", errs.ErrNotFound)
		}
		return model.Tag{}, fmt.Errorf("数据库查询标签失败: %w", err)
	}
	return tag, nil
}

// GetTagByName 根据名称获取标签
func (r *TagRepository) GetTagByName(name string) (model.Tag, error) {
	var tag model.Tag
	err := r.DB.Where("name = ?", name).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Tag{}, fmt.Errorf("查询标签失败: %w", errs.ErrNotFound)
		}
		return model.Tag{}, fmt.Errorf("数据库查询标签失败: %w", err)
	}
	return tag, nil
}

// GetTagBySlug 根据 Slug 获取标签
func (r *TagRepository) GetTagBySlug(slug string) (model.Tag, error) {
	var tag model.Tag
	err := r.DB.Where("slug = ?", slug).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Tag{}, fmt.Errorf("查询标签失败: %w", errs.ErrNotFound)
		}
		return model.Tag{}, fmt.Errorf("数据库查询标签失败: %w", err)
	}
	return tag, nil
}

// UpdateTag 更新标签
func (r *TagRepository) UpdateTag(tag *model.Tag) error {
	err := r.DB.Save(tag).Error
	if err != nil {
		return fmt.Errorf("更新标签失败: %w", err)
	}
	return nil
}

// DeleteTag 删除标签
func (r *TagRepository) DeleteTag(id uint) error {
	err := r.DB.Delete(&model.Tag{}, id).Error
	if err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}
	return nil
}

// GetAllTags 获取所有标签
func (r *TagRepository) GetAllTags() ([]model.Tag, error) {
	var tags []model.Tag
	err := r.DB.Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("获取所有标签失败: %w", err)
	}
	return tags, nil
}

// 根据文章id获取标签id
func (r *TagRepository) GetTagsByArticleID(articleID uint) ([]uint, error) {
	var article model.Article
	// 1. 首先，根据文章ID查找对应的文章实体
	if err := r.DB.First(&article, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果文章不存在，返回特定的错误
			return nil, fmt.Errorf("文章不存在: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("查询文章失败: %w", err)
	}

	var tags []model.Tag
	// 2. 使用 GORM 的 Association 方法加载文章关联的标签
	// GORM 会根据 Article 模型中 Tags 字段的 many2many 标签自动处理中间表
	if err := r.DB.Model(&article).Association("Tags").Find(&tags); err != nil {
		return nil, fmt.Errorf("获取文章关联标签失败: %w", err)
	}

	var tagIDs []uint
	// 3. 遍历获取到的标签，提取它们的ID
	for _, tag := range tags {
		tagIDs = append(tagIDs, tag.ID)
	}

	return tagIDs, nil
}

// GetTagNamesByIDs 根据标签ID列表获取标签名称列表
func (r *TagRepository) GetTagNamesByIDs(tagIDs []uint) ([]string, error) {
	var tags []model.Tag
	if len(tagIDs) == 0 {
		return []string{}, nil
	}
	err := r.DB.Where("id IN (?)", tagIDs).Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("根据ID获取标签名称失败: %w", err)
	}

	var tagNames []string
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}
	return tagNames, nil
}
