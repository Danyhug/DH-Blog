package aigateway

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"dh-blog/internal/model"
)

// Provider is one configured upstream search API.
//
// None of the columns carry a GORM `default:` tag on purpose: GORM omits a
// zero-valued field from the INSERT when the column has a default, which would
// silently turn "rps: 0" (no pacing) or "monthlyQuota: 0" (no ceiling) into the
// default value. Every write below sets its values explicitly instead.
type Provider struct {
	model.BaseModel `gorm:"embedded"`
	Name            string  `gorm:"column:name;uniqueIndex;not null" json:"name"`
	DisplayName     string  `gorm:"column:display_name" json:"displayName"`
	Enabled         bool    `gorm:"column:enabled" json:"enabled"`
	APIKey          string  `gorm:"column:api_key" json:"-"`
	BaseURL         string  `gorm:"column:base_url" json:"baseUrl"`
	Priority        int     `gorm:"column:priority" json:"priority"`
	Weight          int     `gorm:"column:weight" json:"weight"`
	RPS             float64 `gorm:"column:rps" json:"rps"`
	MonthlyQuota    int     `gorm:"column:monthly_quota" json:"monthlyQuota"`
	// MonthlyCostLimit is a spend ceiling in millionths of a US dollar, for the
	// providers that bill by amount instead of by call. Exa is the reason it
	// exists: its free tier is $10 a month and a request costs whatever it
	// costs, so a request count cannot express the budget that actually runs
	// out. Zero means no ceiling, like MonthlyQuota.
	MonthlyCostLimit int    `gorm:"column:monthly_cost_limit" json:"monthlyCostLimitMicroUsd"`
	Extra            string `gorm:"column:extra" json:"extra"`
}

func (Provider) TableName() string { return "ai_gateway_providers" }

// ProviderKey is one upstream credential. A provider may hold several: the
// gateway rotates through them, and parks the ones the upstream rejects so a
// dead credential stops being scheduled instead of failing every other request.
type ProviderKey struct {
	model.BaseModel `gorm:"embedded"`
	Provider        string `gorm:"column:provider;index;not null" json:"provider"`
	// Label is the operator's own note, e.g. which account the key belongs to.
	Label      string     `gorm:"column:label" json:"label"`
	APIKey     string     `gorm:"column:api_key;not null" json:"-"`
	Enabled    bool       `gorm:"column:enabled" json:"enabled"`
	Status     string     `gorm:"column:status" json:"status"`
	LastError  string     `gorm:"column:last_error" json:"lastError"`
	LastUsedAt *time.Time `gorm:"column:last_used_at" json:"lastUsedAt"`
	DisabledAt *time.Time `gorm:"column:disabled_at" json:"disabledAt"`

	// MonthlyQuota and MonthlyCostLimit are this credential's own allowance, and
	// mirror the columns of the same name on Provider. They exist because an
	// allowance belongs to an account rather than to an upstream: two Tavily
	// accounts of 1000 credits each add up to 2000, and one provider-level number
	// cannot express that — it either wastes half the capacity or overruns the
	// half that is left once a credential drops out. Zero means "no ceiling of
	// its own"; a provider whose credentials all say zero falls back to the
	// provider-level pair, so an install that never touches this keeps behaving
	// exactly as it did.
	MonthlyQuota     int `gorm:"column:monthly_quota" json:"monthlyQuota"`
	MonthlyCostLimit int `gorm:"column:monthly_cost_limit" json:"monthlyCostLimitMicroUsd"`
	// UsageKeyID identifies this credential to the provider's usage API, for the
	// upstreams that account per key rather than per account. It has to live here
	// rather than on the provider: one id shared by every credential makes them
	// all report the same key's figure, and summing those multiplies a single
	// key's spend by the number of credentials in rotation.
	UsageKeyID string `gorm:"column:usage_key_id" json:"usageKeyId"`

	// Upstream* mirror what the provider itself reports for this credential,
	// refreshed on a timer. They exist because the gateway's own counter only
	// sees traffic that went through the gateway: the same key called from a
	// laptop, or anything it spent before the gateway existed, is invisible
	// locally. UpstreamUnit records what the numbers count, since a Tavily
	// credit and a Brave request are not the same thing.
	UpstreamUsed     int        `gorm:"column:upstream_used" json:"upstreamUsed"`
	UpstreamLimit    int        `gorm:"column:upstream_limit" json:"upstreamLimit"`
	UpstreamUnit     string     `gorm:"column:upstream_unit" json:"upstreamUnit"`
	UpstreamScope    string     `gorm:"column:upstream_scope" json:"upstreamScope"`
	UpstreamWindow   string     `gorm:"column:upstream_window" json:"upstreamWindow"`
	UpstreamSyncedAt *time.Time `gorm:"column:upstream_synced_at" json:"upstreamSyncedAt"`
	UpstreamError    string     `gorm:"column:upstream_error" json:"upstreamError"`
}

