package repository

import (
	"context"
	"dh-blog/internal/model"

	"gorm.io/gorm"
)

// IFileRepository 文件存储接口
type IFileRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, file *model.File) error        // 创建文件记录
	Update(ctx context.Context, file *model.File) error        // 更新文件记录
	Delete(ctx context.Context, id int) error                  // 删除文件记录
	FindByID(ctx context.Context, id int) (*model.File, error) // 根据ID查找文件

	// 文件系统特有操作
	ListByParentID(ctx context.Context, userID uint64, parentID string) ([]*model.File, error)                 // 获取指定目录下的所有文件
	FindByPath(ctx context.Context, userID uint64, path string) (*model.File, error)                           // 根据存储路径查找文件
	FindByUserIDAndName(ctx context.Context, userID uint64, parentID string, name string) (*model.File, error) // 根据用户ID、父目录和文件名查找文件

	// 统计和批量操作
	BatchDelete(ctx context.Context, ids []int) error                                            // 批量删除文件
	CountByUserID(ctx context.Context, userID uint64) (int64, error)                             // 统计用户的文件总数
	CountByUserIDAndParentID(ctx context.Context, userID uint64, parentID string) (int64, error) // 统计用户在特定目录下的文件数量
	TruncateFiles(ctx context.Context) error                                                     // 清空文件表
}

// FileRepository 文件存储的GORM实现
type FileRepository struct {
	*GormRepository[model.File, int]
	db *gorm.DB
}

// NewFileRepository 创建文件存储实现
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - IFileRepository: 文件存储接口实现
func NewFileRepository(db *gorm.DB) IFileRepository {
	return &FileRepository{
		GormRepository: NewGormRepository[model.File, int](db),
		db:             db,
	}
}

// ListByParentID 根据父目录ID列出文件和子目录
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - parentID: 父目录ID
//
// 返回:
//   - []*model.File: 文件列表
//   - error: 错误信息
func (r *FileRepository) ListByParentID(ctx context.Context, userID uint64, parentID string) ([]*model.File, error) {
	var files []*model.File

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND parent_id = ?", userID, parentID).
		Order("is_folder DESC, name ASC"). // 文件夹优先，然后按名称排序
		Find(&files).Error

	if err != nil {
		return nil, err
	}

	return files, nil
}

// FindByPath 根据文件存储路径查找文件
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - path: 文件存储路径
//
// 返回:
//   - *model.File: 找到的文件
//   - error: 错误信息
func (r *FileRepository) FindByPath(ctx context.Context, userID uint64, path string) (*model.File, error) {
	var file model.File

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND storage_path = ?", userID, path).
		First(&file).Error

	if err != nil {
		return nil, err
	}

	return &file, nil
}

// FindByUserIDAndName 根据用户ID、父目录ID和文件名查找文件
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - parentID: 父目录ID
//   - name: 文件名
//
// 返回:
//   - *model.File: 找到的文件
//   - error: 错误信息
func (r *FileRepository) FindByUserIDAndName(ctx context.Context, userID uint64, parentID string, name string) (*model.File, error) {
	var file model.File

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND parent_id = ? AND name = ?", userID, parentID, name).
		First(&file).Error

	if err != nil {
		return nil, err
	}

	return &file, nil
}

// BatchDelete 批量删除文件
// 参数:
//   - ctx: 上下文
//   - ids: 要删除的文件ID列表
//
// 返回:
//   - error: 错误信息
func (r *FileRepository) BatchDelete(ctx context.Context, ids []int) error {
	return r.db.WithContext(ctx).
		Delete(&model.File{}, ids).Error
}

// CountByUserID 统计用户的文件总数
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//
// 返回:
//   - int64: 文件总数
//   - error: 错误信息
func (r *FileRepository) CountByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountByUserIDAndParentID 统计用户在特定目录下的文件数量
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - parentID: 父目录ID
//
// 返回:
//   - int64: 文件数量
//   - error: 错误信息
func (r *FileRepository) CountByUserIDAndParentID(ctx context.Context, userID uint64, parentID string) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("user_id = ? AND parent_id = ?", userID, parentID).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

// TruncateFiles 清空文件表
// 在更改存储路径时使用，会删除所有文件记录
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - error: 错误信息
func (r *FileRepository) TruncateFiles(ctx context.Context) error {
	// 使用原始SQL执行清空表操作，因为GORM的Delete不会清空表
	return r.db.WithContext(ctx).Exec("DELETE FROM files").Error
}
