package repository

import (
	"context"

	"dh-blog/internal/model"

	"gorm.io/gorm"
)

// IShareRepository 分享记录仓库接口
type IShareRepository interface {
	Create(ctx context.Context, share *model.Share) error
	Update(ctx context.Context, share *model.Share) error
	Delete(ctx context.Context, id int) error
	FindByID(ctx context.Context, id int) (*model.Share, error)
	FindByShareID(ctx context.Context, shareID string) (*model.Share, error)
	FindByFileKey(ctx context.Context, fileKey string) ([]*model.Share, error)
	ListByPage(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error)
	IncrementViewCount(ctx context.Context, shareID string) error
	IncrementDownloadCount(ctx context.Context, shareID string) error
}

// ShareRepository 分享记录仓库实现
type ShareRepository struct {
	*GormRepository[model.Share, int]
	db *gorm.DB
}

// NewShareRepository 创建分享记录仓库
func NewShareRepository(db *gorm.DB) IShareRepository {
	return &ShareRepository{
		GormRepository: NewGormRepository[model.Share, int](db),
		db:             db,
	}
}

// FindByShareID 根据分享ID查找分享记录
func (r *ShareRepository) FindByShareID(ctx context.Context, shareID string) (*model.Share, error) {
	var share model.Share
	err := r.db.WithContext(ctx).Where("share_id = ?", shareID).First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}

// FindByFileKey 根据文件标识查找所有分享记录
func (r *ShareRepository) FindByFileKey(ctx context.Context, fileKey string) ([]*model.Share, error) {
	var shares []*model.Share
	err := r.db.WithContext(ctx).Where("file_key = ?", fileKey).Find(&shares).Error
	if err != nil {
		return nil, err
	}
	return shares, nil
}

// ListByPage 分页获取分享记录
func (r *ShareRepository) ListByPage(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error) {
	var shares []*model.Share
	var total int64

	offset := (page - 1) * pageSize

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&model.Share{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询分页数据
	if err := r.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(pageSize).Find(&shares).Error; err != nil {
		return nil, 0, err
	}

	return shares, total, nil
}

// IncrementViewCount 增加查看次数
func (r *ShareRepository) IncrementViewCount(ctx context.Context, shareID string) error {
	return r.db.WithContext(ctx).Model(&model.Share{}).
		Where("share_id = ?", shareID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementDownloadCount 增加下载次数
func (r *ShareRepository) IncrementDownloadCount(ctx context.Context, shareID string) error {
	return r.db.WithContext(ctx).Model(&model.Share{}).
		Where("share_id = ?", shareID).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error
}

// IShareAccessLogRepository 分享访问日志仓库接口
type IShareAccessLogRepository interface {
	Create(ctx context.Context, log *model.ShareAccessLog) error
	ListByShareID(ctx context.Context, shareID string, page, pageSize int) ([]*model.ShareAccessLog, int64, error)
}

// ShareAccessLogRepository 分享访问日志仓库实现
type ShareAccessLogRepository struct {
	db *gorm.DB
}

// NewShareAccessLogRepository 创建分享访问日志仓库
func NewShareAccessLogRepository(db *gorm.DB) IShareAccessLogRepository {
	return &ShareAccessLogRepository{db: db}
}

// Create 创建访问日志
func (r *ShareAccessLogRepository) Create(ctx context.Context, log *model.ShareAccessLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// ListByShareID 根据分享ID分页获取访问日志
func (r *ShareAccessLogRepository) ListByShareID(ctx context.Context, shareID string, page, pageSize int) ([]*model.ShareAccessLog, int64, error) {
	var logs []*model.ShareAccessLog
	var total int64

	offset := (page - 1) * pageSize

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&model.ShareAccessLog{}).Where("share_id = ?", shareID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询分页数据
	if err := r.db.WithContext(ctx).Where("share_id = ?", shareID).Order("id DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
