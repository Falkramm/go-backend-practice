package c05

import (
	"context"
	"fmt"
	"sync"
)

type job struct {
	index  int
	result int
}

func ParallelMap(ctx context.Context, items []int, concurrency int, fn func(context.Context, int) (int, error)) ([]int, error) {
	if concurrency <= 0 {
		return nil, fmt.Errorf("parallel map: concurrency must be positive, got %d", concurrency)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("parallel map: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	res := make([]int, len(items))

	concurrency = min(concurrency, len(items))
	results := make(chan job, concurrency)
	jobs := make(chan job, concurrency)

	var firstErr error
	var errorOnce sync.Once

	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			for task := range jobs {
				if ctx.Err() != nil {
					return
				}
				value, err := fn(ctx, items[task.index])
				if err != nil {
					errorOnce.Do(func() {
						firstErr = err
						cancel()
					})
					return
				}
				task.result = value
				select {
				case <-ctx.Done():
					return
				case results <- task:
				}
			}
		}()
	}

	collected := make(chan struct{})
	go func() {
		defer close(collected)
		for task := range results {
			res[task.index] = task.result
		}
	}()

sendJobs:
	for i := range items {
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			break sendJobs
		case jobs <- job{index: i}:
		}
	}

	close(jobs)
	wg.Wait()
	close(results)
	<-collected

	if firstErr != nil {
		return nil, fmt.Errorf("parallel map: %w", firstErr)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("parallel map: %w", err)
	}

	return res, nil
}
