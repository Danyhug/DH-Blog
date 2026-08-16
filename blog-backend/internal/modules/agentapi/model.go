package agentapi

import (
	"time"

	"dh-blog/internal/model"
)

// EditGrant is the credential a site owner hands an AI Agent for rewriting
// existing articles.
//
// It exists because this blog picked "agent writes are published" with no draft
// review gate. Rewriting the owner's existing articles is therefore the riskiest
// action an agent can take: a misjudging model, or a prompt injected by a page
// read during a web search, could quietly change published content. That power
// is carved out of the long-lived credentials and granted instead as a short
// token the owner issues on purpose and that expires on its own in an hour.
type EditGrant struct {
	model.BaseModel `gorm:"embedded"`
	// TokenPrefix is the public part, used by the admin list and display.
	TokenPrefix string `gorm:"column:token_prefix;index;not null" json:"tokenPrefix"`
	// TokenHash is what validation compares against, in constant time.
	TokenHash string `gorm:"column:token_hash;index;not null" json:"-"`
	// TokenPlain is only ever read by the admin reveal action, so the owner can
	// copy the same token again. The request path and the list endpoint never
	// touch it.
	TokenPlain string `gorm:"column:token_plain;not null" json:"-"`
	// ExpireAt is the issue time plus one hour. Grants are reusable within that
	// window rather than single-use, and UsedCount records how often.
	ExpireAt model.JSONTime `gorm:"column:expire_at;not null" json:"expireAt"`
	// ArticleID binds the grant to a single article; 0 means any article.
	// Reserved for a future tightening of the rule, not exposed in the UI yet.
	ArticleID  int        `gorm:"column:article_id" json:"articleId"`
	Revoked    bool       `gorm:"column:revoked" json:"revoked"`
	UsedCount  int        `gorm:"column:used_count" json:"usedCount"`
	LastUsedAt *time.Time `gorm:"column:last_used_at" json:"lastUsedAt"`
	// Note is the owner's own remark, e.g. "让 Claude 改错别字".
	Note string `gorm:"column:note" json:"note"`
}

func (EditGrant) TableName() string { return "agent_edit_grants" }

// MigrationModels declares the database tables owned by this module.
func MigrationModels() []any {
	return []any{&EditGrant{}}
}
