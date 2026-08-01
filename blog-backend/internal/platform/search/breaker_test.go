package search

import (
	"testing"
	"time"
)

func newTestBreaker(clock *fakeClock) *Breaker {
	breaker := NewBreaker()
	breaker.now = clock.Now
	return breaker
}

func TestBreakerStartsClosed(t *testing.T) {
	breaker := newTestBreaker(newFakeClock())
	if state := breaker.State(); state != BreakerClosed {
		t.Fatalf("初始状态 = %q, 期望 closed", state)
	}
	if !breaker.Allow() {
		t.Fatal("closed 状态应放行")
	}
}

func TestBreakerTripsOnConsecutiveFailures(t *testing.T) {
	breaker := newTestBreaker(newFakeClock())

	for i := 0; i < defaultConsecutiveTrips-1; i++ {
		breaker.Report(false)
		if state := breaker.State(); state != BreakerClosed {
			t.Fatalf("第 %d 次失败后状态 = %q, 期望仍 closed", i+1, state)
		}
	}
	breaker.Report(false)
	if state := breaker.State(); state != BreakerOpen {
		t.Fatalf("连续 %d 次失败后状态 = %q, 期望 open", defaultConsecutiveTrips, state)
	}
	if breaker.Allow() {
		t.Fatal("open 状态应拦截")
	}
}

func TestBreakerSuccessResetsConsecutiveCount(t *testing.T) {
	breaker := newTestBreaker(newFakeClock())

	for i := 0; i < defaultConsecutiveTrips-1; i++ {
		breaker.Report(false)
	}
	breaker.Report(true)
	for i := 0; i < defaultConsecutiveTrips-1; i++ {
		breaker.Report(false)
	}
	if state := breaker.State(); state != BreakerClosed {
		t.Fatalf("成功打断连败后状态 = %q, 期望 closed", state)
	}
}

func TestBreakerTripsOnFailureRatio(t *testing.T) {
	breaker := newTestBreaker(newFakeClock())

	// 12 samples, 7 failures (58%), never 5 in a row.
	pattern := []bool{false, false, true, false, false, true, false, false, true, false, true, true}
	for _, success := range pattern {
		breaker.Report(success)
	}
	if state := breaker.State(); state != BreakerOpen {
		t.Fatalf("窗口失败率超阈值后状态 = %q, 期望 open", state)
	}
}

func TestBreakerIgnoresFailuresOutsideWindow(t *testing.T) {
	clock := newFakeClock()
	breaker := newTestBreaker(clock)

	for i := 0; i < defaultConsecutiveTrips-1; i++ {
		breaker.Report(false)
	}
	// Age the samples past the sliding window, then keep failing but stay
	// under the consecutive threshold by interleaving a success.
	clock.Advance(2 * defaultBreakerWindow)
	breaker.Report(true)
	breaker.Report(false)

	if state := breaker.State(); state != BreakerClosed {
		t.Fatalf("窗口外样本不应触发熔断, 状态 = %q", state)
	}
}

func TestBreakerHalfOpenAdmitsSingleProbe(t *testing.T) {
	clock := newFakeClock()
	breaker := newTestBreaker(clock)

	for i := 0; i < defaultConsecutiveTrips; i++ {
		breaker.Report(false)
	}
	clock.Advance(defaultBreakerOpenFor)

	if state := breaker.State(); state != BreakerHalfOpen {
		t.Fatalf("冷却后状态 = %q, 期望 half_open", state)
	}
	if !breaker.Allow() {
		t.Fatal("half_open 应放行首个探测")
	}
	if breaker.Allow() {
		t.Fatal("half_open 只应放行一个探测")
	}
}

func TestBreakerHalfOpenProbeSuccessCloses(t *testing.T) {
	clock := newFakeClock()
	breaker := newTestBreaker(clock)

	for i := 0; i < defaultConsecutiveTrips; i++ {
		breaker.Report(false)
	}
	clock.Advance(defaultBreakerOpenFor)
	breaker.Allow()
	breaker.Report(true)

	if state := breaker.State(); state != BreakerClosed {
		t.Fatalf("探测成功后状态 = %q, 期望 closed", state)
	}
	if !breaker.Allow() {
		t.Fatal("恢复后应放行")
	}
}

func TestBreakerReleaseDoesNotCountAsSuccess(t *testing.T) {
	breaker := newTestBreaker(newFakeClock())

	// A run of failures interleaved with releases must still trip: a released
	// call never reached the upstream and proves nothing about its health.
	for i := 0; i < defaultConsecutiveTrips; i++ {
		breaker.Report(false)
		breaker.Allow()
		breaker.Release()
	}
	if state := breaker.State(); state != BreakerOpen {
		t.Fatalf("状态 = %q, Release 不应重置连败计数", state)
	}
}

func TestBreakerReleaseFreesHalfOpenProbeSlot(t *testing.T) {
	clock := newFakeClock()
	breaker := newTestBreaker(clock)

	for i := 0; i < defaultConsecutiveTrips; i++ {
		breaker.Report(false)
	}
	clock.Advance(defaultBreakerOpenFor)

	if !breaker.Allow() {
		t.Fatal("half_open 应放行首个探测")
	}
	breaker.Release()
	if !breaker.Allow() {
		t.Fatal("释放后探测名额应可再次占用，否则供应商将永远卡在 half_open")
	}
	if state := breaker.State(); state != BreakerHalfOpen {
		t.Fatalf("状态 = %q, Release 不应改变状态", state)
	}
}

func TestBreakerHalfOpenProbeFailureReopens(t *testing.T) {
	clock := newFakeClock()
	breaker := newTestBreaker(clock)

	for i := 0; i < defaultConsecutiveTrips; i++ {
		breaker.Report(false)
	}
	clock.Advance(defaultBreakerOpenFor)
	breaker.Allow()
	breaker.Report(false)

	if state := breaker.State(); state != BreakerOpen {
		t.Fatalf("探测失败后状态 = %q, 期望重新 open", state)
	}
	// The cooldown restarts from the failed probe, not from the first trip.
	clock.Advance(defaultBreakerOpenFor - time.Second)
	if state := breaker.State(); state != BreakerOpen {
		t.Fatalf("冷却未满时状态 = %q, 期望仍 open", state)
	}
	clock.Advance(time.Second)
	if state := breaker.State(); state != BreakerHalfOpen {
		t.Fatalf("冷却期满后状态 = %q, 期望 half_open", state)
	}
}
