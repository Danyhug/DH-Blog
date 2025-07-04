package repository

import (
	"errors"
	"fmt"
	"time"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type DailyStatsRepository struct {
	DB *gorm.DB
}

func NewDailyStatsRepository(db *gorm.DB) *DailyStatsRepository {
	return &DailyStatsRepository{DB: db}
}

// CreateDailyStat 创建每日统计
func (r *DailyStatsRepository) CreateDailyStat(stat *model.DailyStat) error {
	err := r.DB.Create(stat).Error
	if err != nil {
		return fmt.Errorf("创建每日统计失败: %w", err)
	}
	return nil
}

// GetDailyStatsByDate 根据日期获取每日统计
func (r *DailyStatsRepository) GetDailyStatsByDate(date time.Time) (model.DailyStat, error) {
	var stat model.DailyStat
	err := r.DB.Where("date = ?", date.Format("2006-01-02")).First(&stat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DailyStat{}, fmt.Errorf("查询每日统计失败: %w", errs.ErrNotFound)
		}
		return model.DailyStat{}, fmt.Errorf("数据库查询每日统计失败: %w", err)
	}
	return stat, nil
}

// CountArticles 获取文章总数
func (r *DailyStatsRepository) CountArticles() (int64, error) {
	var count int64
	err := r.DB.Model(&model.Article{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("获取文章总数失败: %w", err)
	}
	return count, nil
}

// CountTags 获取标签总数
func (r *DailyStatsRepository) CountTags() (int64, error) {
	var count int64
	err := r.DB.Model(&model.Tag{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("获取标签总数失败: %w", err)
	}
	return count, nil
}

// CountComments 获取评论总数
func (r *DailyStatsRepository) CountComments() (int64, error) {
	var count int64
	err := r.DB.Model(&model.Comment{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("获取评论总数失败: %w", err)
	}
	return count, nil
}

// CountCategories 获取分类总数
func (r *DailyStatsRepository) CountCategories() (int64, error) {
	var count int64
	err := r.DB.Model(&model.Category{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("获取分类总数失败: %w", err)
	}
	return count, nil
}

// GetDailyStatsByDateRange 根据日期范围和分页查询每日统计数据
func (r *DailyStatsRepository) GetDailyStatsByDateRange(page, pageSize int, startDate, endDate *time.Time) ([]model.DailyStat, int64, error) {
	var dailyStats []model.DailyStat
	var total int64

	offset := (page - 1) * pageSize

	query := r.DB.Model(&model.DailyStat{})

	if startDate != nil {
		query = query.Where("date >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("date <= ?", endDate)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询每日统计总数失败: %w", err)
	}

	// 查询每日统计列表
	if err := query.Offset(offset).Limit(pageSize).Order("date desc").Find(&dailyStats).Error; err != nil {
		return nil, 0, fmt.Errorf("查询每日统计列表失败: %w", err)
	}

	return dailyStats, total, nil
}
