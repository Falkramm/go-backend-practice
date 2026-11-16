package c10

import (
	"testing"
	"time"

	"example.com/go-practice/internal/testclock"
)

const testTimeout = 2 * time.Second

type accessResult struct {
	key        string
	allowed    bool
	retryAfter time.Duration
}

func newBucket(t *testing.T, capacity, rate int, c *testclock.Clock) *Bucket {
	t.Helper()
	bucket, err := NewBucket(capacity, rate, c)
	if err != nil || bucket == nil {
		t.Fatalf("NewBucket(%d, %v) returned (%v, %v), want a bucket and nil error", capacity, rate, bucket, err)
	}
	t.Cleanup(func() {
		if active := c.ActiveTimers(); active != 0 {
			t.Errorf("Access left %d active timers, want 0", active)
		}
	})
	return bucket
}

func receive[T any](t *testing.T, results <-chan T) T {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(testTimeout):
		t.Fatal("Access did not return within the test deadline")
		var zero T
		return zero
	}
}

func access(t *testing.T, bucket *Bucket, key string) accessResult {
	t.Helper()
	results := make(chan accessResult, 1)
	go func() {
		allowed, retryAfter := bucket.Access(key)
		results <- accessResult{key: key, allowed: allowed, retryAfter: retryAfter}
	}()
	return receive(t, results)
}

func expectAccess(t *testing.T, bucket *Bucket, key string, allowed bool, retryAfter time.Duration) {
	t.Helper()
	got := access(t, bucket, key)
	if got.allowed != allowed || got.retryAfter != retryAfter {
		t.Fatalf("Access(%q) returned (%t, %s), want (%t, %s)", key, got.allowed, got.retryAfter, allowed, retryAfter)
	}
}

func TestBucketRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		capacity int
		rate     int
	}{
		{name: "zero capacity", capacity: 0, rate: 1},
		{name: "negative capacity", capacity: -1, rate: 1},
		{name: "zero rate", capacity: 1, rate: 0},
		{name: "negative rate", capacity: 1, rate: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bucket, err := NewBucket(tc.capacity, tc.rate, testclock.New(time.Unix(100, 0)))
			if bucket != nil || err == nil {
				t.Fatalf("NewBucket returned (%v, %v), want (nil, error)", bucket, err)
			}
		})
	}
}

func TestBucketKeysStartFullAndRefillIndependently(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 2, 1, c)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
	c.Advance(500 * time.Millisecond)
	for _, key := range []string{"bob", ""} {
		for range 2 {
			expectAccess(t, bucket, key, true, 0)
		}
		expectAccess(t, bucket, key, false, time.Second)
	}
	c.Advance(500 * time.Millisecond)
	expectAccess(t, bucket, "alice", true, 0)
	expectAccess(t, bucket, "alice", false, time.Second)
	for _, key := range []string{"bob", ""} {
		expectAccess(t, bucket, key, false, time.Second)
	}
	c.Advance(500 * time.Millisecond)
	for _, key := range []string{"bob", ""} {
		expectAccess(t, bucket, key, true, 0)
		expectAccess(t, bucket, key, false, time.Second)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
	for range 2 {
		expectAccess(t, bucket, "new key", true, 0)
	}
	expectAccess(t, bucket, "new key", false, time.Second)
}

func TestBucketDeniedCallsPreserveRefillBoundaryAndDoNotReserve(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 1, 2, c)
	expectAccess(t, bucket, "alice", true, 0)
	for range 10 {
		expectAccess(t, bucket, "alice", false, time.Second)
	}
	for range 7 {
		c.Advance(125 * time.Millisecond)
		expectAccess(t, bucket, "alice", false, time.Second)
		expectAccess(t, bucket, "alice", false, time.Second)
	}
	c.Advance(125 * time.Millisecond)
	expectAccess(t, bucket, "alice", true, 0)
	expectAccess(t, bucket, "alice", false, time.Second)
}

func TestBucketExactRefillBoundary(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 3, 2, c)
	for range 3 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	c.Advance(time.Second - time.Nanosecond)
	expectAccess(t, bucket, "alice", false, time.Second)
	c.Advance(time.Nanosecond)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
}

func TestBucketAccumulatesCompleteSeconds(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 5, 2, c)
	for range 5 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	c.Advance(2500 * time.Millisecond)
	for range 4 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
	c.Advance(500 * time.Millisecond)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
}

