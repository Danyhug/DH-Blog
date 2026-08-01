package search

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrLimiterBusy means the caller would have had to wait longer than it was
// willing to. The routing policy treats this as "this provider is saturated"
// and moves on to the next candidate.
var ErrLimiterBusy = errors.New("上游限速队列已满")

// Limiter is a token bucket that paces outbound calls to one provider.
//
// Brave's free tier allows a single request per second, so concurrent agents
// must queue on the bucket rather than fail: Wait reserves a token even when
// the bucket is empty and sleeps out the deficit, which serializes callers
// instead of letting them all collide with a 429.
type Limiter struct {
	mu       sync.Mutex
	rps      float64
	capacity float64
	tokens   float64
	last     time.Time

	// Injection points for tests.
	now   func() time.Time
	after func(time.Duration) <-chan time.Time
}

// NewLimiter builds a bucket allowing rps requests per second. A non-positive
// rate disables limiting.
func NewLimiter(rps float64) *Limiter {
	capacity := rps
	if capacity < 1 {
		capacity = 1
	}
	return &Limiter{
		rps:      rps,
		capacity: capacity,
		tokens:   capacity,
		now:      time.Now,
		after:    time.After,
	}
}

// Rate reports the configured requests per second.
func (l *Limiter) Rate() float64 { return l.rps }

// Wait blocks until a token is available, the context ends, or the required
// wait would exceed maxWait. When it returns nil a token has been consumed.
func (l *Limiter) Wait(ctx context.Context, maxWait time.Duration) error {
	if l == nil || l.rps <= 0 {
		return nil
	}

	delay, err := l.reserve(maxWait)
	if err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		l.release()
		return ctx.Err()
	case <-l.after(delay):
		return nil
	}
}

// reserve refills the bucket, takes a token and reports how long the caller
// must sleep before that token is actually earned.
func (l *Limiter) reserve(maxWait time.Duration) (time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.last.IsZero() {
		l.last = now
	}
	if elapsed := now.Sub(l.last); elapsed > 0 {
		l.tokens += elapsed.Seconds() * l.rps
		if l.tokens > l.capacity {
			l.tokens = l.capacity
		}
		l.last = now
	}

	var delay time.Duration
	if l.tokens < 1 {
		deficit := 1 - l.tokens
		delay = time.Duration(deficit / l.rps * float64(time.Second))
		if delay > maxWait {
			return 0, ErrLimiterBusy
		}
	}
	l.tokens--
	return delay, nil
}

// release returns a reserved token after the caller gave up waiting.
func (l *Limiter) release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens++
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
}
