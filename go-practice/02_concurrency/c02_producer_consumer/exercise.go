package c02

import (
	"context"
	"fmt"
)

func Produce(ctx context.Context, n int, out chan<- int) error {
	defer close(out)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return fmt.Errorf("product: %w", ctx.Err())
	}
	if n <= 0 {
		return nil
	}

	for v := range n {
		select {
		case <-ctx.Done():
			return fmt.Errorf("product: %w", ctx.Err())
		case out <- v + 1:
		}
	}

	return nil
}

func Consume(ctx context.Context, in <-chan int) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var sum int
	if ctx.Err() != nil {
		return sum, fmt.Errorf("consume: %w", ctx.Err())
	}
	for {
		select {
		case <-ctx.Done():
			return sum, fmt.Errorf("consume: %w", ctx.Err())
		case v, ok := <-in:
			if !ok {
				return sum, nil
			}
			sum += v
		}
	}
}
