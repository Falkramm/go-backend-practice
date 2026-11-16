package c07

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/go-practice/internal/clock"
)

func TestBatchRealClockCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan int)
	handle := func(context.Context, []int) error { return nil }
	result := startBatch(ctx, input, 2, time.Hour, clock.Real{}, handle)
	sendBatchValue(t, input, 1, result)
	cancellation := time.AfterFunc(100*time.Millisecond, cancel)
	defer cancellation.Stop()
	if err := awaitBatch(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("Batch with a real clock returned %v, want context.Canceled", err)
	}
}
