package c05

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

type mapResult struct {
	values []int
	err    error
}

func waitFor[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(testTimeout):
		t.Fatal("operation did not finish before the timeout")
		var zero T
		return zero
	}
}

func startMap(ctx context.Context, items []int, concurrency int, fn func(context.Context, int) (int, error)) <-chan mapResult {
	finished := make(chan mapResult, 1)
	go func() {
		values, err := ParallelMap(ctx, items, concurrency, fn)
		finished <- mapResult{values: values, err: err}
	}()
	return finished
}

func TestParallelMapValues(t *testing.T) {
	for _, tc := range []struct {
		name        string
		items       []int
		concurrency int
		want        []int
	}{
		{name: "serial", items: []int{3, 1, 3, -2}, concurrency: 1, want: []int{30, 10, 30, -20}},
		{name: "parallel", items: []int{3, 1, 3, -2}, concurrency: 3, want: []int{30, 10, 30, -20}},
		{name: "empty", concurrency: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls atomic.Int32
			got := waitFor(t, startMap(ctx, tc.items, tc.concurrency, func(ctx context.Context, value int) (int, error) {
				calls.Add(1)
				return value * 10, nil
			}))
			if got.err != nil || !slices.Equal(got.values, tc.want) {
				t.Fatalf("ParallelMap returned (%v, %v), want (%v, nil)", got.values, got.err, tc.want)
			}
			if gotCalls := int(calls.Load()); gotCalls != len(tc.items) {
				t.Fatalf("handler calls = %d, want %d", gotCalls, len(tc.items))
			}
		})
	}
}

func TestParallelMapPreservesInputOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	items := []int{10, 20, 30}
	gates := map[int]chan struct{}{10: make(chan struct{}), 20: make(chan struct{}), 30: make(chan struct{})}
	started := make(chan int, len(items))
	completed := make(chan int, len(items))
	finished := startMap(ctx, items, 3, func(ctx context.Context, value int) (int, error) {
		select {
		case started <- value:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		select {
		case <-gates[value]:
			completed <- value
			return value * 2, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})
	seen := make(map[int]bool)
	for range items {
		seen[waitFor(t, started)] = true
	}
	if len(seen) != len(items) {
		t.Fatalf("started items = %v, want each input once", seen)
	}
	for _, value := range []int{30, 10, 20} {
		close(gates[value])
		if got := waitFor(t, completed); got != value {
			t.Fatalf("completed item = %d, want %d", got, value)
		}
	}
	got := waitFor(t, finished)
	if got.err != nil || !slices.Equal(got.values, []int{20, 40, 60}) {
		t.Fatalf("ParallelMap returned (%v, %v), want ([20 40 60], nil)", got.values, got.err)
	}
}

func TestParallelMapConcurrencyLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	items := []int{1, 2, 3, 4, 5, 6, 7, 8}
	const limit = 3
	started := make(chan int, len(items))
	permits := make(chan struct{})
	defer func() { close(permits) }()
	var active, peak atomic.Int32
	finished := startMap(ctx, items, limit, func(ctx context.Context, value int) (int, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			previous := peak.Load()
			if current <= previous || peak.CompareAndSwap(previous, current) {
				break
			}
		}
		select {
		case started <- value:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		select {
		case <-permits:
			return value * 2, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})
	seen := make(map[int]bool)
	for index := range items {
		if index >= limit {
			select {
			case permits <- struct{}{}:
			case <-time.After(testTimeout):
				t.Fatal("active handlers did not accept a completion permit")
			}
		}
		value := waitFor(t, started)
		if value < 1 || value > len(items) || seen[value] {
			t.Fatalf("unexpected or repeated item %d", value)
		}
		seen[value] = true
	}
	for range limit {
		select {
		case permits <- struct{}{}:
		case <-time.After(testTimeout):
			t.Fatal("remaining handlers did not accept completion permits")
		}
	}
	got := waitFor(t, finished)
	if got.err != nil || !slices.Equal(got.values, []int{2, 4, 6, 8, 10, 12, 14, 16}) {
		t.Fatalf("ParallelMap returned (%v, %v)", got.values, got.err)
	}
	if gotPeak := peak.Load(); gotPeak > limit || gotPeak < 1 {
		t.Fatalf("peak active handlers = %d, want 1..%d", gotPeak, limit)
	}
	if gotActive := active.Load(); gotActive != 0 {
		t.Fatalf("active handlers after return = %d, want 0", gotActive)
	}
}

func TestParallelMapInvalidConcurrency(t *testing.T) {
	for _, limit := range []int{0, -1} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			var calls atomic.Int32
			got := waitFor(t, startMap(context.Background(), []int{1}, limit, func(context.Context, int) (int, error) {
				calls.Add(1)
				return 1, nil
			}))
			if got.err == nil || got.values != nil || calls.Load() != 0 {
				t.Fatalf("invalid limit returned (%v, %v), handler calls %d", got.values, got.err, calls.Load())
			}
		})
	}
}

func TestParallelMapCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, items := range [][]int{nil, {1, 2}} {
		var calls atomic.Int32
		got := waitFor(t, startMap(ctx, items, 2, func(context.Context, int) (int, error) {
			calls.Add(1)
			return 1, nil
		}))
		if got.values != nil || !errors.Is(got.err, ctx.Err()) || calls.Load() != 0 {
			t.Fatalf("canceled map returned (%v, %v), handler calls %d", got.values, got.err, calls.Load())
		}
	}
}

func TestParallelMapErrorCancelsAndWaits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	failure := errors.New("handler failure")
	started := make(chan int, 4)
	failNow := make(chan struct{})
	canceled := make(chan struct{}, 1)
	allowReturn := make(chan struct{})
	release := sync.OnceFunc(func() { close(allowReturn) })
	defer release()
	var calls, exited atomic.Int32
	finished := startMap(ctx, []int{1, 2, 3, 4}, 2, func(ctx context.Context, value int) (int, error) {
		position := calls.Add(1)
		defer exited.Add(1)
		started <- value
		switch position {
		case 1:
			select {
			case <-failNow:
				return 0, failure
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		case 2:
			<-ctx.Done()
			canceled <- struct{}{}
			<-allowReturn
			return 0, ctx.Err()
		default:
			return value, nil
		}
	})
	first, second := waitFor(t, started), waitFor(t, started)
	if first == second || first < 1 || first > 4 || second < 1 || second > 4 {
		t.Fatalf("initial handlers = [%d %d], want two distinct input items", first, second)
	}
	close(failNow)
	waitFor(t, canceled)
	select {
	case got := <-finished:
		t.Fatalf("ParallelMap returned while a canceled handler was held: %+v", got)
	default:
	}
	release()
	got := waitFor(t, finished)
	if got.values != nil || !errors.Is(got.err, failure) {
		t.Fatalf("ParallelMap returned (%v, %v), want (nil, handler failure)", got.values, got.err)
	}
	if calls.Load() != 2 || exited.Load() != 2 {
		t.Fatalf("handler calls/exits = %d/%d, want 2/2", calls.Load(), exited.Load())
	}
}

func TestParallelMapExternalCancellationWaits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan int, 3)
	canceled := make(chan struct{}, 2)
	allowReturn := make(chan struct{})
	release := sync.OnceFunc(func() { close(allowReturn) })
	defer release()
	var calls, exited atomic.Int32
	finished := startMap(ctx, []int{1, 2, 3}, 2, func(ctx context.Context, value int) (int, error) {
		calls.Add(1)
		defer exited.Add(1)
		started <- value
		<-ctx.Done()
		canceled <- struct{}{}
		<-allowReturn
		return 0, ctx.Err()
	})
	waitFor(t, started)
	waitFor(t, started)
	cancel()
	waitFor(t, canceled)
	waitFor(t, canceled)
	select {
	case got := <-finished:
		t.Fatalf("ParallelMap returned while canceled handlers were held: %+v", got)
	default:
	}
	release()
	got := waitFor(t, finished)
	if got.values != nil || !errors.Is(got.err, ctx.Err()) {
		t.Fatalf("ParallelMap returned (%v, %v), want (nil, error matching %v)", got.values, got.err, ctx.Err())
	}
	if calls.Load() != 2 || exited.Load() != 2 {
		t.Fatalf("handler calls/exits = %d/%d, want 2/2", calls.Load(), exited.Load())
	}
}

func TestParallelMapExternalCancellationWithSuccessfulHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	finished := startMap(ctx, []int{7}, 1, func(ctx context.Context, value int) (int, error) {
		close(started)
		<-ctx.Done()
		return value, nil
	})
	waitFor(t, started)
	cancel()
	got := waitFor(t, finished)
	if got.values != nil || !errors.Is(got.err, ctx.Err()) {
		t.Fatalf("ParallelMap returned (%v, %v), want (nil, error matching %v)", got.values, got.err, ctx.Err())
	}
}
