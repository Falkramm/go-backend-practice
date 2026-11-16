package c03

import (
	"context"
	"slices"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

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

func startMerge(t *testing.T, ctx context.Context, inputs ...<-chan int) <-chan int {
	t.Helper()
	returned := make(chan (<-chan int), 1)
	go func() { returned <- Merge(ctx, inputs...) }()
	out := waitFor(t, returned)
	if out == nil {
		t.Fatal("Merge returned a nil output channel")
	}
	return out
}

func collect(t *testing.T, out <-chan int) []int {
	t.Helper()
	timer := time.NewTimer(testTimeout)
	defer timer.Stop()
	var values []int
	for {
		select {
		case value, ok := <-out:
			if !ok {
				return values
			}
			values = append(values, value)
		case <-timer.C:
			t.Fatal("output channel did not close before the timeout")
			return nil
		}
	}
}

func input(values ...int) <-chan int {
	ch := make(chan int, len(values))
	for _, value := range values {
		ch <- value
	}
	close(ch)
	return ch
}

func TestMergeValuesAndOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got := collect(t, startMerge(t, ctx,
		input(101, 102, 103), input(201, 202), input(301),
	))
	want := []int{101, 102, 103, 201, 202, 301}
	sorted := slices.Clone(got)
	slices.Sort(sorted)
	if !slices.Equal(sorted, want) {
		t.Fatalf("merged values = %v, want exactly %v", got, want)
	}
	positions := make(map[int]int, len(got))
	for index, value := range got {
		positions[value] = index
	}
	for _, pair := range [][2]int{{101, 102}, {102, 103}, {201, 202}} {
		if positions[pair[0]] >= positions[pair[1]] {
			t.Fatalf("source order was lost in %v", got)
		}
	}
}

func TestMergeEmptyInputs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inputs []<-chan int
	}{
		{name: "no inputs"},
		{name: "closed inputs", inputs: []<-chan int{input(), input()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := collect(t, startMerge(t, context.Background(), tc.inputs...))
			if len(got) != 0 {
				t.Fatalf("merged values = %v, want an empty result", got)
			}
		})
	}
}

func TestMergeForwardsBeforeEveryInputCloses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	later := make(chan int)
	out := startMerge(t, ctx, input(1, 2), later)
	for _, want := range []int{1, 2} {
		select {
		case got, ok := <-out:
			if !ok || got != want {
				t.Fatalf("received (%d, %t), want (%d, true)", got, ok, want)
			}
		case <-time.After(testTimeout):
			t.Fatal("an open input prevented forwarding from another input")
		}
	}
	select {
	case value, ok := <-out:
		t.Fatalf("output produced (%d, %t) while the remaining input was open and empty", value, ok)
	default:
	}
	sent := make(chan struct{}, 1)
	go func() {
		defer close(later)
		select {
		case later <- 3:
		case <-ctx.Done():
		}
		sent <- struct{}{}
	}()
	if got := collect(t, out); !slices.Equal(got, []int{3}) {
		t.Fatalf("remaining values = %v, want [3]", got)
	}
	waitFor(t, sent)
}

func TestMergeWithDelayedReceiver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan int)
	firstSent := make(chan struct{}, 1)
	producerDone := make(chan struct{}, 1)
	go func() {
		defer close(in)
		defer func() { producerDone <- struct{}{} }()
		for _, value := range []int{1, 2, 3} {
			select {
			case in <- value:
				if value == 1 {
					firstSent <- struct{}{}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	out := startMerge(t, ctx, in)
	waitFor(t, firstSent)
	if got := collect(t, out); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("merged values = %v, want [1 2 3]", got)
	}
	waitFor(t, producerDone)
}

func TestMergeCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	in := make(chan int)
	if got := collect(t, startMerge(t, ctx, in)); len(got) != 0 {
		t.Fatalf("merged values = %v, want an empty result", got)
	}
}

func TestMergeCanceledAfterFirstInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan int)
	firstSent := make(chan struct{}, 1)
	go func() {
		select {
		case in <- 1:
		case <-ctx.Done():
		}
		firstSent <- struct{}{}
	}()
	out := startMerge(t, ctx, in)
	waitFor(t, firstSent)
	cancel()
	got := collect(t, out)
	if len(got) > 1 || (len(got) == 1 && got[0] != 1) {
		t.Fatalf("canceled merge returned %v, want a prefix of [1]", got)
	}
}
