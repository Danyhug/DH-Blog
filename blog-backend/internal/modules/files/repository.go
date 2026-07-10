package files

import (
	"context"

	"gorm.io/gorm"
)

// fileRepository 是文件模块内部的持久化契约。
type fileRepository interface {
	Create(ctx context.Context, file *File) error        // 创建文件记录
	Update(ctx context.Context, file *File) error        // 更新文件记录
	Delete(ctx context.Context, id int) error            // 删除文件记录
	FindByID(ctx context.Context, id int) (*File, error) // 根据ID查找文件

	// ListByParentID 文件系统特有操作
	ListByParentID(ctx context.Context, userID uint64, parentID string) ([]*File, error)                 // 获取指定目录下的所有文件
	FindByPath(ctx context.Context, userID uint64, path string) (*File, error)                           // 根据存储路径查找文件
	FindByUserIDAndName(ctx context.Context, userID uint64, parentID string, name string) (*File, error) // 根据用户ID、父目录和文件名查找文件

	// BatchDelete 统计和批量操作
	BatchDelete(ctx context.Context, ids []int) error                                            // 批量删除文件
	CountByUserID(ctx context.Context, userID uint64) (int64, error)                             // 统计用户的文件总数
	CountByUserIDAndParentID(ctx context.Context, userID uint64, parentID string) (int64, error) // 统计用户在特定目录下的文件数量
	TruncateFiles(ctx context.Context) error                                                     // 清空文件表
	Snapshot(ctx context.Context) ([]File, error)                                                // 保存完整索引快照
	RestoreSnapshot(ctx context.Context, files []File) error                                     // 原样恢复索引和 ID
}

// Repository 是文件模块的 GORM 持久化实现。
type Repository struct {
	db *gorm.DB
}

// newRepository 创建文件存储实现
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - fileRepository: 文件存储接口实现
func newRepository(db *gorm.DB) fileRepository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, file *File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *Repository) Update(ctx context.Context, file *File) error {
	return r.db.WithContext(ctx).Save(file).Error
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&File{}, id).Error
}

func (r *Repository) FindByID(ctx context.Context, id int) (*File, error) {
	var file File
	if err := r.db.WithContext(ctx).First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

// ListByParentID 根据父目录ID列出文件和子目录
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - parentID: 父目录ID
//
// 返回:
//   - []*File: 文件列表
//   - error: 错误信息
func (r *Repository) ListByParentID(ctx context.Context, userID uint64, parentID string) ([]*File, error) {
	var files []*File

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
//   - *File: 找到的文件
//   - error: 错误信息
func (r *Repository) FindByPath(ctx context.Context, userID uint64, path string) (*File, error) {
	var file File

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
//   - *File: 找到的文件
//   - error: 错误信息
func (r *Repository) FindByUserIDAndName(ctx context.Context, userID uint64, parentID string, name string) (*File, error) {
	var file File

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
func (r *Repository) BatchDelete(ctx context.Context, ids []int) error {
	return r.db.WithContext(ctx).
		Delete(&File{}, ids).Error
}

// CountByUserID 统计用户的文件总数
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//
// 返回:
//   - int64: 文件总数
//   - error: 错误信息
func (r *Repository) CountByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&File{}).
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
func (r *Repository) CountByUserIDAndParentID(ctx context.Context, userID uint64, parentID string) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&File{}).
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
func (r *Repository) TruncateFiles(ctx context.Context) error {
	// 使用原始SQL执行清空表操作，因为GORM的Delete不会清空表
	return r.db.WithContext(ctx).Exec("DELETE FROM files").Error
}

func (r *Repository) Snapshot(ctx context.Context) ([]File, error) {
	var files []File
	err := r.db.WithContext(ctx).Unscoped().Order("id").Find(&files).Error
	return files, err
}

func (r *Repository) RestoreSnapshot(ctx context.Context, files []File) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM files").Error; err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}
		return tx.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(&files).Error
	})
}
