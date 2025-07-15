package repository

import (
	"dh-blog/internal/model"
)

// IUserRepository 用户存储接口
// 定义了用户相关的数据操作方法
type IUserRepository interface {
	// GetUserByUsername 根据用户名获取用户信息
	GetUserByUsername(username string) (model.User, error)

	// CreateUser 创建新用户
	CreateUser(user *model.User) error
}

// IArticleRepository 文章存储接口
// 定义了文章相关的数据操作方法
type IArticleRepository interface {
	// 文章存储的基本操作接口
	// 实际项目中应该根据具体需求添加方法
	// 例如：创建、更新、删除、查询文章等
}

// ICommentRepository 评论存储接口
// 定义了评论相关的数据操作方法
type ICommentRepository interface {
	// 评论存储的基本操作接口
	// 实际项目中应该根据具体需求添加方法
	// 例如：发表评论、回复评论、删除评论等
}

// ICategoryRepository 分类存储接口
// 定义了分类相关的数据操作方法
type ICategoryRepository interface {
	// 分类存储的基本操作接口
	// 实际项目中应该根据具体需求添加方法
	// 例如：创建分类、更新分类、删除分类等
}

// ITagRepository 标签存储接口
// 定义了标签相关的数据操作方法
type ITagRepository interface {
	// 标签存储的基本操作接口
	// 实际项目中应该根据具体需求添加方法
	// 例如：创建标签、更新标签、删除标签等
}

// ISystemSettingRepository 系统设置存储接口
// 定义了系统设置相关的数据操作方法
type ISystemSettingRepository interface {
	// 系统设置存储的基本操作接口
	// 实际项目中应该根据具体需求添加方法
	// 例如：获取系统设置、更新系统设置等
}

// ILogRepository 日志存储接口
// 定义了日志相关的数据操作方法
type ILogRepository interface {
	// 日志存储的基本操作接口
	// 实际项目中应该根据具体需求添加方法
	// 例如：记录日志、查询日志等
}

// IFileRepository 接口定义在 file.go 文件中
// 定义了文件存储相关的数据操作方法
