package c02

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

type consumeResult struct {
	sum int
	err error
}

func waitFor[T any](t *testing.T, results <-chan T) T {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(testTimeout):
		t.Fatal("function did not return before the timeout")
		var zero T
		return zero
	}
}

func requireClosed(t *testing.T, out <-chan int) {
	t.Helper()
	select {
	case value, ok := <-out:
		if ok {
			t.Fatalf("unexpected value %d after Produce returned", value)
		}
	default:
		t.Fatal("Produce returned with its output channel still open")
	}
}

func TestProduce(t *testing.T) {
	for _, n := range []int{-3, 0, 1, 5, 6} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			out := make(chan int, max(n, 0))
			finished := make(chan error, 1)
			go func() { finished <- Produce(context.Background(), n, out) }()
			if err := waitFor(t, finished); err != nil {
				t.Fatalf("Produce returned an error: %v", err)
			}
			for want := 1; want <= n; want++ {
				select {
				case got, ok := <-out:
					if !ok || got != want {
						t.Fatalf("received (%d, %t), want (%d, true)", got, ok, want)
					}
				default:
					t.Fatalf("Produce returned before value %d was available", want)
				}
			}
			requireClosed(t, out)
		})
	}
}

func TestConsume(t *testing.T) {
	for _, n := range []int{0, 1, 5, 6} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			in := make(chan int, n)
			for value := 1; value <= n; value++ {
				in <- value
			}
			close(in)
			finished := make(chan consumeResult, 1)
			go func() {
				sum, err := Consume(context.Background(), in)
				finished <- consumeResult{sum: sum, err: err}
			}()
			got := waitFor(t, finished)
			if got.err != nil || got.sum != n*(n+1)/2 {
				t.Fatalf("Consume returned (%d, %v), want (%d, nil)", got.sum, got.err, n*(n+1)/2)
			}
		})
	}
}

func TestProducerConsumer(t *testing.T) {
	for _, n := range []int{0, 1, 6} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stream := make(chan int)
			producerFinished := make(chan error, 1)
			consumerFinished := make(chan consumeResult, 1)

			go func() { producerFinished <- Produce(ctx, n, stream) }()
			go func() {
				sum, err := Consume(ctx, stream)
				consumerFinished <- consumeResult{sum: sum, err: err}
			}()

			got := waitFor(t, consumerFinished)
			if got.err != nil || got.sum != n*(n+1)/2 {
				t.Fatalf("Consume returned (%d, %v), want (%d, nil)", got.sum, got.err, n*(n+1)/2)
			}
			if err := waitFor(t, producerFinished); err != nil {
				t.Fatalf("Produce returned an error: %v", err)
			}
		})
	}
}

func TestProduceCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, n := range []int{0, 3} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			out := make(chan int)
			finished := make(chan error, 1)
			go func() { finished <- Produce(ctx, n, out) }()
			if err := waitFor(t, finished); !errors.Is(err, ctx.Err()) {
				t.Fatalf("Produce returned %v, want an error matching %v", err, ctx.Err())
			}
			requireClosed(t, out)
		})
	}
}

func TestConsumeCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, closed := range []bool{false, true} {
		t.Run("input_closed="+strconv.FormatBool(closed), func(t *testing.T) {
			in := make(chan int)
			if closed {
				close(in)
			}
			finished := make(chan consumeResult, 1)
			go func() {
				sum, err := Consume(ctx, in)
				finished <- consumeResult{sum: sum, err: err}
			}()
			got := waitFor(t, finished)
			if got.sum != 0 || !errors.Is(got.err, ctx.Err()) {
				t.Fatalf("Consume returned (%d, %v), want (0, error matching %v)", got.sum, got.err, ctx.Err())
			}
		})
	}
}

func TestProduceCanceledAfterFirstValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan int)
	finished := make(chan error, 1)
	go func() { finished <- Produce(ctx, 3, out) }()
	select {
	case value, ok := <-out:
		if !ok || value != 1 {
			t.Fatalf("first receive was (%d, %t), want (1, true)", value, ok)
		}
	case err := <-finished:
		t.Fatalf("Produce returned before its first value: %v", err)
	case <-time.After(testTimeout):
		t.Fatal("Produce did not send its first value before the timeout")
	}
	cancel()
	if err := waitFor(t, finished); !errors.Is(err, ctx.Err()) {
		t.Fatalf("Produce returned %v, want an error matching %v", err, ctx.Err())
	}
	requireClosed(t, out)
}

func TestConsumeCanceledAfterFirstValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan int)
	defer close(in)
	finished := make(chan consumeResult, 1)
	go func() {
		sum, err := Consume(ctx, in)
		finished <- consumeResult{sum: sum, err: err}
	}()
	select {
	case in <- 7:
	case got := <-finished:
		t.Fatalf("Consume returned before its first value: (%d, %v)", got.sum, got.err)
	case <-time.After(testTimeout):
		t.Fatal("Consume did not receive its first value before the timeout")
	}
	cancel()
	got := waitFor(t, finished)
	if got.sum != 7 || !errors.Is(got.err, ctx.Err()) {
		t.Fatalf("Consume returned (%d, %v), want (7, error matching %v)", got.sum, got.err, ctx.Err())
	}
}
