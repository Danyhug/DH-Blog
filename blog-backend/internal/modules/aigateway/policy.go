package aigateway

import (
	"errors"
	"sort"
	"strings"

	"dh-blog/internal/platform/search"
)

// providerAuto asks the gateway to choose the upstream itself.
const providerAuto = "auto"

// Weights of the two scoring terms. Remaining allowance dominates so a provider
// approaching its ceiling steps aside; the balance term then spreads whatever
// traffic is left over the providers that still have room.
const (
	quotaScoreWeight   = 0.7
	balanceScoreWeight = 0.3
	// operatorBonus moves a provider that honours search operators ahead of one
	// that does not when the query uses them. Sized to outrank a moderate
	// difference in remaining allowance but not an account that is nearly spent:
	// a slightly worse result set is a better outcome than an exhausted upstream.
	operatorBonus = 0.1
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
	// UpstreamHeadroom is the share of the upstreams' own allowance still
	// unspent, set only when a recent report said so. It is a ratio rather than
	// a count because the plans it summarises are sold in different units —
	// credits, requests, dollars — and only a ratio compares across them.
	//
	// It exists because a locally uncapped provider is not the same thing as an
	// unlimited one. Leaving the local ceiling blank is how an operator says "I
	// do not know what the allowance is", and reading that as "there is no
	// allowance" is what let an account the upstream had already reported empty
	// outscore every provider that did declare a ceiling.
	UpstreamHeadroom    float64
	HasUpstreamHeadroom bool
	Healthy             bool
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

	sortByStrategy(pool, in.Strategy, in.PrefersOperators)

	order := make([]string, 0, len(pool))
	for _, item := range pool {
		order = append(order, item.Name)
	}
	return order
}

// sortByStrategy orders the surviving candidates. Every comparison ends in a
// name tiebreak so routing stays deterministic for identical configurations.
func sortByStrategy(pool []candidate, strategy RoutingStrategy, prefersOperators bool) {
	switch strategy {
	case StrategyPriority:
		// Primary/standby: the configured order is the whole answer, and
		// remaining quota only matters for the exclusion done earlier. An
		// operator preference does not get to reorder an explicit ranking.
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
		totals := balanceTotalsOf(pool)
		sort.SliceStable(pool, func(i, j int) bool {
			left := score(pool[i], totals, prefersOperators)
			right := score(pool[j], totals, prefersOperators)
			if left != right {
				return left > right
			}
			// An untouched pool scores everyone alike, so weight is what says
			// who should be picked first; the name keeps it deterministic.
			if pool[i].Weight != pool[j].Weight {
				return pool[i].Weight > pool[j].Weight
			}
			return pool[i].Name < pool[j].Name
		})
	}
}

// filterCapability drops providers that cannot express what the caller asked
// for. Answer and raw content are hard requirements because silently omitting
// them would look like an upstream with nothing to say.
//
// Operator support is deliberately not filtered on. It is a preference: a
// provider that ignores site: still answers the query sensibly, whereas keeping
// only the providers that honour it hands every quoted query to whichever single
// upstream has the feature, no matter how little that account has left. It is
// scored instead, in score() below.
func filterCapability(pool []candidate, in routeInput) []candidate {
	if in.NeedAnswer {
		pool = keep(pool, func(item candidate) bool { return item.Capability.Answer })
	}
	if in.NeedRawContent {
		pool = keep(pool, func(item candidate) bool { return item.Capability.RawContent })
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

// balanceTotals carries the pool-wide figure the balanced strategy needs to
// turn one candidate's load into a comparable [0,1] number.
type balanceTotals struct{ maxLoad float64 }

func balanceTotalsOf(pool []candidate) balanceTotals {
	var totals balanceTotals
	for _, item := range pool {
		if load := loadFactor(item); load > totals.maxLoad {
			totals.maxLoad = load
		}
	}
	return totals
}

// loadFactor is calls already served per unit of weight — how much of its share
// a provider has taken. A weight-5 provider has to serve five times the calls of
// a weight-1 one before the two count as equally loaded, which is what makes
// weight express a share of traffic rather than a fixed rank.
func loadFactor(item candidate) float64 {
	weight := item.Weight
	if weight <= 0 {
		weight = 1
	}
	return float64(item.Used) / float64(weight)
}

// score blends how much allowance a provider has left with how far it is behind
// its share of traffic, plus a nudge when the query wants search operators.
//
// The balance term is what makes this strategy balance at all. Ranking on
// remaining allowance alone cannot: a provider with no ceiling sits at a
// permanent 1.0 and strictly dominates one that has a ceiling, so a capped
// provider loses every comparison from its first unit of spend onward. Exa was
// exactly that case — the only upstream that reports what a call cost, and so
// the only one whose score ever moved, always downward.
//
// Priority plays no part here; expressing a fixed order is what the priority
// strategy is for.
func score(item candidate, totals balanceTotals, prefersOperators bool) float64 {
	total := headroomRatio(item)*quotaScoreWeight + idleRatio(item, totals)*balanceScoreWeight
	if prefersOperators && item.Capability.SearchOperators {
		total += operatorBonus
	}
	return total
}

// headroomRatio is the share of the tightest allowance still unspent. A provider
// capped several ways is judged on whichever ceiling stops it first: counting
// calls says nothing about a pay-as-you-go plan, counting dollars says nothing
// about a plan sold in requests, and neither says anything about an upstream
// that has been reporting an empty balance since yesterday.
func headroomRatio(item candidate) float64 {
	ratio := remainingRatio(item.Used, item.MonthlyQuota)
	if costRatio := remainingRatio(item.CostUsed, item.MonthlyCostLimit); costRatio < ratio {
		ratio = costRatio
	}
	if item.HasUpstreamHeadroom && item.UpstreamHeadroom < ratio {
		ratio = item.UpstreamHeadroom
	}
	return ratio
}

// idleRatio is 1 for the least loaded provider in the pool and 0 for the most
// loaded, so traffic keeps moving towards whoever is furthest behind its weight.
// An untouched pool scores everyone 1 and leaves the ordering to weight.
func idleRatio(item candidate, totals balanceTotals) float64 {
	if totals.maxLoad <= 0 {
		return 1
	}
	return 1 - loadFactor(item)/totals.maxLoad
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
	if item.MonthlyCostLimit > 0 && item.CostUsed >= item.MonthlyCostLimit {
		return true
	}
	// The upstream saying its allowance is gone is the last word, even if the
	// credential that reported it has not been parked yet.
	return item.HasUpstreamHeadroom && item.UpstreamHeadroom <= 0
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
