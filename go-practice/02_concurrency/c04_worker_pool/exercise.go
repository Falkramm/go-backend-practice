package c04

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrIllegalNumOfWorkers = errors.New("illegal num of workers")
	ErrIllegalQueueSize    = errors.New("illegal queue size")
)

type Job struct {
	ID    int
	Value int
}

type Result struct {
	ID    int
	Value int
	Err   error
}

type Pool struct {
	Jobs    chan<- Job
	Results <-chan Result
	Done    <-chan struct{}
}

func NewPool(ctx context.Context, workers, queueSize int, fn func(context.Context, int) (int, error)) (*Pool, error) {
	if workers <= 0 {
		return nil, ErrIllegalNumOfWorkers
	}

	if queueSize < 0 {
		return nil, ErrIllegalQueueSize
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("new pool: %w", ctx.Err())
	}

	jobs := make(chan Job, queueSize)
	done := make(chan struct{}, 1)
	results := make(chan Result)
	var wg sync.WaitGroup
	pool := &Pool{
		Jobs:    jobs,
		Results: results,
		Done:    done,
	}

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					jobResult, err := fn(ctx, job.Value)
					result := Result{
						ID:    job.ID,
						Value: jobResult,
						Err:   err,
					}

					select {
					case <-ctx.Done():
						return
					case results <- result:
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
		close(done)
	}()
	return pool, nil
}

func Submit(ctx context.Context, jobs chan<- Job, job Job) error {
	if ctx.Err() != nil {
		return fmt.Errorf("submit: %w", ctx.Err())
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("submit: %w", ctx.Err())
	case jobs <- job:
		return nil
	}
}
