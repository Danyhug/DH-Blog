package repository

import (
	"gorm.io/gorm"
)

// IRepository 仓库层总接口
// 提供获取各个具体资源存储库的方法
type IRepository interface {
	// 文件存储
	GetFileRepository() IFileRepository // 获取文件存储库接口
}

// Repository 数据仓库实现
// 集中管理所有资源的存储实现
type Repository struct {
	fileRepo IFileRepository // 文件存储库
}

// NewRepository 创建仓库实例
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - IRepository: 仓库层总接口实现
func NewRepository(db *gorm.DB) IRepository {
	// 创建文件存储
	fileRepo := NewFileRepository(db)

	return &Repository{
		fileRepo: fileRepo,
	}
}

// GetFileRepository 获取文件存储接口
// 返回:
//   - IFileRepository: 文件存储接口
func (r *Repository) GetFileRepository() IFileRepository {
	return r.fileRepo
}
