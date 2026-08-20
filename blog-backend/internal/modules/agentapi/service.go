package agentapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"dh-blog/internal/model"

	"gorm.io/gorm"
)

// Grant validation errors. They are sentinels so the tool layer can branch on
// them, and their messages are user-facing Chinese so the agent gets a reason
// it can act on (or relay to the owner).
var (
	// ErrGrantNotFound means the token did not match any stored grant. It is
	// the umbrella for "no such grant" and "tampered token".
	ErrGrantNotFound = errors.New("该授权不存在或已失效")
	// ErrGrantExpired means the grant outlived its one-hour window.
	ErrGrantExpired = errors.New("授权已过期，请重新签发")
	// ErrGrantRevoked means the owner pulled the grant back early.
	ErrGrantRevoked = errors.New("该授权已被吊销")
	// ErrGrantWrongArticle means the grant is bound to another article.
	ErrGrantWrongArticle = errors.New("该授权只允许修改指定的文章")
	// ErrGrantUnusable is returned when a grant passed every in-memory check but
	// the atomic write that counts the use matched no row — the grant was
	// revoked or expired by somebody else between our read and our write. The
	// caller must treat it like a dead grant and ask the owner for a new one.
	ErrGrantUnusable = errors.New("该授权已失效，请重新索取")
	// ErrGrantInvalidArticleID means the requested article binding is
	// impossible. A negative id is a client mistake, not a missing row.
	ErrGrantInvalidArticleID = errors.New("articleId 不能为负数")
)

// grantTTL is how long an issued grant stays valid. One hour matches the
// design: long enough for a working session, short enough that a leaked token
// is only dangerous briefly.
const grantTTL = time.Hour

// Grant token format, mirroring the APIKey pattern: a readable literal, a
// random body, a stored hash for validation and a short public prefix for the
// admin list. The generator lives here rather than importing aigateway
// because the dependency direction is the other way: aigateway consumes this
// module's tools, so importing it back would create a cycle.
const (
	grantTokenLiteral      = "ag_grant_"
	grantTokenRandomLength = 32
	grantPrefixDigits      = 8
)

const grantTokenAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateGrantToken returns a fresh plaintext token.
func generateGrantToken() (string, error) {
	buffer := make([]byte, grantTokenRandomLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成授权 Token 失败: %w", err)
	}
	var builder strings.Builder
	builder.WriteString(grantTokenLiteral)
	for _, b := range buffer {
		builder.WriteByte(grantTokenAlphabet[int(b)%len(grantTokenAlphabet)])
	}
	return builder.String(), nil
}

// grantPrefixOf returns the indexed lookup prefix of a plaintext token.
func grantPrefixOf(plain string) string {
	limit := len(grantTokenLiteral) + grantPrefixDigits
	if len(plain) < limit {
		return plain
	}
	return plain[:limit]
}

// hashGrantToken digests a plaintext token for storage and comparison. SHA-256
// mirrors HashAPIKey: every validation hashes the token, and a password hash
// would dominate that latency for no extra protection on a one-hour credential.
func hashGrantToken(plain string) string {
	digest := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(digest[:])
}

// grantService issues, validates and revokes EditGrants.
type grantService struct {
	repo *grantRepository
	now  func() time.Time
}

func newGrantService(repo *grantRepository) *grantService {
	return &grantService{repo: repo, now: time.Now}
}

// Grant issues a new grant for the given article (0 = any article) and note.
// Expired grants are swept as a side effect of issuing, so the table never
// grows without a timer of its own.
func (s *grantService) Grant(articleID int, note string) (*EditGrant, error) {
	if articleID < 0 {
		return nil, ErrGrantInvalidArticleID
	}
	// Sweep is best-effort housekeeping: a failing cleanup must not block
	// issuing the new grant.
	_, _ = s.repo.DeleteExpired(s.now())

	plain, err := generateGrantToken()
	if err != nil {
		return nil, err
	}
	grant := &EditGrant{
		TokenPrefix: grantPrefixOf(plain),
		TokenHash:   hashGrantToken(plain),
		TokenPlain:  plain,
		ExpireAt:    model.JSONTime{Time: s.now().Add(grantTTL)},
		ArticleID:   articleID,
		Note:        strings.TrimSpace(note),
	}
	if err := s.repo.Create(grant); err != nil {
		return nil, err
	}
	return grant, nil
}

// Validate checks a token against the grant for the given article. A successful
// validation counts (UsedCount++, LastUsedAt) and returns the grant; every
// failure path returns a distinct sentinel error and leaves the counters alone,
// so the audit trail reflects only real uses.
func (s *grantService) Validate(articleID int, token string) (*EditGrant, error) {
	token = strings.TrimSpace(token)
	// 校验就是「拿明文的摘要去查行」：查得到即等值，没有可比较的第二个值，
	// 因此这里不需要（也无法）再做一次恒定时间比较。
	grant, err := s.repo.ByHash(hashGrantToken(token))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGrantNotFound
		}
		return nil, err
	}

	now := s.now()
	if !grant.ExpireAt.After(now) {
		return nil, ErrGrantExpired
	}
	if grant.Revoked {
		return nil, ErrGrantRevoked
	}
	if grant.ArticleID != 0 && grant.ArticleID != articleID {
		return nil, ErrGrantWrongArticle
	}

	// The counters on the returned grant track what the caller (the admin
	// list, the tool echo) should see; the truth lives in the atomic row
	// update below. If that update matches zero rows, the grant was revoked
	// or expired between the read above and this write — refuse rather than
	// resurrect or double-count it.
	grant.UsedCount++
	lastUsed := now
	grant.LastUsedAt = &lastUsed
	used, err := s.repo.IncrementUsed(grant.ID, now)
	if err != nil {
		return nil, err
	}
	if !used {
		return nil, ErrGrantUnusable
	}
	return grant, nil
}

// Revoke pulls a grant back early. The row is kept (revocation is not
// deletion) so the UsedCount audit stays queryable until it expires. The flip
// is a single per-column update, so it never drags the writer's own stale
// counters back into the row.
func (s *grantService) Revoke(id int) error {
	grant, err := s.repo.ByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGrantNotFound
		}
		return err
	}
	return s.repo.MarkRevoked(grant.ID)
}

// Reveal returns the plaintext token so the owner can copy it again. Refused
// once the grant is revoked, since a revoked token is exactly the one that
// must stop circulating.
func (s *grantService) Reveal(id int) (string, error) {
	grant, err := s.repo.ByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrGrantNotFound
		}
		return "", err
	}
	if grant.Revoked {
		return "", ErrGrantRevoked
	}
	return grant.TokenPlain, nil
}

// List returns the grants that are still valid at now, newest first.
func (s *grantService) List(now time.Time) ([]EditGrant, error) {
	return s.repo.ListActive(now)
}
