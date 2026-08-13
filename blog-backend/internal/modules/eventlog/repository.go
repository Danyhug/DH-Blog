package eventlog

import (
	"context"

	"gorm.io/gorm"
)

// listFilter narrows the history query. Empty fields mean "no restriction".
type listFilter struct {
	Source   string
	Kind     string
	Status   string
	Page     int
	PageSize int
}

type repository struct {
	db *gorm.DB
}

func newRepository(db *gorm.DB) *repository { return &repository{db: db} }

func (r *repository) create(ctx context.Context, event *Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// list returns one page of history, newest first.
func (r *repository) list(ctx context.Context, filter listFilter) ([]*Event, int64, error) {
	query := r.db.WithContext(ctx).Model(&Event{})
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.Kind != "" {
		query = query.Where("kind = ?", filter.Kind)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	events := make([]*Event, 0, filter.PageSize)
	err := query.Order("id desc").
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

// since returns events newer than id, oldest first, so a client that dropped
// its socket can replay exactly what it missed instead of reloading the feed.
// The limit stops a client that was away for a week from asking for everything.
func (r *repository) since(ctx context.Context, id int64, limit int) ([]*Event, error) {
	events := make([]*Event, 0, limit)
	err := r.db.WithContext(ctx).
		Where("id > ?", id).
		Order("id asc").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// prune keeps the feed bounded by row count rather than by age: a personal
// blog can go quiet for months, and an age-based cutoff would throw away the
// only history there is.
func (r *repository) prune(ctx context.Context, keep int) (int64, error) {
	var maxID int64
	err := r.db.WithContext(ctx).Model(&Event{}).
		Select("COALESCE(MAX(id), 0)").
		Scan(&maxID).Error
	if err != nil {
		return 0, err
	}
	cutoff := maxID - int64(keep)
	if cutoff <= 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Where("id <= ?", cutoff).Delete(&Event{})
	return result.RowsAffected, result.Error
}
