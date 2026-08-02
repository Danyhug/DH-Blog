package aigateway

import (
	"errors"
	"sort"
	"strings"

	"dh-blog/internal/platform/search"
)

// providerAuto asks the gateway to choose the upstream itself.
const providerAuto = "auto"

// Weights of the two scoring terms. Remaining quota dominates so a free tier
// is spent evenly rather than exhausted on whichever provider ranks first.
const (
	quotaScoreWeight  = 0.7
	weightScoreWeight = 0.3
)

var (
	// ErrProviderNotFound means the caller named a provider the gateway does
	// not have registered and enabled.
	ErrProviderNotFound = errors.New("指定的搜索供应商不存在或未启用")
	// ErrProviderNotAllowed means the key's whitelist excludes the provider.
	ErrProviderNotAllowed = errors.New("当前 API Key 无权使用该搜索供应商")
	// ErrNoProviderAvailable means every candidate was filtered out.
	ErrNoProviderAvailable = errors.New("没有可用的搜索供应商")
	// ErrProviderKeyNotFound means the referenced upstream credential is gone.
	ErrProviderKeyNotFound = errors.New("指定的供应商密钥不存在")
)

// candidate is a provider as the routing policy sees it: configuration plus
// live health and quota state, with no dependency on the database or HTTP.
type candidate struct {
	Name         string
	Capability   search.Capability
	Priority     int
	Weight       int
	MonthlyQuota int
	Used         int
	// MonthlyCostLimit and CostUsed are the same idea in money, both in
	// millionths of a US dollar. A provider can be capped by either, whichever
	// runs out first: counting calls says nothing about a pay-as-you-go plan,
	// and counting dollars says nothing about a plan sold in requests.
	MonthlyCostLimit int
	CostUsed         int
	Healthy          bool
}

// routeInput is everything the policy needs to order the candidates.
type routeInput struct {
	Requested        string // "", "auto", or an explicit provider name
	Strategy         RoutingStrategy
	AllowFallback    bool
	Allowed          []string // key whitelist; empty means every provider
	NeedAnswer       bool
	NeedRawContent   bool
	PrefersOperators bool
	Candidates       []candidate
}

// route returns the providers to try, best first. The caller attempts them in
// order, stopping at the first success or at a non-retryable failure.
func route(in routeInput) ([]string, error) {
	if len(in.Candidates) == 0 {
		return nil, ErrNoProviderAvailable
	}

	requested := strings.ToLower(strings.TrimSpace(in.Requested))
	if requested != "" && requested != providerAuto {
		found := findCandidate(in.Candidates, requested)
		if found == nil {
			return nil, ErrProviderNotFound
		}
		if !allowedBy(in.Allowed, requested) {
			return nil, ErrProviderNotAllowed
		}
		// An explicit choice overrides capability matching: the caller asked
		// for this provider and gets whatever it can express.
		if usable(*found) {
			order := []string{requested}
			if in.AllowFallback {
				order = append(order, autoOrder(in, requested)...)
			}
			return order, nil
		}
		if !in.AllowFallback {
			return nil, ErrNoProviderAvailable
		}
	}

	order := autoOrder(in, "")
	if len(order) == 0 {
		return nil, ErrNoProviderAvailable
	}
	return order, nil
}

// autoOrder applies the whitelist, capability, health and quota filters, then
// ranks what is left according to the configured strategy.
func autoOrder(in routeInput, exclude string) []string {
	pool := make([]candidate, 0, len(in.Candidates))
	for _, item := range in.Candidates {
		if item.Name == exclude || !allowedBy(in.Allowed, item.Name) || !usable(item) {
			continue
		}
		pool = append(pool, item)
	}

	pool = filterCapability(pool, in)
	if len(pool) == 0 {
		return nil
	}

	sortByStrategy(pool, in.Strategy)

	order := make([]string, 0, len(pool))
	for _, item := range pool {
		order = append(order, item.Name)
	}
	return order
}

