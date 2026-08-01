package aigateway

import (
	"context"

	"dh-blog/internal/platform/search"
)

// searchProbe is the fixed query used by the connectivity test button.
func searchProbe() search.Request {
	return search.Request{Query: "hello world", MaxResults: 1}
}

// providerViews assembles the admin provider list, masking every credential.
func (s *Service) providerViews(ctx context.Context) ([]providerView, error) {
	providers, err := s.repo.listProviders(ctx)
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

	views := make([]providerView, 0, len(providers))
	for _, provider := range providers {
		health := string(search.BreakerClosed)
		if runtime := s.runtime(provider.Name); runtime != nil {
			health = string(runtime.breaker.State())
		}
		meta := search.MetaFor(provider.Name)
		views = append(views, providerView{
			Name:          provider.Name,
			DisplayName:   provider.DisplayName,
			HomeURL:       meta.HomeURL,
			DocsURL:       meta.DocsURL,
			ConsoleURL:    meta.ConsoleURL,
			LogoURL:       meta.LogoURL,
			Billing:       meta.Billing,
			Enabled:       provider.Enabled,
			APIKeyMasked:  MaskSecret(provider.APIKey),
			APIKeyPresent: provider.APIKey != "",
			BaseURL:       provider.BaseURL,
			Priority:      provider.Priority,
			Weight:        provider.Weight,
			RPS:           provider.RPS,
			MonthlyQuota:  provider.MonthlyQuota,
			MonthlyUsed:   usage[providerSubject(provider.Name)].Count,
			MonthlyCost:   usage[providerSubject(provider.Name)].CostMicroUSD,
			Extra:         provider.Extra,
			Health:        health,
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

// QuotaRow is one provider's monthly consumption against its ceiling.
type QuotaRow struct {
	Provider     string `json:"provider"`
	MonthlyQuota int    `json:"monthlyQuota"`
	MonthlyUsed  int    `json:"monthlyUsed"`
	MonthlyCost  int    `json:"monthlyCostMicroUsd"`
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
			Provider:     provider.Name,
			MonthlyQuota: provider.MonthlyQuota,
			MonthlyUsed:  usage[providerSubject(provider.Name)].Count,
			MonthlyCost:  usage[providerSubject(provider.Name)].CostMicroUSD,
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