func TestBucketIdleRefillIsCapped(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 2, 2, c)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	c.Advance(10*time.Hour + 500*time.Millisecond)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
	c.Advance(500 * time.Millisecond)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
}

func TestBucketRetryAfterIsWholeSecond(t *testing.T) {
	for _, rate := range []int{1, 3, 10} {
		c := testclock.New(time.Unix(100, 0))
		bucket := newBucket(t, 1, rate, c)
		expectAccess(t, bucket, "alice", true, 0)
		expectAccess(t, bucket, "alice", false, time.Second)
		c.Advance(500 * time.Millisecond)
		expectAccess(t, bucket, "alice", false, time.Second)
		c.Advance(500 * time.Millisecond)
		expectAccess(t, bucket, "alice", true, 0)
		expectAccess(t, bucket, "alice", false, time.Second)
	}
}

func TestBucketRefillCapsBeforeIntegerOverflow(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	rate := int(^uint(0) >> 1)
	bucket := newBucket(t, 2, rate, c)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	c.Advance(2 * time.Second)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
}

func TestBucketLongIdleDoesNotRefillTwice(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 1, 1, c)
	expectAccess(t, bucket, "alice", true, 0)
	const maxAdvance = time.Duration(1<<63 - 1)
	c.Advance(maxAdvance)
	c.Advance(maxAdvance)
	expectAccess(t, bucket, "alice", true, 0)
	expectAccess(t, bucket, "alice", false, time.Second)
}

func TestBucketConcurrentSameKeyRespectsCapacity(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 7, 2, c)
	const callers = 100
	start := make(chan struct{})
	results := make(chan accessResult, callers)
	for range callers {
		go func() {
			<-start
			allowed, retryAfter := bucket.Access("shared")
			results <- accessResult{key: "shared", allowed: allowed, retryAfter: retryAfter}
		}()
	}
	close(start)
	allowedCount := 0
	for range callers {
		got := receive(t, results)
		if got.allowed {
			allowedCount++
			if got.retryAfter != 0 {
				t.Fatalf("allowed Access returned retryAfter %s, want 0", got.retryAfter)
			}
		} else if got.retryAfter != time.Second {
			t.Fatalf("denied Access returned retryAfter %s, want 1s", got.retryAfter)
		}
	}
	if allowedCount != 7 {
		t.Fatalf("concurrent permits = %d, want 7", allowedCount)
	}
}

func TestBucketConcurrentKeysHaveIndependentCapacity(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 3, 4, c)
	keys := []string{"alice", "bob", "carol", ""}
	const callersPerKey = 24
	start := make(chan struct{})
	results := make(chan accessResult, len(keys)*callersPerKey)
	for _, key := range keys {
		for range callersPerKey {
			go func() {
				<-start
				allowed, retryAfter := bucket.Access(key)
				results <- accessResult{key: key, allowed: allowed, retryAfter: retryAfter}
			}()
		}
	}
	close(start)
	allowedByKey := make(map[string]int)
	for range len(keys) * callersPerKey {
		got := receive(t, results)
		if got.allowed {
			allowedByKey[got.key]++
			if got.retryAfter != 0 {
				t.Fatalf("allowed Access(%q) returned retryAfter %s, want 0", got.key, got.retryAfter)
			}
		} else if got.retryAfter != time.Second {
			t.Fatalf("denied Access(%q) returned retryAfter %s, want 1s", got.key, got.retryAfter)
		}
	}
	for _, key := range keys {
		if got := allowedByKey[key]; got != 3 {
			t.Fatalf("concurrent permits for %q = %d, want 3", key, got)
		}
	}
}

func TestBucketSuccessfulAccessPreservesRefillBoundary(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 4, 2, c)
	expectAccess(t, bucket, "alice", true, 0)
	c.Advance(500 * time.Millisecond)
	for range 3 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	c.Advance(500 * time.Millisecond)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
}

func TestBucketDelayedAccessPreservesElapsedRemainder(t *testing.T) {
	c := testclock.New(time.Unix(100, 0))
	bucket := newBucket(t, 3, 2, c)
	for range 3 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	c.Advance(1500 * time.Millisecond)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
	c.Advance(500 * time.Millisecond)
	for range 2 {
		expectAccess(t, bucket, "alice", true, 0)
	}
	expectAccess(t, bucket, "alice", false, time.Second)
}
