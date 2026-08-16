package agentapi

import (
	"time"

	"gorm.io/gorm"
)

// grantRepository is the persistence layer for EditGrant. Every mutation is a
// narrow, atomic statement (validation counts via IncrementUsed, revocation via
// MarkRevoked) rather than a read-modify-save cycle, so concurrent readers and
// writers can never clobber the audit fields (Revoked, UsedCount, LastUsedAt).
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

// IncrementUsed atomically counts one successful validation. The predicate is
// what makes it race-proof: the WHERE clause re-checks revoked and expiry at
// write time, so a grant revoked (or expired) between the service's read and
// this UPDATE matches zero rows and the caller must refuse the use. The counter
// moves inside the single statement, so two overlapping validations can never
// clobber each other's increments.
func (r *grantRepository) IncrementUsed(id int, now time.Time) (bool, error) {
	result := r.db.Model(&EditGrant{}).
		Where("id = ? AND revoked = ? AND expire_at > ?", id, false, now).
		UpdateColumns(map[string]any{
			"used_count":   gorm.Expr("used_count + 1"),
			"last_used_at": now,
		})
	return result.RowsAffected == 1, result.Error
}

// MarkRevoked flips only the revoked column. Per-column update (rather than a
// whole-row Save) guarantees it never drags a stale used_count or last_used_at
// out of the writer's memory, and the bare id predicate makes it idempotent —
// revoking an already-revoked grant is a no-op, not an error.
func (r *grantRepository) MarkRevoked(id int) error {
	return r.db.Model(&EditGrant{}).Where("id = ?", id).Update("revoked", true).Error
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
