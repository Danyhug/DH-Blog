package agentapi

import (
	"time"

	"gorm.io/gorm"
)

// grantRepository is the persistence layer for EditGrant. Rows are never
// updated by partial updates: every mutation loads the row first and saves the
// whole struct, so the audit fields (UsedCount, LastUsedAt) can never be
// clobbered by a stale zero value.
type grantRepository struct {
	db *gorm.DB
}

func (r *grantRepository) Create(grant *EditGrant) error {
	return r.db.Create(grant).Error
}

func (r *grantRepository) ByHash(hash string) (*EditGrant, error) {
	var grant EditGrant
	if err := r.db.Where("token_hash = ?", hash).First(&grant).Error; err != nil {
		return nil, err
	}
	return &grant, nil
}

func (r *grantRepository) ByID(id int) (*EditGrant, error) {
	var grant EditGrant
	if err := r.db.First(&grant, id).Error; err != nil {
		return nil, err
	}
	return &grant, nil
}

// ListActive returns the grants that are neither expired nor revoked, newest
// first.
func (r *grantRepository) ListActive(now time.Time) ([]EditGrant, error) {
	var grants []EditGrant
	if err := r.db.Where("expire_at > ? AND revoked = ?", now, false).Order("id DESC").Find(&grants).Error; err != nil {
		return nil, err
	}
	return grants, nil
}

func (r *grantRepository) Update(grant *EditGrant) error {
	return r.db.Save(grant).Error
}

// DeleteExpired physically removes expired rows. A hard delete is right here
// because an expired grant is dead weight with no audit value left: it cannot
// validate, and its only remaining story (that it was used twice before the
// hour ran out) lives in the counters of grants that are still active.
// Revoked grants, by contrast, are kept until they expire on their own, so the
// UsedCount audit stays queryable while the grant is still relevant.
func (r *grantRepository) DeleteExpired(now time.Time) (int64, error) {
	result := r.db.Unscoped().Where("expire_at <= ?", now).Delete(&EditGrant{})
	return result.RowsAffected, result.Error
}
