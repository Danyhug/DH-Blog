package article

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type gormRepository[T any, K any] struct {
	db *gorm.DB
}

func newGormRepository[T any, K any](db *gorm.DB) *gormRepository[T, K] {
	return &gormRepository[T, K]{db: db}
}

func (r *gormRepository[T, K]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *gormRepository[T, K]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *gormRepository[T, K]) Delete(ctx context.Context, id K) error {
	var entity T
	return r.db.WithContext(ctx).Delete(&entity, id).Error
}

func (r *gormRepository[T, K]) FindByID(ctx context.Context, id K) (*T, error) {
	var entity T
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *gormRepository[T, K]) FindByIDWithPreload(ctx context.Context, id K, preloads ...string) (*T, error) {
	var entity T
	db := r.db.WithContext(ctx)
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	if err := db.First(&entity, id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *gormRepository[T, K]) FindAll(ctx context.Context) ([]T, error) {
	var entities []T
	if err := r.db.WithContext(ctx).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *gormRepository[T, K]) FindPage(ctx context.Context, page, pageSize int) ([]T, int64, error) {
	var (
		entities []T
		total    int64
	)
	if err := r.db.WithContext(ctx).Model(new(T)).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(pageSize).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询分页数据失败: %w", err)
	}
	return entities, total, nil
}

func (r *gormRepository[T, K]) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(new(T)).Count(&count).Error
	return count, err
}
