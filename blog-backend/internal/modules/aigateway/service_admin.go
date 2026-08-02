package aigateway

import (
	"context"
	"strings"
	"time"

	"dh-blog/internal/platform/search"
)

// searchProbe is the fixed query used by the connectivity test button.
func searchProbe() search.Request {
	return search.Request{Query: "hello world", MaxResults: 1}
}

// ProviderProbe describes which credential a connectivity test should use.
// Every field is optional; the draft ones deliberately never touch the database.
type ProviderProbe struct {
	KeyID   int    `json:"keyId"`
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseUrl"`
	Extra   string `json:"extra"`
}

// ProbeResult is what the connectivity test reports back.
type ProbeResult struct {
	OK          bool   `json:"ok"`
	LatencyMS   int    `json:"latencyMs"`
	ResultCount int    `json:"resultCount"`
	Error       string `json:"error,omitempty"`
	KeyLabel    string `json:"keyLabel,omitempty"`
}

// TestProvider issues one real search so an operator can confirm a credential.
//
// The draft fields are the point: a key has to be testable *before* it is
// saved, otherwise the only way to discover a bad key is to store it and wait
// for real traffic to fail. Nothing here is persisted, and the probe is built
// from a throwaway adapter so the live rotation is untouched.
func (s *Service) TestProvider(ctx context.Context, name string, probe ProviderProbe) (ProbeResult, error) {
	config, err := s.repo.providerByName(ctx, name)
	if err != nil {
		return ProbeResult{}, err
	}
	if draft := strings.TrimSpace(probe.BaseURL); draft != "" {
		config.BaseURL = draft
	}
	if draft := strings.TrimSpace(probe.Extra); draft != "" {
		config.Extra = draft
	}

	apiKey := strings.TrimSpace(probe.APIKey)
	label := "未保存的密钥"
	switch {
	case apiKey != "":
		// 用草稿里的密钥，正是"保存前先测"的场景
	case probe.KeyID > 0:
		stored, err := s.repo.providerKeyByID(ctx, probe.KeyID)
		if err != nil {
			return ProbeResult{}, err
		}
		apiKey, label = stored.APIKey, stored.Label
	default:
		runtime := s.runtime(name)
		if runtime == nil {
			return ProbeResult{}, ErrProviderNotFound
		}
		picked := runtime.firstUsableKey(s.now())
		if picked == nil {
			return ProbeResult{}, ErrProviderKeyNotFound
		}
		apiKey, label = picked.config.APIKey, picked.config.Label
	}
	if strings.TrimSpace(apiKey) == "" {
		return ProbeResult{}, ErrProviderKeyNotFound
	}

	adapter, err := s.buildAdapter(config, apiKey)
	if err != nil {
		return ProbeResult{}, err
	}

	started := time.Now()
	response, callErr := adapter.Search(ctx, searchProbe())
	latency := int(time.Since(started) / time.Millisecond)
	if callErr != nil {
		return ProbeResult{OK: false, LatencyMS: latency, Error: callErr.Error(), KeyLabel: label}, nil
	}
	return ProbeResult{OK: true, LatencyMS: latency, ResultCount: len(response.Results), KeyLabel: label}, nil
}

// providerViews assembles the admin provider list, masking every credential.
func (s *Service) providerViews(ctx context.Context) ([]providerView, error) {
	providers, err := s.repo.listProviders(ctx)
	if err != nil {
		return nil, err
	}
	credentials, err := s.repo.listProviderKeys(ctx)
	if err != nil {
		return nil, err
	}

	subjects := make([]string, 0, len(providers))
	for _, provider := range providers {
		subjects = append(subjects, providerSubject(provider.Name))
	}
	usage, err := s.repo.usageFor(ctx, currentPeriod(s.now()), subjects)
	if err != nil {
		return nil, err
	}

	now := s.now()
	keysByProvider := make(map[string][]providerKeyView, len(providers))
	activeByProvider := make(map[string]int, len(providers))
	for _, credential := range credentials {
		rotating := credential.Usable(now)
		if rotating {
			activeByProvider[credential.Provider]++
		}
		status := credential.Status
		if status == "" {
			status = ProviderKeyActive
		}
		// 跨月自愈的密钥在页面上就该显示为正常，别让人以为还停着
		if credential.Recovered(now) {
			status = ProviderKeyActive
		}
		keysByProvider[credential.Provider] = append(keysByProvider[credential.Provider], providerKeyView{
			ID:               credential.ID,
			Label:            credential.Label,
			Masked:           MaskSecret(credential.APIKey),
			Enabled:          credential.Enabled,
			Status:           status,
			LastError:        credential.LastError,
			LastUsedAt:       credential.LastUsedAt,
			DisabledAt:       credential.DisabledAt,
			InRotation:       rotating,
			UpstreamUsed:     credential.UpstreamUsed,
			UpstreamLimit:    credential.UpstreamLimit,
			UpstreamUnit:     credential.UpstreamUnit,
			UpstreamScope:    credential.UpstreamScope,
			UpstreamWindow:   credential.UpstreamWindow,
			UpstreamSyncedAt: credential.UpstreamSyncedAt,
			UpstreamError:    credential.UpstreamError,
		})
	}

	views := make([]providerView, 0, len(providers))
	for _, provider := range providers {
		health := string(search.BreakerClosed)
		reportsUsage := false
		if runtime := s.runtime(provider.Name); runtime != nil {
			health = string(runtime.breaker.State())
			reportsUsage = runtime.reportsUsage
		}
		meta := search.MetaFor(provider.Name)
		keys := keysByProvider[provider.Name]
		if keys == nil {
			keys = []providerKeyView{}
		}
		views = append(views, providerView{
			Name:              provider.Name,
			DisplayName:       provider.DisplayName,
			HomeURL:           meta.HomeURL,
			DocsURL:           meta.DocsURL,
			ConsoleURL:        meta.ConsoleURL,
			LogoURL:           meta.LogoURL,
			Billing:           meta.Billing,
			Enabled:           provider.Enabled,
			Keys:              keys,
			ActiveKeys:        activeByProvider[provider.Name],
			BaseURL:           provider.BaseURL,
			Priority:          provider.Priority,
			Weight:            provider.Weight,
			RPS:               provider.RPS,
			MonthlyQuota:      provider.MonthlyQuota,
			MonthlyUsed:       usage[providerSubject(provider.Name)].Count,
			MonthlyCost:       usage[providerSubject(provider.Name)].CostMicroUSD,
			MonthlyCostLimit:  provider.MonthlyCostLimit,
			SupportsUsageSync: reportsUsage,
			Extra:             provider.Extra,
			Health:            health,
		})
	}
	return views, nil
}

