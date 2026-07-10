package article

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	TagsListCacheKey     = "tag:all_tags"
	TagsNamesCacheKey    = "tag:all_names"
	TagCacheExpireMedium = 30 * time.Minute
)

type TagRepository struct {
	*gormRepository[Tag, int]
	db    *gorm.DB
	cache Cache
}

func NewTagRepository(db *gorm.DB, cache Cache) *TagRepository {
	return &TagRepository{gormRepository: newGormRepository[Tag, int](db), db: db, cache: cache}
}

func (r *TagRepository) GetAllTagNamesWithCache(ctx context.Context) ([]string, error) {
	if cached, found := r.cache.Get(TagsNamesCacheKey); found {
		if names, ok := cached.([]string); ok {
			logrus.Debug("从缓存获取标签名称列表")
			return names, nil
		}
		logrus.Warn("标签名称列表缓存类型转换失败，将从数据库重新获取")
	}

	var tags []Tag
	if err := r.db.WithContext(ctx).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}
	names := make([]string, len(tags))
	for i := range tags {
		names[i] = tags[i].Name
	}
	_ = r.cache.Set(TagsNamesCacheKey, names, TagCacheExpireMedium)
	logrus.Debug("标签名称列表已缓存")
	return names, nil
}

func (r *TagRepository) GetAllTagsWithCache(ctx context.Context) ([]Tag, error) {
	if cached, found := r.cache.Get(TagsListCacheKey); found {
		if tags, ok := cached.([]Tag); ok {
			logrus.Debug("从缓存获取标签列表")
			return tags, nil
		}
		logrus.Warn("标签列表缓存类型转换失败，将从数据库重新获取")
	}

	var tags []Tag
	if err := r.db.WithContext(ctx).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}
	_ = r.cache.Set(TagsListCacheKey, tags, TagCacheExpireMedium)
	logrus.Debug("标签列表已缓存")
	return tags, nil
}

func (r *TagRepository) ClearTagCache() {
	if deleted := r.cache.Delete(TagsListCacheKey); !deleted {
		logrus.Warn("清除标签列表缓存失败: 缓存中未找到")
	}
	if deleted := r.cache.Delete(TagsNamesCacheKey); !deleted {
		logrus.Warn("清除标签名称列表缓存失败: 缓存中未找到")
	} else {
		logrus.Debug("标签缓存已清除")
	}
}

func (r *TagRepository) FindOrCreateByNames(tx *gorm.DB, names []string) ([]*Tag, error) {
	if len(names) == 0 {
		return []*Tag{}, nil
	}

	var existingTags []*Tag
	if err := tx.Where("name IN ?", names).Find(&existingTags).Error; err != nil {
		return nil, fmt.Errorf("查找现有标签失败: %w", err)
	}
	existingByName := make(map[string]*Tag, len(existingTags))
	for _, tag := range existingTags {
		existingByName[tag.Name] = tag
	}

	var newTags []*Tag
	for _, name := range names {
		if _, exists := existingByName[name]; !exists {
			newTags = append(newTags, &Tag{Name: name})
		}
	}
	if len(newTags) > 0 {
		if err := tx.Create(&newTags).Error; err != nil {
			return nil, fmt.Errorf("创建新标签失败: %w", err)
		}
		existingTags = append(existingTags, newTags...)
		r.ClearTagCache()
	}

	resultByName := make(map[string]*Tag, len(existingTags))
	for _, tag := range existingTags {
		resultByName[tag.Name] = tag
	}
	result := make([]*Tag, 0, len(names))
	for _, name := range names {
		if tag, exists := resultByName[name]; exists {
			result = append(result, tag)
		}
	}
	return result, nil
}

func (r *TagRepository) Create(ctx context.Context, tag *Tag) error {
	err := r.gormRepository.Create(ctx, tag)
	if err == nil {
		r.ClearTagCache()
	}
	return err
}

func (r *TagRepository) Update(ctx context.Context, tag *Tag) error {
	err := r.gormRepository.Update(ctx, tag)
	if err == nil {
		r.ClearTagCache()
	}
	return err
}

func (r *TagRepository) Delete(ctx context.Context, id int) error {
	err := r.gormRepository.Delete(ctx, id)
	if err == nil {
		r.ClearTagCache()
	}
	return err
}