func (ProviderKey) TableName() string { return "ai_gateway_provider_keys" }

// Credential states. Anything other than active is out of rotation.
const (
	ProviderKeyActive        = "active"
	ProviderKeyAuthFailed    = "auth_failed"
	ProviderKeyQuotaExceeded = "quota_exceeded"
)

// upstreamUsageTTL bounds how long a synced number may override the calendar
// guess below. A stale figure must not keep a credential parked forever if the
// sync itself is what broke.
const upstreamUsageTTL = 24 * time.Hour

// UpstreamExhausted reports whether the provider's own recent accounting says
// this credential has nothing left to spend.
func (k *ProviderKey) UpstreamExhausted(now time.Time) bool {
	if k.UpstreamLimit <= 0 || k.UpstreamSyncedAt == nil {
		return false
	}
	if now.Sub(*k.UpstreamSyncedAt) > upstreamUsageTTL {
		return false
	}
	return k.UpstreamUsed >= k.UpstreamLimit
}

// Usable reports whether the credential may be handed to the upstream now.
//
// A key parked for quota comes back on its own after the month rolls over,
// because that is roughly when an upstream allowance resets; a key parked for
// a rejected credential stays out until someone fixes it, since retrying a
// wrong key only burns requests.
func (k *ProviderKey) Usable(now time.Time) bool {
	if !k.Enabled {
		return false
	}
	switch k.Status {
	case "", ProviderKeyActive:
		return true
	case ProviderKeyQuotaExceeded:
		// A freshly synced number beats the calendar guess. The month rolling
		// over means nothing to Brave, whose window is a rolling 30 days, and
		// nothing to a Tavily plan that renews on its own billing date.
		if k.UpstreamExhausted(now) {
			return false
		}
		return k.DisabledAt == nil || currentPeriod(*k.DisabledAt) != currentPeriod(now)
	default:
		return false
	}
}

// Recovered reports whether a parked key has become usable again by itself, so
// the caller can write the status back and let the UI stop showing it as dead.
func (k *ProviderKey) Recovered(now time.Time) bool {
	return k.Status == ProviderKeyQuotaExceeded && k.Usable(now)
}

// APIKey is a credential issued to a downstream agent. Zero means "unlimited"
// for both allowances, which is why these columns carry no database default.
type APIKey struct {
	model.BaseModel `gorm:"embedded"`
	Name            string `gorm:"column:name;not null" json:"name"`
	KeyPrefix       string `gorm:"column:key_prefix;uniqueIndex;not null" json:"keyPrefix"`
	KeyHash         string `gorm:"column:key_hash;not null" json:"-"`
	// KeyPlain lets the admin page hand the same key out again. Verification
	// still goes through KeyHash — this column is only read by an explicit
	// reveal action, never by the request path and never by the list endpoint.
	KeyPlain         string     `gorm:"column:key_plain" json:"-"`
	Enabled          bool       `gorm:"column:enabled" json:"enabled"`
	AllowedProviders string     `gorm:"column:allowed_providers" json:"allowedProviders"`
	RateLimitPerMin  int        `gorm:"column:rate_limit_per_min" json:"rateLimitPerMin"`
	MonthlyQuota     int        `gorm:"column:monthly_quota" json:"monthlyQuota"`
	ExpireAt         *time.Time `gorm:"column:expire_at" json:"expireAt"`
	LastUsedAt       *time.Time `gorm:"column:last_used_at" json:"lastUsedAt"`
	Note             string     `gorm:"column:note" json:"note"`
}

func (APIKey) TableName() string { return "ai_gateway_api_keys" }

// Allows reports whether this key may use the named provider. An empty
// AllowedProviders list means every registered provider is permitted.
func (k *APIKey) Allows(provider string) bool {
	allowed := k.AllowedList()
	if len(allowed) == 0 {
		return true
	}
	for _, name := range allowed {
		if name == provider {
			return true
		}
	}
	return false
}

// AllowedList splits the stored comma-separated provider whitelist.
func (k *APIKey) AllowedList() []string {
	raw := strings.Split(k.AllowedProviders, ",")
	allowed := make([]string, 0, len(raw))
	for _, name := range raw {
		if name = strings.TrimSpace(name); name != "" {
			allowed = append(allowed, name)
		}
	}
	return allowed
}

