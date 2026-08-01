package search

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeClock drives the limiter and breaker deterministically.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	pending []chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	fired := c.pending
	c.pending = nil
	c.mu.Unlock()
	for _, ch := range fired {
		ch <- c.Now()
	}
}

// After records the timer so Advance can release every waiter at once. The
// limiter's own deficit maths decides the delay; the test only controls when
// that delay is considered elapsed.
func (c *fakeClock) After(time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	c.mu.Lock()
	c.pending = append(c.pending, ch)
	c.mu.Unlock()
	return ch
}

func newTestLimiter(rps float64, clock *fakeClock) *Limiter {
	limiter := NewLimiter(rps)
	limiter.now = clock.Now
	limiter.after = clock.After
	return limiter
}

func TestLimiterAllowsBurstUpToCapacity(t *testing.T) {
	clock := newFakeClock()
	limiter := newTestLimiter(1, clock)

	if err := limiter.Wait(context.Background(), time.Second); err != nil {
		t.Fatalf("首个请求应立即通过: %v", err)
	}
}

func TestLimiterRejectsWhenWaitExceedsBudget(t *testing.T) {
	clock := newFakeClock()
	limiter := newTestLimiter(1, clock)

	// Consume the single token Brave's free tier allows per second.
	if err := limiter.Wait(context.Background(), time.Second); err != nil {
		t.Fatalf("首个请求应立即通过: %v", err)
	}
	// The next token is a full second away, more than this caller will wait.
	err := limiter.Wait(context.Background(), 100*time.Millisecond)
	if !errors.Is(err, ErrLimiterBusy) {
		t.Fatalf("期望 ErrLimiterBusy, 得到 %v", err)
	}
}

func TestLimiterQueuesInsteadOfFailing(t *testing.T) {
	clock := newFakeClock()
	limiter := newTestLimiter(1, clock)

	if err := limiter.Wait(context.Background(), 2*time.Second); err != nil {
		t.Fatalf("首个请求应立即通过: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- limiter.Wait(context.Background(), 2*time.Second) }()

	// The second caller must be parked, not rejected.
	select {
	case err := <-done:
		t.Fatalf("第二个请求不应立即返回, 得到 %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	clock.Advance(time.Second)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("排队请求应最终成功: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("排队请求超时未返回")
	}
}

func TestLimiterRefillsOverTime(t *testing.T) {
	clock := newFakeClock()
	limiter := newTestLimiter(2, clock)

	// Capacity is 2 at 2 rps, so two calls pass before the bucket empties.
	for i := 0; i < 2; i++ {
		if err := limiter.Wait(context.Background(), 0); err != nil {
			t.Fatalf("第 %d 个请求应立即通过: %v", i+1, err)
		}
	}
	if err := limiter.Wait(context.Background(), 0); !errors.Is(err, ErrLimiterBusy) {
		t.Fatalf("桶空后应拒绝, 得到 %v", err)
	}

	clock.Advance(time.Second) // refills 2 tokens
	if err := limiter.Wait(context.Background(), 0); err != nil {
		t.Fatalf("补充后应通过: %v", err)
	}
}

func TestLimiterHonoursContextCancellation(t *testing.T) {
	clock := newFakeClock()
	limiter := newTestLimiter(1, clock)
	if err := limiter.Wait(context.Background(), 2*time.Second); err != nil {
		t.Fatalf("首个请求应立即通过: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- limiter.Wait(ctx, 2*time.Second) }()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("期望 context.Canceled, 得到 %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("取消后未及时返回")
	}
}

func TestLimiterDisabledWhenRateNonPositive(t *testing.T) {
	limiter := NewLimiter(0)
	for i := 0; i < 100; i++ {
		if err := limiter.Wait(context.Background(), 0); err != nil {
			t.Fatalf("速率为 0 时不应限流: %v", err)
		}
	}
}
