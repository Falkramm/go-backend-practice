package c07

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"example.com/go-practice/internal/clock"
	"example.com/go-practice/internal/testclock"
)

func startBatch(ctx context.Context, input <-chan int, size int, wait time.Duration, c clock.Clock, handle func(context.Context, []int) error) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- Batch(ctx, input, size, wait, c, handle)
	}()
	return result
}

func awaitBatch(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Batch did not return within the test timeout")
		return nil
	}
}

func sendBatchValue(t *testing.T, input chan<- int, value int, result <-chan error) {
	t.Helper()
	select {
	case input <- value:
	case err := <-result:
		t.Fatalf("Batch returned before accepting %d: %v", value, err)
	case <-time.After(2 * time.Second):
		t.Fatalf("Batch did not accept %d", value)
	}
}

func waitForBatchTimer(t *testing.T, parent context.Context, c *testclock.Clock) {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	if err := c.WaitForTimers(ctx, 1); err != nil {
		t.Fatalf("timer for the pending batch was not created: %v", err)
	}
}

func expectBatch(t *testing.T, batches <-chan []int, result <-chan error, want []int) {
	t.Helper()
	select {
	case got := <-batches:
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("batch = %v, want %v", got, want)
		}
	case err := <-result:
		t.Fatalf("Batch returned before flushing %v: %v", want, err)
	case <-time.After(2 * time.Second):
		t.Fatalf("batch %v was not flushed", want)
	}
}

func TestBatchSizesAndInputClosure(t *testing.T) {
	cases := []struct {
		name   string
		values []int
		size   int
		want   [][]int
	}{
		{name: "empty input", size: 3},
		{name: "full batch", values: []int{1, 2, 3}, size: 3, want: [][]int{{1, 2, 3}}},
		{name: "several full batches", values: []int{1, 2, 3, 4}, size: 2, want: [][]int{{1, 2}, {3, 4}}},
		{name: "final remainder", values: []int{1, 2, 3, 4, 5}, size: 3, want: [][]int{{1, 2, 3}, {4, 5}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			input := make(chan int, len(tc.values))
			for _, value := range tc.values {
				input <- value
			}
			close(input)
			c := testclock.New(time.Unix(0, 0))
			var got [][]int
			handle := func(_ context.Context, values []int) error {
				got = append(got, append([]int(nil), values...))
				return nil
			}
			if err := awaitBatch(t, startBatch(ctx, input, tc.size, 10*time.Second, c, handle)); err != nil {
				t.Fatalf("Batch returned %v, want nil", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("batches = %v, want %v", got, tc.want)
			}
			if active := c.ActiveTimers(); active != 0 {
				t.Errorf("active timers after return = %d, want 0", active)
			}
		})
	}
}

func TestBatchTimerStartsWithFirstElement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := testclock.New(time.Unix(0, 0))
	input := make(chan int)
	batches := make(chan []int, 8)
	handle := func(_ context.Context, values []int) error {
		batches <- append([]int(nil), values...)
		return nil
	}
	result := startBatch(ctx, input, 3, 10*time.Second, c, handle)
	sendBatchValue(t, input, 1, result)
	waitForBatchTimer(t, ctx, c)
	c.Advance(4 * time.Second)
	sendBatchValue(t, input, 2, result)
	c.Advance(6 * time.Second)
	expectBatch(t, batches, result, []int{1, 2})

	// An empty period creates no output batch.
	c.Advance(100 * time.Second)
	sendBatchValue(t, input, 3, result)
	waitForBatchTimer(t, ctx, c)
	c.Advance(10 * time.Second)
	expectBatch(t, batches, result, []int{3})
	close(input)
	if err := awaitBatch(t, result); err != nil {
		t.Fatalf("Batch returned %v, want nil", err)
	}
	select {
	case extra := <-batches:
		t.Errorf("unexpected extra batch %v", extra)
	default:
	}
	if active := c.ActiveTimers(); active != 0 {
		t.Errorf("active timers after return = %d, want 0", active)
	}
}

func TestBatchPreservesHandlerErrorAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan int, 3)
	input <- 1
	input <- 2
	input <- 3
	close(input)
	c := testclock.New(time.Unix(0, 0))
	want := errors.New("handler failed")
	calls := 0
	handle := func(_ context.Context, values []int) error {
		calls++
		return want
	}
	err := awaitBatch(t, startBatch(ctx, input, 1, time.Second, c, handle))
	if !errors.Is(err, want) || calls != 1 {
		t.Fatalf("Batch returned %v after %d handler calls, want preserved error after one call", err, calls)
	}
	if active := c.ActiveTimers(); active != 0 {
		t.Errorf("active timers after handler error = %d, want 0", active)
	}
}

func TestBatchCancellationWithPendingTimer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan int)
	c := testclock.New(time.Unix(0, 0))
	handle := func(context.Context, []int) error { return nil }
	result := startBatch(ctx, input, 3, time.Minute, c, handle)
	sendBatchValue(t, input, 1, result)
	waitForBatchTimer(t, ctx, c)
	cancel()
	if err := awaitBatch(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("Batch returned %v, want context.Canceled", err)
	}
	if active := c.ActiveTimers(); active != 0 {
		t.Errorf("active timers after cancellation = %d, want 0", active)
	}
}

func TestBatchPropagatesCancellationToHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan int)
	c := testclock.New(time.Unix(0, 0))
	entered := make(chan struct{})
	handle := func(handlerContext context.Context, _ []int) error {
		close(entered)
		<-handlerContext.Done()
		return handlerContext.Err()
	}
	result := startBatch(ctx, input, 1, time.Minute, c, handle)
	sendBatchValue(t, input, 1, result)
	select {
	case <-entered:
	case err := <-result:
		t.Fatalf("Batch returned before handler entry: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not called")
	}
	cancel()
	if err := awaitBatch(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("Batch returned %v, want context.Canceled", err)
	}
	if active := c.ActiveTimers(); active != 0 {
		t.Errorf("active timers after handler cancellation = %d, want 0", active)
	}
}

func TestBatchCancellationDuringFinalFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan int, 1)
	input <- 1
	close(input)
	c := testclock.New(time.Unix(0, 0))
	entered := make(chan struct{})
	handle := func(handlerContext context.Context, _ []int) error {
		close(entered)
		<-handlerContext.Done()
		return nil
	}
	result := startBatch(ctx, input, 2, time.Minute, c, handle)
	select {
	case <-entered:
	case err := <-result:
		t.Fatalf("Batch returned before final handler entry: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("final handler was not called")
	}
	cancel()
	if err := awaitBatch(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("Batch returned %v, want context.Canceled", err)
	}
	if active := c.ActiveTimers(); active != 0 {
		t.Errorf("active timers after final flush cancellation = %d, want 0", active)
	}
}

func TestBatchAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := testclock.New(time.Unix(0, 0))
	calls := 0
	handle := func(context.Context, []int) error {
		calls++
		return nil
	}
	err := awaitBatch(t, startBatch(ctx, make(chan int), 3, time.Minute, c, handle))
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("Batch returned %v with %d handler calls, want context.Canceled and zero calls", err, calls)
	}
	if active := c.ActiveTimers(); active != 0 {
		t.Errorf("active timers after cancellation = %d, want 0", active)
	}
}

func TestBatchRejectsInvalidParameters(t *testing.T) {
	cases := []struct {
		name string
		size int
		wait time.Duration
	}{
		{name: "zero size", size: 0, wait: time.Second},
		{name: "negative size", size: -1, wait: time.Second},
		{name: "zero wait", size: 2, wait: 0},
		{name: "negative wait", size: 2, wait: -time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			input := make(chan int, 1)
			input <- 1
			close(input)
			c := testclock.New(time.Unix(0, 0))
			calls := 0
			handle := func(context.Context, []int) error {
				calls++
				return nil
			}
			err := awaitBatch(t, startBatch(ctx, input, tc.size, tc.wait, c, handle))
			if err == nil || calls != 0 {
				t.Fatalf("Batch returned %v with %d calls, want an error before processing", err, calls)
			}
			if active := c.ActiveTimers(); active != 0 {
				t.Errorf("active timers after invalid parameters = %d, want 0", active)
			}
		})
	}
}
