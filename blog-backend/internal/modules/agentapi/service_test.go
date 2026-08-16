package agentapi

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func fixedNow() time.Time {
	return time.Date(2026, 8, 16, 10, 0, 0, 0, time.Local)
}

func countGrants(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var total int64
	if err := db.Model(&EditGrant{}).Count(&total).Error; err != nil {
		t.Fatalf("count grants: %v", err)
	}
	return total
}

func loadGrant(t *testing.T, db *gorm.DB, id int) EditGrant {
	t.Helper()
	var grant EditGrant
	if err := db.First(&grant, id).Error; err != nil {
		t.Fatalf("load grant %d: %v", id, err)
	}
	return grant
}

func TestGrantIssuesWellFormedToken(t *testing.T) {
	now := fixedNow()
	service, repo := newTestService(t, now)

	grant, err := service.Grant(0, "让 Claude 改错别字")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if !strings.HasPrefix(grant.TokenPlain, "ag_grant_") {
		t.Fatalf("token = %q, want ag_grant_ prefix", grant.TokenPlain)
	}
	if len(grant.TokenPlain) != len("ag_grant_")+32 {
		t.Fatalf("token length = %d, want %d", len(grant.TokenPlain), len("ag_grant_")+32)
	}
	if grant.TokenPrefix != "ag_grant_"+grant.TokenPlain[len("ag_grant_"):len("ag_grant_")+8] {
		t.Fatalf("prefix = %q, want literal plus first 8 random chars", grant.TokenPrefix)
	}
	if grant.TokenHash != hashGrantToken(grant.TokenPlain) {
		t.Fatal("stored hash does not match the plaintext token")
	}
	wantExpire := now.Add(grantTTL)
	if !grant.ExpireAt.Equal(wantExpire) {
		t.Fatalf("expireAt = %v, want %v", grant.ExpireAt.Time, wantExpire)
	}
	if grant.ArticleID != 0 || grant.Revoked || grant.UsedCount != 0 {
		t.Fatalf("grant = %#v, want zero articleId/revoked/usedCount", grant)
	}
	if grant.Note != "让 Claude 改错别字" {
		t.Fatalf("note = %q", grant.Note)
	}

	stored := loadGrant(t, repo.db, grant.ID)
	if stored.TokenHash != grant.TokenHash || stored.TokenPlain != grant.TokenPlain {
		t.Fatalf("stored grant does not match issued grant: %#v", stored)
	}
}

func TestGrantRejectsNegativeArticleID(t *testing.T) {
	service, _ := newTestService(t, fixedNow())
	if _, err := service.Grant(-1, ""); !errors.Is(err, ErrGrantInvalidArticleID) {
		t.Fatalf("grant(-1) err = %v, want ErrGrantInvalidArticleID", err)
	}
}

func TestGrantCleansUpExpiredWhileIssuing(t *testing.T) {
	now := fixedNow()
	service, repo := newTestService(t, now)

	first, err := service.Grant(0, "第一张")
	if err != nil {
		t.Fatalf("grant first: %v", err)
	}
	second, err := service.Grant(0, "第二张")
	if err != nil {
		t.Fatalf("grant second: %v", err)
	}
	if total := countGrants(t, repo.db); total != 2 {
		t.Fatalf("grant count = %d, want 2", total)
	}

	// 把第一张的过期时间改到过去，再签第三张 —— 第一张应被顺带物理删除
	past := now.Add(-time.Minute)
	if err := repo.db.Model(&EditGrant{}).Where("id = ?", first.ID).Update("expire_at", past).Error; err != nil {
		t.Fatalf("backdate first grant: %v", err)
	}
	if _, err := service.Grant(0, "第三张"); err != nil {
		t.Fatalf("grant third: %v", err)
	}

	if total := countGrants(t, repo.db); total != 2 {
		t.Fatalf("grant count = %d, want 2 after cleanup", total)
	}
	if err := repo.db.First(&EditGrant{}, first.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expired grant %d still reachable: %v", first.ID, err)
	}
	if err := repo.db.Unscoped().First(&EditGrant{}, second.ID).Error; err != nil {
		t.Fatalf("active grant %d was cleaned up too: %v", second.ID, err)
	}
}

func TestValidateCountsEverySuccessfulUse(t *testing.T) {
	now := fixedNow()
	service, repo := newTestService(t, now)

	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	first, err := service.Validate(0, grant.TokenPlain)
	if err != nil {
		t.Fatalf("first validate: %v", err)
	}
	if first.UsedCount != 1 || first.LastUsedAt == nil {
		t.Fatalf("after first use: usedCount = %d, lastUsedAt = %v", first.UsedCount, first.LastUsedAt)
	}

	later := now.Add(time.Minute)
	service.now = func() time.Time { return later }
	second, err := service.Validate(0, grant.TokenPlain)
	if err != nil {
		t.Fatalf("second validate: %v", err)
	}
	if second.UsedCount != 2 {
		t.Fatalf("after second use: usedCount = %d, want 2", second.UsedCount)
	}
	if second.LastUsedAt == nil || !second.LastUsedAt.Equal(later) {
		t.Fatalf("lastUsedAt = %v, want %v", second.LastUsedAt, later)
	}

	stored := loadGrant(t, repo.db, grant.ID)
	if stored.UsedCount != 2 || stored.LastUsedAt == nil || !stored.LastUsedAt.Equal(later) {
		t.Fatalf("stored grant = %#v, want usedCount 2 and lastUsedAt %v", stored, later)
	}
}

