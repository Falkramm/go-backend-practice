package c06

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

var (
	ErrInvalidWorkersNum = errors.New("invalid workers num")
)

// SumSquares returns the sum of squared integers parsed from input.
// TODO: Implement the pipeline and wait for its internal stages to finish.
func SumSquares(ctx context.Context, input <-chan string, workers int) (int, error) {
	if workers <= 0 {
		return 0, ErrInvalidWorkersNum
	}

	if ctx.Err() != nil {
		return 0, fmt.Errorf("sum squares: %w", ctx.Err())
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	results := make(chan int, workers)
	done := make(chan struct{})

	wg.Add(workers)

	var firstErr error
	var errorOnce sync.Once

	for range workers {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					errorOnce.Do(func() {
						firstErr = fmt.Errorf("sum squares: %w", ctx.Err())
						cancel()
					})
					return
				case v, ok := <-input:
					if !ok {
						return
					}
					i, err := strconv.Atoi(v)
					if err != nil {
						errorOnce.Do(func() {
							firstErr = fmt.Errorf("sum squares: %w", err)
							cancel()
						})
						return
					}

					i *= i
					select {
					case <-ctx.Done():
						errorOnce.Do(func() {
							firstErr = fmt.Errorf("sum squares: %w", ctx.Err())
							cancel()
						})
						return
					case results <- i:
					}

				}
			}
		}()
	}

	var res int
	go func() {
		defer close(done)
		for v := range results {
			res += v
		}
	}()

	wg.Wait()
	close(results)
	<-done
	if firstErr != nil {
		return 0, firstErr
	}

	if ctx.Err() != nil {
		return 0, fmt.Errorf("sum squares: %w", ctx.Err())
	}

	return res, nil
}