// Expired reports whether the key is past its expiry moment.
func (k *APIKey) Expired(now time.Time) bool {
	return k.ExpireAt != nil && !k.ExpireAt.After(now)
}

// Usable reports whether the key may authenticate a request right now.
func (k *APIKey) Usable(now time.Time) bool {
	return k.Enabled && !k.Expired(now)
}

// RequestLog is one gateway call, written asynchronously off the hot path.
type RequestLog struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreatedAt    time.Time `gorm:"column:created_at;index" json:"createdAt"`
	APIKeyID     int       `gorm:"column:api_key_id;index" json:"apiKeyId"`
	Provider     string    `gorm:"column:provider" json:"provider"`
	Endpoint     string    `gorm:"column:endpoint" json:"endpoint"`
	Query        string    `gorm:"column:query" json:"query"`
	ResultCount  int       `gorm:"column:result_count" json:"resultCount"`
	Cached       bool      `gorm:"column:cached" json:"cached"`
	FallbackFrom string    `gorm:"column:fallback_from" json:"fallbackFrom"`
	Status       string    `gorm:"column:status" json:"status"`
	HTTPStatus   int       `gorm:"column:http_status" json:"httpStatus"`
	LatencyMS    int       `gorm:"column:latency_ms" json:"latencyMs"`
	Credits      int       `gorm:"column:credits" json:"credits"`
	// CostMicroUSD is the request's price in millionths of a US dollar. Only
	// providers that bill by amount rather than by unit report it (Exa).
	CostMicroUSD int    `gorm:"column:cost_micro_usd" json:"costMicroUsd"`
	Error        string `gorm:"column:error" json:"error"`
	ClientIP     string `gorm:"column:client_ip" json:"clientIp"`
}

func (RequestLog) TableName() string { return "ai_gateway_request_logs" }

// Usage is a monthly counter. It exists so quota checks on the request path do
// not have to aggregate the request log table.
type Usage struct {
	ID           int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Subject      string `gorm:"column:subject;uniqueIndex:idx_gw_usage_subject_period;not null" json:"subject"`
	Period       string `gorm:"column:period;uniqueIndex:idx_gw_usage_subject_period;not null" json:"period"`
	Count        int    `gorm:"column:count" json:"count"`
	Credits      int    `gorm:"column:credits" json:"credits"`
	CostMicroUSD int    `gorm:"column:cost_micro_usd" json:"costMicroUsd"`
}

func (Usage) TableName() string { return "ai_gateway_usage" }

// Request log status values.
const (
	StatusOK                 = "ok"
	StatusProviderError      = "provider_error"
	StatusRateLimited        = "rate_limited"
	StatusQuotaExceeded      = "quota_exceeded"
	StatusInvalidRequest     = "invalid_request"
	StatusProviderNotAllowed = "provider_not_allowed"
	StatusProviderNotFound   = "provider_not_found"
	StatusNoProvider         = "no_provider_available"
)

// API key format. The plaintext is shown once at creation; only the prefix and
// a SHA-256 digest are stored.
const (
	apiKeyLiteral      = "gw_live_"
	apiKeyRandomLength = 32
	apiKeyPrefixDigits = 8
)

const apiKeyAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateAPIKey returns a new plaintext gateway key.
func GenerateAPIKey() (string, error) {
	buffer := make([]byte, apiKeyRandomLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成 API Key 失败: %w", err)
	}
	var builder strings.Builder
	builder.WriteString(apiKeyLiteral)
	for _, b := range buffer {
		builder.WriteByte(apiKeyAlphabet[int(b)%len(apiKeyAlphabet)])
	}
	return builder.String(), nil
}

// APIKeyPrefixOf returns the indexed lookup prefix of a plaintext key.
func APIKeyPrefixOf(plain string) string {
	limit := len(apiKeyLiteral) + apiKeyPrefixDigits
	if len(plain) < limit {
		return plain
	}
	return plain[:limit]
}

// HashAPIKey digests a plaintext key for storage and comparison. SHA-256 is
// deliberate: every gateway request verifies a key, and a password hash would
// dominate the request latency.
func HashAPIKey(plain string) string {
	digest := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(digest[:])
}

// MaskSecret renders a stored credential for the admin UI without exposing it.
func MaskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	switch {
	case secret == "":
		return ""
	case len(secret) <= 8:
		return "****"
	default:
		return secret[:4] + "****" + secret[len(secret)-4:]
	}
}