func TestValidateRejectsUnknownTokenWithoutCounting(t *testing.T) {
	service, repo := newTestService(t, fixedNow())
	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	if _, err := service.Validate(0, "ag_grant_notARealToken0000000000000000"); !errors.Is(err, ErrGrantNotFound) {
		t.Fatalf("unknown token err = %v, want ErrGrantNotFound", err)
	}
	if _, err := service.Validate(0, grant.TokenPlain+"x"); !errors.Is(err, ErrGrantNotFound) {
		t.Fatalf("tampered token err = %v, want ErrGrantNotFound", err)
	}
	if _, err := service.Validate(0, "  "); !errors.Is(err, ErrGrantNotFound) {
		t.Fatalf("blank token err = %v, want ErrGrantNotFound", err)
	}
	if stored := loadGrant(t, repo.db, grant.ID); stored.UsedCount != 0 {
		t.Fatalf("usedCount = %d, want 0 (failed attempts must not count)", stored.UsedCount)
	}
}

func TestValidateRejectsExpiredWithoutCounting(t *testing.T) {
	now := fixedNow()
	service, repo := newTestService(t, now)

	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if _, err := service.Validate(0, grant.TokenPlain); err != nil {
		t.Fatalf("validate before expiry: %v", err)
	}

	service.now = func() time.Time { return now.Add(grantTTL) }
	if _, err := service.Validate(0, grant.TokenPlain); !errors.Is(err, ErrGrantExpired) {
		t.Fatalf("expired token err = %v, want ErrGrantExpired", err)
	}

	stored := loadGrant(t, repo.db, grant.ID)
	if stored.UsedCount != 1 {
		t.Fatalf("usedCount = %d, want 1 (expired attempt must not count)", stored.UsedCount)
	}
}

