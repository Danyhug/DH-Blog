package repository

import (
	"context"
	"fmt"
	"time"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 缓存相关常量
const (
	TagsListCacheKey  = "tag:all_tags"
	TagsNamesCacheKey = "tag:all_names"
	// TagCacheExpireMedium 缓存过期时间
	TagCacheExpireMedium = time.Minute * 30 // 中期缓存，30分钟
)

// TagRepository 封装标签相关的数据库操作
type TagRepository struct {
	*GormRepository[model.Tag, int]
	db    *gorm.DB
	cache dhcache.Cache
}

// NewTagRepository 创建标签仓库
func NewTagRepository(db *gorm.DB, cache dhcache.Cache) *TagRepository {
	return &TagRepository{
		GormRepository: NewGormRepository[model.Tag, int](db),
		db:             db,
		cache:          cache,
	}
}

// GetAllTagNamesWithCache 获取所有标签名称，优先从缓存获取
func (r *TagRepository) GetAllTagNamesWithCache(ctx context.Context) ([]string, error) {
	// 尝试从缓存获取
	if cached, found := r.cache.Get(TagsNamesCacheKey); found {
		if names, ok := cached.([]string); ok {
			logrus.Debug("从缓存获取标签名称列表")
			return names, nil
		}
	}

	// 缓存未命中，从数据库获取
	var tags []model.Tag
	if err := r.db.WithContext(ctx).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}

	// 提取标签名称
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}

	// 存入缓存，设置30分钟过期
	r.cache.Set(TagsNamesCacheKey, names, TagCacheExpireMedium)
	logrus.Debug("标签名称列表已缓存")

	return names, nil
}

// GetAllTagsWithCache 获取所有标签，优先从缓存获取
func (r *TagRepository) GetAllTagsWithCache(ctx context.Context) ([]model.Tag, error) {
	// 尝试从缓存获取
	if cached, found := r.cache.Get(TagsListCacheKey); found {
		if tags, ok := cached.([]model.Tag); ok {
			logrus.Debug("从缓存获取标签列表")
			return tags, nil
		}
	}

	// 缓存未命中，从数据库获取
	var tags []model.Tag
	if err := r.db.WithContext(ctx).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}

	// 存入缓存，设置30分钟过期
	r.cache.Set(TagsListCacheKey, tags, TagCacheExpireMedium)
	logrus.Debug("标签列表已缓存")

	return tags, nil
}

// ClearTagCache 清除标签缓存
func (r *TagRepository) ClearTagCache() {
	r.cache.Delete(TagsListCacheKey)
	r.cache.Delete(TagsNamesCacheKey)
	logrus.Debug("标签缓存已清除")
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

		// 清除标签缓存
		r.ClearTagCache()
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

// 重写基础方法，添加缓存清除逻辑

// Create 创建标签，并清除缓存
func (r *TagRepository) Create(ctx context.Context, tag *model.Tag) error {
	err := r.GormRepository.Create(ctx, tag)
	if err == nil {
		r.ClearTagCache()
	}
	return err
}

// Update 更新标签，并清除缓存
func (r *TagRepository) Update(ctx context.Context, tag *model.Tag) error {
	err := r.GormRepository.Update(ctx, tag)
	if err == nil {
		r.ClearTagCache()
	}
	return err
}

// Delete 删除标签，并清除缓存
func (r *TagRepository) Delete(ctx context.Context, id int) error {
	err := r.GormRepository.Delete(ctx, id)
	if err == nil {
		r.ClearTagCache()
	}
	return err
}
