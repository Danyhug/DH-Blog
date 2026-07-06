package repository

import (
	"errors"
	"fmt"

	"dh-blog/internal/model"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("用户不存在")

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// GetUserByUsername 根据用户名获取用户信息
func (r *UserRepository) GetUserByUsername(username string) (model.User, error) {
	var user model.User

	err := r.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("数据库查询用户失败: %w", err)
	}
	return user, nil
}

// IsFirstStart 查看是否是首次启动程序
func (r *UserRepository) IsFirstStart() bool {
	var count int64
	r.DB.Model(&model.User{}).Count(&count)
	return count == 0
}

// CreateUser 创建新用户
func (r *UserRepository) CreateUser(user *model.User) error {
	err := r.DB.Create(user).Error
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}
