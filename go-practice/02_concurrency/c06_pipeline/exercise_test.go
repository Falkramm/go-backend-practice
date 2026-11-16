package c06

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

type squaresResult struct {
	sum int
	err error
}

func awaitSquares(t *testing.T, result <-chan squaresResult) squaresResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("SumSquares did not return within the test timeout")
		return squaresResult{}
	}
}

func callSquares(t *testing.T, parent context.Context, input <-chan string, workers int) squaresResult {
	t.Helper()
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	result := make(chan squaresResult, 1)
	go func() {
		sum, err := SumSquares(ctx, input, workers)
		result <- squaresResult{sum: sum, err: err}
	}()
	return awaitSquares(t, result)
}

func stringsInput(values ...string) <-chan string {
	input := make(chan string, len(values))
	for _, value := range values {
		input <- value
	}
	close(input)
	return input
}

func TestSumSquares(t *testing.T) {
	cases := []struct {
		name    string
		values  []string
		workers int
		want    int
	}{
		{name: "one worker", values: []string{"1", "2", "3"}, workers: 1, want: 14},
		{name: "several workers", values: []string{"-3", "0", "2", "2"}, workers: 4, want: 17},
		{name: "empty input", workers: 3, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := callSquares(t, context.Background(), stringsInput(tc.values...), tc.workers)
			if got.err != nil || got.sum != tc.want {
				t.Fatalf("SumSquares = (%d, %v), want (%d, nil)", got.sum, got.err, tc.want)
			}
		})
	}
}

func TestSumSquaresPreservesParseError(t *testing.T) {
	cases := []struct {
		name  string
		value string
		cause error
	}{
		{name: "syntax", value: "broken", cause: strconv.ErrSyntax},
		{name: "range", value: strings.Repeat("9", 100), cause: strconv.ErrRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := callSquares(t, context.Background(), stringsInput("2", tc.value, "4"), 3)
			var numberError *strconv.NumError
			if got.sum != 0 || !errors.Is(got.err, tc.cause) || !errors.As(got.err, &numberError) {
				t.Fatalf("SumSquares = (%d, %v), want zero and wrapped Atoi error %v", got.sum, got.err, tc.cause)
			}
			if numberError.Num != tc.value {
				t.Errorf("parse input = %q, want %q", numberError.Num, tc.value)
			}
		})
	}
}

func TestSumSquaresRejectsInvalidWorkers(t *testing.T) {
	for _, workers := range []int{0, -1} {
		got := callSquares(t, context.Background(), stringsInput("1"), workers)
		if got.sum != 0 || got.err == nil {
			t.Errorf("workers=%d: got (%d, %v), want zero and an error", workers, got.sum, got.err)
		}
	}
}

func TestSumSquaresWithUnbufferedInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan string)
	producerDone := make(chan struct{})
	go func() {
		defer close(input)
		defer close(producerDone)
		for _, value := range []string{"2", "-3", "4"} {
			select {
			case input <- value:
			case <-ctx.Done():
				return
			}
		}
	}()
	got := callSquares(t, ctx, input, 2)
	if got.err != nil || got.sum != 29 {
		t.Fatalf("SumSquares = (%d, %v), want (29, nil)", got.sum, got.err)
	}
	select {
	case <-producerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("input producer did not finish")
	}
}

func TestSumSquaresCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := callSquares(t, ctx, make(chan string), 2)
	if got.sum != 0 || !errors.Is(got.err, context.Canceled) {
		t.Fatalf("SumSquares = (%d, %v), want (0, context.Canceled)", got.sum, got.err)
	}
}

func TestSumSquaresCancellationAfterInputExchange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan string)
	result := make(chan squaresResult, 1)
	go func() {
		sum, err := SumSquares(ctx, input, 2)
		result <- squaresResult{sum: sum, err: err}
	}()
	select {
	case input <- "3":
	case got := <-result:
		t.Fatalf("returned before accepting input: (%d, %v)", got.sum, got.err)
	case <-time.After(2 * time.Second):
		t.Fatal("pipeline did not accept input")
	}
	cancel()
	got := awaitSquares(t, result)
	if got.sum != 0 || !errors.Is(got.err, context.Canceled) {
		t.Fatalf("SumSquares = (%d, %v), want (0, context.Canceled)", got.sum, got.err)
	}
}