// Usage subject keys.
func providerSubject(name string) string { return "provider:" + name }
func keySubject(id int) string           { return fmt.Sprintf("key:%d", id) }

// providerKeySubject counts one upstream credential's own consumption. The
// provider-wide counter cannot answer "which account spent what", which is
// exactly the question a per-account allowance has to be measured against.
func providerKeySubject(id int) string { return fmt.Sprintf("providerkey:%d", id) }

// currentPeriod is the month bucket used by every quota counter.
func currentPeriod(now time.Time) string { return now.Format("2006-01") }

// RoutingStrategy decides how the gateway orders candidates when the caller
// asks for `provider: auto`. It never overrides capability, health or quota
// filtering — those are correctness constraints, not preferences; a strategy
// only chooses the order of what survives them.
type RoutingStrategy string

const (
	// StrategyBalanced spreads traffic by remaining monthly quota and weight,
	// ignoring Priority. Use it to make several free tiers last.
	StrategyBalanced RoutingStrategy = "balanced"
	// StrategyPriority follows the configured Priority strictly, using quota
	// only to exclude an exhausted provider. This is primary/standby: the
	// primary keeps serving as long as it is usable.
	StrategyPriority RoutingStrategy = "priority"
	// StrategyModel will ask a small model which provider suits the query.
	// Not implemented yet; the engine falls back to StrategyBalanced and the
	// admin page says so rather than pretending the routing changed.
	StrategyModel RoutingStrategy = "model"
)

// Valid reports whether the strategy is one the gateway knows.
func (s RoutingStrategy) Valid() bool {
	switch s {
	case StrategyBalanced, StrategyPriority, StrategyModel:
		return true
	default:
		return false
	}
}

// Implemented reports whether the strategy actually changes routing today.
func (s RoutingStrategy) Implemented() bool { return s != StrategyModel }

// Setting is a gateway-wide key/value option, mirroring the shape the system
// module uses for blog settings.
type Setting struct {
	ID           int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SettingKey   string `gorm:"column:setting_key;uniqueIndex;not null" json:"settingKey"`
	SettingValue string `gorm:"column:setting_value;not null" json:"settingValue"`
}

func (Setting) TableName() string { return "ai_gateway_settings" }

// SettingKeyRoutingStrategy stores the active RoutingStrategy.
const SettingKeyRoutingStrategy = "routing_strategy"

// MigrationModels declares the database tables owned by this module.
func MigrationModels() []any {
	return []any{&Provider{}, &ProviderKey{}, &APIKey{}, &RequestLog{}, &Usage{}, &Setting{}}
}

// Options inside a provider's Extra blob that the admin API manages by name
// instead of leaving to the free-form JSON, because one of them is a credential
// and must never be handed back in a list response.
const (
	extraKeyUsageServiceKey = "usage_service_key"
	extraKeyUsageKeyID      = "usage_key_id"
)

// mergeExtra sets or clears named options inside a provider's Extra JSON while
// leaving the options it does not know about untouched. A nil value means "no
// change"; an empty string removes the option.
func mergeExtra(extra string, values map[string]*string) (string, error) {
	fields := map[string]any{}
	if trimmed := strings.TrimSpace(extra); trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
			return "", fmt.Errorf("解析附加配置失败: %w", err)
		}
	}
	for key, value := range values {
		if value == nil {
			continue
		}
		if trimmed := strings.TrimSpace(*value); trimmed != "" {
			fields[key] = trimmed
		} else {
			delete(fields, key)
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// extraString reads one string option out of an Extra blob, returning "" when
// the blob is absent, malformed, or simply does not carry that option.
func extraString(extra, key string) string {
	trimmed := strings.TrimSpace(extra)
	if trimmed == "" {
		return ""
	}
	fields := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return ""
	}
	value, _ := fields[key].(string)
	return value
}

// redactExtra masks the credentials inside an Extra blob so the admin list can
// show the rest of a provider's options without handing back a secret.
func redactExtra(extra string) string {
	trimmed := strings.TrimSpace(extra)
	if trimmed == "" {
		return extra
	}
	fields := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return extra
	}
	secret, ok := fields[extraKeyUsageServiceKey].(string)
	if !ok || secret == "" {
		return extra
	}
	fields[extraKeyUsageServiceKey] = MaskSecret(secret)
	encoded, err := json.Marshal(fields)
	if err != nil {
		return extra
	}
	return string(encoded)
}
