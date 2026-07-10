package user

import (
	"errors"
	"fmt"

	"dh-blog/internal/utils"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("用户不存在")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByUsername(username string) (User, error) {
	var user User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("数据库查询用户失败: %w", err)
	}
	return user, nil
}

func (r *Repository) Authenticate(username, password string) bool {
	user, err := r.GetByUsername(username)
	return err == nil && utils.CheckPasswordHash(password, user.Password)
}

func (r *Repository) IsFirstStart() bool {
	var count int64
	r.db.Model(&User{}).Count(&count)
	return count == 0
}

func (r *Repository) Create(user *User) error {
	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}
