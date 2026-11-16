package c07

import (
	"context"
	"errors"
	"time"

	"fmt"

	"example.com/go-practice/internal/clock"
)

var (
	ErrBatchSizeIllegal = errors.New("illegal batch size")
	ErrMaxWaitIllegal   = errors.New("illegal max wait duration")
)

// Batch sends ordered batches to handle by size, age, or input closure.
// TODO: Implement the contract and release every timer before returning.
func Batch(ctx context.Context, input <-chan int, batchSize int, maxWait time.Duration, c clock.Clock, handle func(context.Context, []int) error) error {
	if batchSize <= 0 {
		return ErrBatchSizeIllegal
	}

	if maxWait <= 0 {
		return ErrMaxWaitIllegal
	}

	batch := make([]int, 0, batchSize)
	var timer clock.Timer
	var timerC <-chan time.Time

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
		}
		timerC = nil
	}

	defer stopTimer()

	flush := func() error {
		stopTimer()
		if ctx.Err() != nil {
			return fmt.Errorf("flush: %w", ctx.Err())
		}

		if len(batch) == 0 {
			return nil
		}

		value := batch
		batch = make([]int, 0, batchSize)
		if err := handle(ctx, value); err != nil {
			return fmt.Errorf("flush: %w", err)
		}

		return ctx.Err()
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("batch: %w", ctx.Err())
		case <-timerC:
			if err := flush(); err != nil {
				return fmt.Errorf("batch: %w", err)
			}
		case v, ok := <-input:
			if !ok {
				return flush()
			}

			batch = append(batch, v)
			if batchSize == len(batch) {
				if err := flush(); err != nil {
					return fmt.Errorf("batch: %w", err)
				}
			} else if len(batch) == 1 {
				timer = c.NewTimer(maxWait)
				timerC = timer.C()
			}
		}
	}

}
