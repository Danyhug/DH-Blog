package repository

import (
	"context"
	"fmt"
	"gorm.io/gorm"
)

// BaseRepository defines the basic repository operations
type BaseRepository[T any, K any] interface {
	FindByID(ctx context.Context, id K) (*T, error)
	FindAll(ctx context.Context) ([]T, error)
	Save(ctx context.Context, model *T) error
	Update(ctx context.Context, model *T) error
	Delete(ctx context.Context, id K) error
	FindPage(ctx context.Context, page, pageSize int) ([]T, int64, error)
	FindByIDWithPreload(ctx context.Context, id K, preloads ...string) (*T, error)
}

// GormRepository is a generic repository implementation using GORM
type GormRepository[T any, K interface{}] struct {
	db *gorm.DB
}

// NewGormRepository creates a new instance of GormRepository
func NewGormRepository[T any, K interface{}](db *gorm.DB) *GormRepository[T, K] {
	return &GormRepository[T, K]{db: db}
}

// Create creates a new record in the database
func (r *GormRepository[T, K]) Create(ctx context.Context, model *T) error {
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing record in the database
func (r *GormRepository[T, K]) Update(ctx context.Context, model *T) error {
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a record from the database by its ID
func (r *GormRepository[T, K]) Delete(ctx context.Context, id K) error {
	var model T
	return r.db.WithContext(ctx).Delete(&model, id).Error
}

// FindByID finds a record by its ID
func (r *GormRepository[T, K]) FindByID(ctx context.Context, id K) (*T, error) {
	var model T
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// FindByIDWithPreload finds a record by its ID with preloaded associations
func (r *GormRepository[T, K]) FindByIDWithPreload(ctx context.Context, id K, preloads ...string) (*T, error) {
	var model T
	db := r.db.WithContext(ctx)
	
	// 添加所有的预加载关联
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	
	err := db.First(&model, id).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// FindAll finds all records
func (r *GormRepository[T, K]) FindAll(ctx context.Context) ([]T, error) {
	var models []T
	err := r.db.WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}

// FindPage finds a page of records
func (r *GormRepository[T, K]) FindPage(ctx context.Context, page, pageSize int) ([]T, int64, error) {
	var (
		entities []T
		total    int64
	)

	offset := (page - 1) * pageSize

	// 查询总数
	if err := r.db.WithContext(ctx).Model(new(T)).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 查询分页数据
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询分页数据失败: %w", err)
	}

	return entities, total, nil
}

// Count counts the number of records
func (r *GormRepository[T, K]) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(new(T)).Count(&count).Error
	return count, err
} 