package search

import (
	"sync"
	"time"
)

// BreakerState is the current position in the circuit breaker cycle.
type BreakerState string

const (
	BreakerClosed   BreakerState = "closed"
	BreakerOpen     BreakerState = "open"
	BreakerHalfOpen BreakerState = "half_open"
)

// Breaker defaults, matching the gateway design.
const (
	defaultBreakerWindow      = time.Minute
	defaultBreakerOpenFor     = 30 * time.Second
	defaultConsecutiveTrips   = 5
	defaultBreakerMinSamples  = 10
	defaultBreakerFailureRate = 0.5
)

type breakerEvent struct {
	at      time.Time
	failure bool
}

// Breaker keeps a provider that is failing out of the routing pool for a
// cooldown, then lets a single probe through before fully reopening.
type Breaker struct {
	mu sync.Mutex

	window       time.Duration
	openFor      time.Duration
	consecutive  int
	minSamples   int
	failureRatio float64

	state               BreakerState
	events              []breakerEvent
	consecutiveFailures int
	openedAt            time.Time
	probeInFlight       bool

	now func() time.Time
}

// NewBreaker builds a breaker with the gateway's default thresholds.
func NewBreaker() *Breaker {
	return &Breaker{
		window:       defaultBreakerWindow,
		openFor:      defaultBreakerOpenFor,
		consecutive:  defaultConsecutiveTrips,
		minSamples:   defaultBreakerMinSamples,
		failureRatio: defaultBreakerFailureRate,
		state:        BreakerClosed,
		now:          time.Now,
	}
}

// State reports the current state, applying any pending open-to-half-open
// transition so callers see an accurate value.
func (b *Breaker) State() BreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.promote()
	return b.state
}

// Allow reports whether a call may proceed. In the half-open state exactly one
// probe is admitted until it reports back.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.promote()

	switch b.state {
	case BreakerOpen:
		return false
	case BreakerHalfOpen:
		if b.probeInFlight {
			return false
		}
		b.probeInFlight = true
		return true
	default:
		return true
	}
}

// Release cancels a call that Allow admitted but that was never actually sent
// upstream. Reporting it as a success would be worse than saying nothing: a
// phantom success resets the consecutive-failure count and can keep a genuinely
// broken provider out of the open state indefinitely.
func (b *Breaker) Release() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == BreakerHalfOpen {
		b.probeInFlight = false
	}
}

// Report records the outcome of a call admitted by Allow.
func (b *Breaker) Report(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()

	if b.state == BreakerHalfOpen {
		b.probeInFlight = false
		if success {
			b.reset()
		} else {
			b.trip(now)
		}
		return
	}

	b.events = append(b.events, breakerEvent{at: now, failure: !success})
	b.pruneLocked(now)

	if success {
		b.consecutiveFailures = 0
		return
	}
	b.consecutiveFailures++

	if b.consecutiveFailures >= b.consecutive {
		b.trip(now)
		return
	}
	failures := 0
	for _, event := range b.events {
		if event.failure {
			failures++
		}
	}
	if len(b.events) >= b.minSamples && float64(failures)/float64(len(b.events)) > b.failureRatio {
		b.trip(now)
	}
}

// promote moves an expired open breaker into the half-open state.
func (b *Breaker) promote() {
	if b.state == BreakerOpen && b.now().Sub(b.openedAt) >= b.openFor {
		b.state = BreakerHalfOpen
		b.probeInFlight = false
	}
}

func (b *Breaker) trip(now time.Time) {
	b.state = BreakerOpen
	b.openedAt = now
	b.events = nil
	b.consecutiveFailures = 0
	b.probeInFlight = false
}

func (b *Breaker) reset() {
	b.state = BreakerClosed
	b.events = nil
	b.consecutiveFailures = 0
	b.probeInFlight = false
}

func (b *Breaker) pruneLocked(now time.Time) {
	cutoff := now.Add(-b.window)
	kept := b.events[:0]
	for _, event := range b.events {
		if event.at.After(cutoff) {
			kept = append(kept, event)
		}
	}
	b.events = kept
}
