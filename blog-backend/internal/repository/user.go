package repository

import (
	"errors"
	"fmt"

	"dh-blog/internal/model"
	"gorm.io/gorm"
)

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
			return model.User{}, fmt.Errorf("查询用户失败: %w", gorm.ErrRecordNotFound)
		}
		return model.User{}, fmt.Errorf("数据库查询用户失败: %w", err)
	}
	return user, nil
}

// CreateUser 创建新用户
func (r *UserRepository) CreateUser(user *model.User) error {
	err := r.DB.Create(user).Error
	if err != nil {
		// 实际项目中需要更细致的错误类型判断，例如唯一索引冲突
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}
