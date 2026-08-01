package aigateway

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Gateway credential errors, all surfaced to the caller as 401.
var (
	ErrMissingAPIKey = errors.New("缺少 API Key")
	ErrInvalidAPIKey = errors.New("API Key 无效")
	ErrAPIKeyRevoked = errors.New("API Key 已停用")
	ErrAPIKeyExpired = errors.New("API Key 已过期")
)

// ErrRateLimited means the key exceeded its per-minute allowance.
var ErrRateLimited = errors.New("请求过于频繁，请稍后再试")

// ErrQuotaExceeded means the key used up its monthly allowance.
var ErrQuotaExceeded = errors.New("API Key 本月配额已用尽")

// apiKeyCacheTTL keeps hot keys out of the database without making a
// revocation take noticeably long to bite.
const apiKeyCacheTTL = 60 * time.Second

func apiKeyCacheKey(prefix string) string { return "gw:key:" + prefix }

// authenticate resolves and validates a plaintext gateway key.
func (s *Service) authenticate(ctx context.Context, plain string) (*APIKey, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return nil, ErrMissingAPIKey
	}

	prefix := APIKeyPrefixOf(plain)
	key, err := s.loadAPIKey(ctx, prefix)
	if err != nil {
		return nil, err
	}

	// Constant-time comparison: the prefix is public, the digest is not.
	if subtle.ConstantTimeCompare([]byte(HashAPIKey(plain)), []byte(key.KeyHash)) != 1 {
		return nil, ErrInvalidAPIKey
	}
	if !key.Enabled {
		return nil, ErrAPIKeyRevoked
	}
	if key.Expired(s.now()) {
		return nil, ErrAPIKeyExpired
	}
	return key, nil
}

func (s *Service) loadAPIKey(ctx context.Context, prefix string) (*APIKey, error) {
	if cached, found := s.cache.Get(apiKeyCacheKey(prefix)); found {
		if key, ok := cached.(APIKey); ok {
			copied := key
			return &copied, nil
		}
	}
	key, err := s.repo.apiKeyByPrefix(ctx, prefix)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}
	_ = s.cache.Set(apiKeyCacheKey(prefix), key, apiKeyCacheTTL)
	copied := key
	return &copied, nil
}

// invalidateAPIKey drops a cached credential after an admin change so a
// revocation or quota edit takes effect immediately.
func (s *Service) invalidateAPIKey(prefix string) {
	s.cache.Delete(apiKeyCacheKey(prefix))
}

// minuteCounters tracks per-key requests inside the current minute. It is kept
// in memory rather than in the shared cache so the increment stays atomic.
type minuteCounters struct {
	mu     sync.Mutex
	counts map[int]*minuteCount
}

type minuteCount struct {
	minute int64
	count  int
}

func newMinuteCounters() *minuteCounters {
	return &minuteCounters{counts: make(map[int]*minuteCount)}
}

// allow records one request and reports whether it stays within limit. A
// non-positive limit disables the check.
func (m *minuteCounters) allow(id, limit int, now time.Time) bool {
	if limit <= 0 {
		return true
	}
	minute := now.Unix() / 60

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(minute)

	entry, ok := m.counts[id]
	if !ok || entry.minute != minute {
		m.counts[id] = &minuteCount{minute: minute, count: 1}
		return true
	}
	if entry.count >= limit {
		return false
	}
	entry.count++
	return true
}

// pruneLocked drops counters from earlier minutes once the map grows enough to
// be worth sweeping.
func (m *minuteCounters) pruneLocked(minute int64) {
	if len(m.counts) < 512 {
		return
	}
	for id, entry := range m.counts {
		if entry.minute != minute {
			delete(m.counts, id)
		}
	}
}

// extractAPIKey reads the credential from either accepted header.
func extractAPIKey(authorization, apiKeyHeader string) string {
	if header := strings.TrimSpace(apiKeyHeader); header != "" {
		return header
	}
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return ""
	}
	if len(authorization) > 7 && strings.EqualFold(authorization[:7], "bearer ") {
		return strings.TrimSpace(authorization[7:])
	}
	return authorization
}
