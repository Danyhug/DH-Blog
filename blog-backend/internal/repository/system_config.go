package repository

import (
	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type SystemConfigRepository interface {
	Get() (*model.SystemConfig, error)
	Update(config *model.SystemConfig) error
}

type systemConfigRepository struct {
	db *gorm.DB
}

func NewSystemConfigRepository(db *gorm.DB) SystemConfigRepository {
	return &systemConfigRepository{db: db}
}

func (r *systemConfigRepository) Get() (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := r.db.First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *systemConfigRepository) Update(config *model.SystemConfig) error {
	return r.db.Save(config).Error
}
