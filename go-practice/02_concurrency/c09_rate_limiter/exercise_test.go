package c09

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/go-practice/internal/clock"
	"example.com/go-practice/internal/testclock"
)

const testTimeout = 2 * time.Second

type waitResult struct {
	at  time.Time
	err error
}

func newLimiter(t *testing.T, interval time.Duration, c *testclock.Clock) *Limiter {
	t.Helper()
	limiter, err := NewLimiter(interval, c)
	if err != nil || limiter == nil {
		t.Fatalf("NewLimiter(%s) = %v, %v; want a limiter and nil error", interval, limiter, err)
	}
	return limiter
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return ctx
}

func startWait(limiter *Limiter, ctx context.Context, c *testclock.Clock) <-chan waitResult {
	result := make(chan waitResult, 1)
	go func() {
		err := limiter.Wait(ctx)
		result <- waitResult{at: c.Now(), err: err}
	}()
	return result
}

func receive(t *testing.T, results <-chan waitResult) waitResult {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(testTimeout):
		t.Fatal("Wait did not finish within the test deadline")
		return waitResult{}
	}
}

func waitForTimer(t *testing.T, c *testclock.Clock) {
	t.Helper()
	if err := c.WaitForTimers(testContext(t), 1); err != nil {
		t.Fatalf("Wait did not register a future timer: %v", err)
	}
}

func expectPending(t *testing.T, results <-chan waitResult) {
	t.Helper()
	select {
	case result := <-results:
		t.Fatalf("Wait finished before a permit was available: %+v", result)
	default:
	}
}

func expectPermit(t *testing.T, results <-chan waitResult, at time.Time) {
	t.Helper()
	result := receive(t, results)
	if result.err != nil || !result.at.Equal(at) {
		t.Fatalf("Wait = %+v; want nil error at %s", result, at)
	}
}

func TestLimiterRejectsInvalidInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Nanosecond} {
		t.Run(interval.String(), func(t *testing.T) {
			limiter, err := NewLimiter(interval, testclock.New(time.Unix(0, 0)))
			if err == nil || limiter != nil {
				t.Fatalf("NewLimiter(%s) = %v, %v; want nil and an error", interval, limiter, err)
			}
		})
	}
}

func TestLimiterFirstPermitAndIntervalBoundary(t *testing.T) {
	start := time.Unix(100, 0)
	c := testclock.New(start)
	limiter := newLimiter(t, time.Second, c)
	ctx := testContext(t)
	expectPermit(t, startWait(limiter, ctx, c), start)

	next := startWait(limiter, ctx, c)
	waitForTimer(t, c)
	c.Advance(time.Second - time.Nanosecond)
	expectPending(t, next)
	if c.ActiveTimers() == 0 {
		t.Fatal("the interval boundary still needs a future timer")
	}
	c.Advance(time.Nanosecond)
	expectPermit(t, next, start.Add(time.Second))
	if c.ActiveTimers() != 0 {
		t.Fatal("completed Wait left an active timer")
	}
}

func TestLimiterIdleDoesNotAccumulateBurst(t *testing.T) {
	start := time.Unix(100, 0)
	c := testclock.New(start)
	limiter := newLimiter(t, time.Second, c)
	ctx := testContext(t)
	expectPermit(t, startWait(limiter, ctx, c), start)
	c.Advance(10 * time.Second)
	expectPermit(t, startWait(limiter, ctx, c), start.Add(10*time.Second))

	next := startWait(limiter, ctx, c)
	waitForTimer(t, c)
	expectPending(t, next)
	c.Advance(time.Second)
	expectPermit(t, next, start.Add(11*time.Second))
}

func TestLimiterConcurrentWaitersHaveSpacedPermits(t *testing.T) {
	start := time.Unix(100, 0)
	c := testclock.New(start)
	limiter := newLimiter(t, time.Second, c)
	ctx := testContext(t)
	expectPermit(t, startWait(limiter, ctx, c), start)

	const callers = 4
	results := make(chan waitResult, callers)
	for range callers {
		go func() {
			err := limiter.Wait(ctx)
			results <- waitResult{at: c.Now(), err: err}
		}()
	}
	for i := 1; i <= callers; i++ {
		waitForTimer(t, c)
		expectPending(t, results)
		c.Advance(time.Second)
		expectPermit(t, results, start.Add(time.Duration(i)*time.Second))
		expectPending(t, results)
	}
	if c.ActiveTimers() != 0 {
		t.Fatal("finished callers left active timers")
	}
}

func TestLimiterCanceledWaitDoesNotDelayNextCaller(t *testing.T) {
	start := time.Unix(100, 0)
	c := testclock.New(start)
	limiter := newLimiter(t, time.Second, c)
	expectPermit(t, startWait(limiter, testContext(t), c), start)

	ctx, cancel := context.WithCancel(testContext(t))
	pending := startWait(limiter, ctx, c)
	waitForTimer(t, c)
	cancel()
	if result := receive(t, pending); !errors.Is(result.err, context.Canceled) {
		t.Fatalf("canceled Wait = %v; want context.Canceled", result.err)
	}
	if c.ActiveTimers() != 0 {
		t.Fatal("canceled Wait left an active timer")
	}

	next := startWait(limiter, testContext(t), c)
	waitForTimer(t, c)
	c.Advance(time.Second)
	expectPermit(t, next, start.Add(time.Second))
}

func TestLimiterPreCanceledContextDoesNotConsumeFirstPermit(t *testing.T) {
	start := time.Unix(100, 0)
	c := testclock.New(start)
	limiter := newLimiter(t, time.Second, c)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := receive(t, startWait(limiter, ctx, c)); !errors.Is(result.err, context.Canceled) {
		t.Fatalf("pre-canceled Wait = %v; want context.Canceled", result.err)
	}
	expectPermit(t, startWait(limiter, testContext(t), c), start)
}

func TestLimiterRealClockDeadline(t *testing.T) {
	limiter, err := NewLimiter(time.Hour, clock.Real{})
	if err != nil || limiter == nil {
		t.Fatalf("NewLimiter with real clock = %v, %v; want a limiter and nil error", limiter, err)
	}
	guard, cancelGuard := context.WithTimeout(context.Background(), testTimeout)
	defer cancelGuard()
	results := make(chan error, 1)
	go func() { results <- limiter.Wait(guard) }()
	select {
	case err := <-results:
		if err != nil {
			t.Fatalf("first Wait: %v", err)
		}
	case <-guard.Done():
		t.Fatal("first real-clock permit did not arrive")
	}
	ctx, cancel := context.WithTimeout(guard, 100*time.Millisecond)
	defer cancel()
	go func() { results <- limiter.Wait(ctx) }()
	select {
	case err := <-results:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("real-clock Wait = %v; want context.DeadlineExceeded", err)
		}
	case <-guard.Done():
		t.Fatal("real-clock Wait ignored its deadline")
	}
}