// apiKeyViews assembles the admin key list. The plaintext was returned once at
// creation and cannot be recovered from here.
func (s *Service) apiKeyViews(ctx context.Context) ([]apiKeyView, error) {
	keys, err := s.repo.listAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	subjects := make([]string, 0, len(keys))
	for _, key := range keys {
		subjects = append(subjects, keySubject(key.ID))
	}
	usage, err := s.repo.usageFor(ctx, currentPeriod(s.now()), subjects)
	if err != nil {
		return nil, err
	}

	views := make([]apiKeyView, 0, len(keys))
	for _, key := range keys {
		views = append(views, apiKeyView{
			ID:               key.ID,
			Name:             key.Name,
			KeyPrefix:        key.KeyPrefix,
			Enabled:          key.Enabled,
			AllowedProviders: key.AllowedProviders,
			RateLimitPerMin:  key.RateLimitPerMin,
			MonthlyQuota:     key.MonthlyQuota,
			MonthlyUsed:      usage[keySubject(key.ID)].Count,
			ExpireAt:         key.ExpireAt,
			LastUsedAt:       key.LastUsedAt,
			Note:             key.Note,
		})
	}
	return views, nil
}

// updateAPIKey applies an admin patch and drops the cached credential so a
// revocation or quota change takes effect on the next request.
func (s *Service) updateAPIKey(ctx context.Context, id int, updates map[string]any) error {
	key, err := s.repo.apiKeyByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.updateAPIKey(ctx, id, updates); err != nil {
		return err
	}
	s.invalidateAPIKey(key.KeyPrefix)
	return nil
}

func (s *Service) deleteAPIKey(ctx context.Context, id int) error {
	key, err := s.repo.apiKeyByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.deleteAPIKey(ctx, id); err != nil {
		return err
	}
	s.invalidateAPIKey(key.KeyPrefix)
	return nil
}

// QuotaRow is one provider's monthly consumption against its ceiling. Both
// ceilings appear because a provider may be capped by calls, by spend, or
// neither — and the dashboard should show whichever one is real.
type QuotaRow struct {
	Provider         string `json:"provider"`
	MonthlyQuota     int    `json:"monthlyQuota"`
	MonthlyUsed      int    `json:"monthlyUsed"`
	MonthlyCost      int    `json:"monthlyCostMicroUsd"`
	MonthlyCostLimit int    `json:"monthlyCostLimitMicroUsd"`
}

// StatsSummary backs the admin dashboard.
type StatsSummary struct {
	Days         int        `json:"days"`
	Total        int64      `json:"total"`
	Succeeded    int64      `json:"succeeded"`
	Cached       int64      `json:"cached"`
	Credits      int64      `json:"credits"`
	CostMicroUSD int64      `json:"costMicroUsd"`
	Providers    []statsRow `json:"providers"`
	Quotas       []QuotaRow `json:"quotas"`
}

func (s *Service) stats(ctx context.Context, days int) (StatsSummary, error) {
	since := s.now().AddDate(0, 0, -days)
	rows, err := s.repo.statsSince(ctx, since)
	if err != nil {
		return StatsSummary{}, err
	}

	summary := StatsSummary{Days: days, Providers: rows}
	for _, row := range rows {
		summary.Total += row.Total
		summary.Succeeded += row.Succeeded
		summary.Cached += row.Cached
		summary.Credits += row.Credits
		summary.CostMicroUSD += row.CostMicroUSD
	}

	providers, err := s.repo.listProviders(ctx)
	if err != nil {
		return StatsSummary{}, err
	}
	subjects := make([]string, 0, len(providers))
	for _, provider := range providers {
		subjects = append(subjects, providerSubject(provider.Name))
	}
	usage, err := s.repo.usageFor(ctx, currentPeriod(s.now()), subjects)
	if err != nil {
		return StatsSummary{}, err
	}
	for _, provider := range providers {
		summary.Quotas = append(summary.Quotas, QuotaRow{
			Provider:         provider.Name,
			MonthlyQuota:     provider.MonthlyQuota,
			MonthlyUsed:      usage[providerSubject(provider.Name)].Count,
			MonthlyCost:      usage[providerSubject(provider.Name)].CostMicroUSD,
			MonthlyCostLimit: provider.MonthlyCostLimit,
		})
	}
	return summary, nil
}

// pruneLogs drops request logs past the configured retention.
func (s *Service) pruneLogs(ctx context.Context) (int64, error) {
	days := s.options.LogRetentionDays
	if days <= 0 {
		return 0, nil
	}
	return s.repo.deleteLogsBefore(ctx, s.now().AddDate(0, 0, -days))
}