// sortByStrategy orders the surviving candidates. Every comparison ends in a
// name tiebreak so routing stays deterministic for identical configurations.
func sortByStrategy(pool []candidate, strategy RoutingStrategy) {
	switch strategy {
	case StrategyPriority:
		// Primary/standby: the configured order is the whole answer, and
		// remaining quota only matters for the exclusion done earlier.
		sort.SliceStable(pool, func(i, j int) bool {
			if pool[i].Priority != pool[j].Priority {
				return pool[i].Priority < pool[j].Priority
			}
			if pool[i].Weight != pool[j].Weight {
				return pool[i].Weight > pool[j].Weight
			}
			return pool[i].Name < pool[j].Name
		})
	default:
		// Balanced, and the not-yet-built model strategy, which falls back
		// here rather than inventing an ordering it cannot justify.
		maxWeight := 0
		for _, item := range pool {
			if item.Weight > maxWeight {
				maxWeight = item.Weight
			}
		}
		sort.SliceStable(pool, func(i, j int) bool {
			left, right := score(pool[i], maxWeight), score(pool[j], maxWeight)
			if left != right {
				return left > right
			}
			return pool[i].Name < pool[j].Name
		})
	}
}

// filterCapability drops providers that cannot express what the caller asked
// for. Answer and raw content are hard requirements because silently omitting
// them would look like an upstream with nothing to say; operator support is a
// preference, since a provider without it still returns sane results.
func filterCapability(pool []candidate, in routeInput) []candidate {
	if in.NeedAnswer {
		pool = keep(pool, func(item candidate) bool { return item.Capability.Answer })
	}
	if in.NeedRawContent {
		pool = keep(pool, func(item candidate) bool { return item.Capability.RawContent })
	}
	if in.PrefersOperators {
		if preferred := keep(pool, func(item candidate) bool { return item.Capability.SearchOperators }); len(preferred) > 0 {
			pool = preferred
		}
	}
	return pool
}

func keep(pool []candidate, predicate func(candidate) bool) []candidate {
	kept := make([]candidate, 0, len(pool))
	for _, item := range pool {
		if predicate(item) {
			kept = append(kept, item)
		}
	}
	return kept
}

// score blends remaining monthly allowance with the configured weight. It is
// the balanced strategy's ranking function; Priority plays no part, because
// expressing a fixed order is what the priority strategy is for.
//
// A provider capped both ways is scored on whichever allowance is tighter — the
// one that will stop it first is the one that describes how much room is left.
func score(item candidate, maxWeight int) float64 {
	quotaRatio := remainingRatio(item.Used, item.MonthlyQuota)
	if costRatio := remainingRatio(item.CostUsed, item.MonthlyCostLimit); costRatio < quotaRatio {
		quotaRatio = costRatio
	}
	normalizedWeight := 1.0
	if maxWeight > 0 {
		normalizedWeight = float64(item.Weight) / float64(maxWeight)
	}
	return quotaRatio*quotaScoreWeight + normalizedWeight*weightScoreWeight
}

// remainingRatio is the share of an allowance still unspent, and 1 when there
// is no ceiling — an uncapped provider is never the one to avoid.
func remainingRatio(used, limit int) float64 {
	if limit <= 0 {
		return 1
	}
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	return float64(remaining) / float64(limit)
}

// usable reports whether a candidate is currently eligible at all.
func usable(item candidate) bool {
	return item.Healthy && !quotaExhausted(item)
}

func quotaExhausted(item candidate) bool {
	if item.MonthlyQuota > 0 && item.Used >= item.MonthlyQuota {
		return true
	}
	return item.MonthlyCostLimit > 0 && item.CostUsed >= item.MonthlyCostLimit
}

func findCandidate(candidates []candidate, name string) *candidate {
	for index := range candidates {
		if candidates[index].Name == name {
			return &candidates[index]
		}
	}
	return nil
}

func allowedBy(allowed []string, name string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if item == name {
			return true
		}
	}
	return false
}

// queryUsesOperators detects the search syntax only Brave honours, so a query
// written with operators is steered towards a provider that understands them.
func queryUsesOperators(query string) bool {
	lower := strings.ToLower(query)
	for _, operator := range []string{"site:", "filetype:", "inurl:", "intitle:", "ext:"} {
		if strings.Contains(lower, operator) {
			return true
		}
	}
	return strings.Count(query, `"`) >= 2
}
