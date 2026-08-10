package search

import (
	"context"
	"errors"
)

// Units an upstream can count consumption in. They are not interchangeable: one
// Tavily advanced search costs two credits, while Brave charges one request no
// matter what the search asked for. Storing the unit alongside the number keeps
// the gateway from presenting the two as if they were the same measure.
const (
	UsageUnitCredit  = "credit"
	UsageUnitRequest = "request"
	// UsageUnitMicroUSD is money, in millionths of a US dollar. Exa bills
	// against a prepaid balance rather than an allowance of calls, so what it
	// reports is spend and only a currency can express it.
	UsageUnitMicroUSD = "micro_usd"
)

// Scopes a UsageReport can describe. The distinction matters when several
// credentials sit on one plan: a per-key ceiling stops that key alone, an
// account ceiling stops every key at once, so rotating to a sibling key only
// helps in the first case.
const (
	UsageScopeKey     = "key"
	UsageScopeAccount = "account"
)

// UsageReport is what an upstream says has been consumed under the credential
// the adapter holds. It is the provider's own accounting, not ours — the point
// of asking is that the gateway's local counter only sees traffic that went
// through the gateway.
type UsageReport struct {
	// Used and Limit are in Unit. Limit is 0 when the upstream reports no
	// ceiling, which must not be read as "nothing left".
	Used  int
	Limit int
	Unit  string
	Scope string
	// Window names the period the numbers cover, because providers do not agree
	// on one: Tavily resets on the subscription's billing cycle, Brave counts a
	// rolling 30 days. Neither is the calendar month the gateway counts in.
	Window string
}

// Exhausted reports whether the upstream's own numbers leave nothing to spend.
func (r UsageReport) Exhausted() bool { return r.Limit > 0 && r.Used >= r.Limit }

// ErrUsageUnavailable means the adapter has nothing to report yet rather than
// that something went wrong. Brave publishes its quota only in the headers of
// real traffic, so a fresh process genuinely does not know where it stands.
var ErrUsageUnavailable = errors.New("上游用量暂不可用")

// UsageReporter is implemented by adapters that can state their own
// consumption. Not every provider can, and none of them are made to pretend:
// Exa only reports through a separate team-management credential, so its
// adapter reports nothing until that credential is configured and the gateway
// keeps counting locally in the meantime.
type UsageReporter interface {
	Provider
	Usage(ctx context.Context) (UsageReport, error)
}
