package share

import (
	"context"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func newRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, share *Share) error {
	return r.db.WithContext(ctx).Create(share).Error
}

func (r *Repository) Update(ctx context.Context, share *Share) error {
	return r.db.WithContext(ctx).Save(share).Error
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&Share{}, id).Error
}

func (r *Repository) FindByID(ctx context.Context, id int) (*Share, error) {
	var share Share
	if err := r.db.WithContext(ctx).First(&share, id).Error; err != nil {
		return nil, err
	}
	return &share, nil
}

func (r *Repository) FindByShareID(ctx context.Context, shareID string) (*Share, error) {
	var share Share
	if err := r.db.WithContext(ctx).Where("share_id = ?", shareID).First(&share).Error; err != nil {
		return nil, err
	}
	return &share, nil
}

func (r *Repository) FindByFileKey(ctx context.Context, fileKey string) ([]*Share, error) {
	var shares []*Share
	if err := r.db.WithContext(ctx).Where("file_key = ?", fileKey).Find(&shares).Error; err != nil {
		return nil, err
	}
	return shares, nil
}

func (r *Repository) ListByPage(ctx context.Context, page, pageSize int) ([]*Share, int64, error) {
	var shares []*Share
	var total int64

	if err := r.db.WithContext(ctx).Model(&Share{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&shares).Error; err != nil {
		return nil, 0, err
	}
	return shares, total, nil
}

func (r *Repository) IncrementViewCount(ctx context.Context, shareID string) error {
	return r.db.WithContext(ctx).Model(&Share{}).
		Where("share_id = ?", shareID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

func (r *Repository) IncrementDownloadCount(ctx context.Context, shareID string) error {
	return r.db.WithContext(ctx).Model(&Share{}).
		Where("share_id = ?", shareID).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error
}

type AccessLogRepository struct {
	db *gorm.DB
}

func newAccessLogRepository(db *gorm.DB) *AccessLogRepository {
	return &AccessLogRepository{db: db}
}

func (r *AccessLogRepository) Create(ctx context.Context, log *ShareAccessLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AccessLogRepository) ListByShareID(ctx context.Context, shareID string, page, pageSize int) ([]*ShareAccessLog, int64, error) {
	var logs []*ShareAccessLog
	var total int64

	query := r.db.WithContext(ctx).Where("share_id = ?", shareID)
	if err := query.Model(&ShareAccessLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