func TestValidateRejectsRevokedWithoutCounting(t *testing.T) {
	service, repo := newTestService(t, fixedNow())
	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if err := service.Revoke(grant.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := service.Validate(0, grant.TokenPlain); !errors.Is(err, ErrGrantRevoked) {
		t.Fatalf("revoked token err = %v, want ErrGrantRevoked", err)
	}
	if stored := loadGrant(t, repo.db, grant.ID); stored.UsedCount != 0 {
		t.Fatalf("usedCount = %d, want 0 (revoked attempt must not count)", stored.UsedCount)
	}
}

func TestValidateRejectsWrongArticleWithoutCounting(t *testing.T) {
	service, repo := newTestService(t, fixedNow())
	grant, err := service.Grant(42, "只改第 42 篇")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if _, err := service.Validate(7, grant.TokenPlain); !errors.Is(err, ErrGrantWrongArticle) {
		t.Fatalf("wrong article err = %v, want ErrGrantWrongArticle", err)
	}
	if stored := loadGrant(t, repo.db, grant.ID); stored.UsedCount != 0 {
		t.Fatalf("usedCount = %d, want 0 (wrong article must not count)", stored.UsedCount)
	}
	if _, err := service.Validate(42, grant.TokenPlain); err != nil {
		t.Fatalf("validate bound article: %v", err)
	}
	if stored := loadGrant(t, repo.db, grant.ID); stored.UsedCount != 1 {
		t.Fatalf("usedCount = %d, want 1 after bound-article use", stored.UsedCount)
	}
}

func TestFailedValidateKeepsExpiredGrantInDatabase(t *testing.T) {
	now := fixedNow()
	service, repo := newTestService(t, now)
	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	service.now = func() time.Time { return now.Add(2 * grantTTL) }
	if _, err := service.Validate(0, grant.TokenPlain); !errors.Is(err, ErrGrantExpired) {
		t.Fatalf("expired token err = %v, want ErrGrantExpired", err)
	}
	if err := repo.db.First(&EditGrant{}, grant.ID).Error; err != nil {
		t.Fatalf("expired grant was deleted by Validate: %v", err)
	}
}

func TestRevokeFailsForUnknownID(t *testing.T) {
	service, _ := newTestService(t, fixedNow())
	if err := service.Revoke(4242); !errors.Is(err, ErrGrantNotFound) {
		t.Fatalf("revoke unknown id err = %v, want ErrGrantNotFound", err)
	}
}

func TestRevealReturnsPlaintextOnlyWhileActive(t *testing.T) {
	service, _ := newTestService(t, fixedNow())
	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	plain, err := service.Reveal(grant.ID)
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if plain != grant.TokenPlain {
		t.Fatalf("reveal = %q, want %q", plain, grant.TokenPlain)
	}
	if _, err := service.Reveal(4242); !errors.Is(err, ErrGrantNotFound) {
		t.Fatalf("reveal unknown id err = %v, want ErrGrantNotFound", err)
	}
	if err := service.Revoke(grant.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := service.Reveal(grant.ID); !errors.Is(err, ErrGrantRevoked) {
		t.Fatalf("reveal after revoke err = %v, want ErrGrantRevoked", err)
	}
}

// TestGrantIncrementUsedAccumulatesAndTouchesLastUsed pins the atomic row
// update Validate relies on: two successful increments land as used_count = 2
// with last_used_at from the second call, written by one UPDATE rather than a
// read-modify-save round trip.
func TestGrantIncrementUsedAccumulatesAndTouchesLastUsed(t *testing.T) {
	service, repo := newTestService(t, fixedNow())
	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	later := fixedNow().Add(time.Minute)
	for i := 0; i < 2; i++ {
		ok, err := repo.IncrementUsed(grant.ID, later)
		if err != nil {
			t.Fatalf("increment %d: %v", i+1, err)
		}
		if !ok {
			t.Fatalf("increment %d returned false, want true", i+1)
		}
	}

	stored := loadGrant(t, repo.db, grant.ID)
	if stored.UsedCount != 2 {
		t.Fatalf("usedCount = %d, want 2", stored.UsedCount)
	}
	if stored.LastUsedAt == nil || !stored.LastUsedAt.Equal(later) {
		t.Fatalf("lastUsedAt = %v, want %v", stored.LastUsedAt, later)
	}
}

// TestGrantIncrementUsedRefusedAfterRevoke reproduces the review race at the
// invariant level (SQLite serializes writes, so the interleaving is ordered by
// hand): once a grant is revoked, every later IncrementUsed must be refused and
// the used_count tally must not gain a phantom count.
func TestGrantIncrementUsedRefusedAfterRevoke(t *testing.T) {
	service, repo := newTestService(t, fixedNow())
	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	ok, err := repo.IncrementUsed(grant.ID, fixedNow())
	if err != nil || !ok {
		t.Fatalf("first increment: ok=%v err=%v", ok, err)
	}

	if err := repo.MarkRevoked(grant.ID); err != nil {
		t.Fatalf("mark revoked: %v", err)
	}
	// Idempotent: revoking again is a no-op, not an error.
	if err := repo.MarkRevoked(grant.ID); err != nil {
		t.Fatalf("mark revoked twice: %v", err)
	}
	for i := 0; i < 3; i++ {
		ok, err := repo.IncrementUsed(grant.ID, fixedNow())
		if err != nil {
			t.Fatalf("increment after revoke %d: %v", i+1, err)
		}
		if ok {
			t.Fatalf("increment %d after revoke returned true, want false", i+1)
		}
	}

	stored := loadGrant(t, repo.db, grant.ID)
	if stored.UsedCount != 1 {
		t.Fatalf("usedCount = %d, want 1 (revoked uses must not count)", stored.UsedCount)
	}
	if !stored.Revoked {
		t.Fatal("revoked flag was lost")
	}
}

// TestGrantIncrementUsedRefusedWhenExpired covers the second half of the
// read-then-write window: a grant that passes the in-memory expiry check but
// expires before the atomic write must not count either.
func TestGrantIncrementUsedRefusedWhenExpired(t *testing.T) {
	service, repo := newTestService(t, fixedNow())
	grant, err := service.Grant(0, "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	past := fixedNow().Add(-time.Minute)
	if err := repo.db.Model(&EditGrant{}).Where("id = ?", grant.ID).Update("expire_at", past).Error; err != nil {
		t.Fatalf("backdate grant: %v", err)
	}
	ok, err := repo.IncrementUsed(grant.ID, fixedNow())
	if err != nil {
		t.Fatalf("increment expired: %v", err)
	}
	if ok {
		t.Fatal("increment on an expired grant succeeded, want false")
	}
	if stored := loadGrant(t, repo.db, grant.ID); stored.UsedCount != 0 {
		t.Fatalf("usedCount = %d, want 0", stored.UsedCount)
	}
}

func TestListOnlyShowsActiveGrantsNewestFirst(t *testing.T) {
	now := fixedNow()
	service, repo := newTestService(t, now)

	first, err := service.Grant(0, "第一张")
	if err != nil {
		t.Fatalf("grant first: %v", err)
	}
	second, err := service.Grant(0, "第二张")
	if err != nil {
		t.Fatalf("grant second: %v", err)
	}
	third, err := service.Grant(0, "第三张")
	if err != nil {
		t.Fatalf("grant third: %v", err)
	}

	// 吊销第二张、让第三张过期
	if err := service.Revoke(second.ID); err != nil {
		t.Fatalf("revoke second: %v", err)
	}
	past := now.Add(-time.Minute)
	if err := repo.db.Model(&EditGrant{}).Where("id = ?", third.ID).Update("expire_at", past).Error; err != nil {
		t.Fatalf("backdate third grant: %v", err)
	}

	list, err := service.List(now)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != first.ID {
		t.Fatalf("list = %#v, want only grant %d", list, first.ID)
	}
}
