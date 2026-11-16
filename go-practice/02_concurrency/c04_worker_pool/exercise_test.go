package c04

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

type poolResult struct {
	pool *Pool
	err  error
}

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

func createPool(ctx context.Context, workers, queueSize int, fn func(context.Context, int) (int, error)) <-chan poolResult {
	returned := make(chan poolResult, 1)
	go func() {
		pool, err := NewPool(ctx, workers, queueSize, fn)
		returned <- poolResult{pool: pool, err: err}
	}()
	return returned
}

func newPool(t *testing.T, ctx context.Context, workers, queueSize int, fn func(context.Context, int) (int, error)) *Pool {
	t.Helper()
	got := waitFor(t, createPool(ctx, workers, queueSize, fn))
	if got.err != nil || got.pool == nil {
		t.Fatalf("NewPool returned (%v, %v), want an initialized pool", got.pool, got.err)
	}
	if got.pool.Jobs == nil || got.pool.Results == nil || got.pool.Done == nil {
		t.Fatal("NewPool returned a nil public channel")
	}
	if capacity := cap(got.pool.Jobs); capacity != queueSize {
		t.Fatalf("job queue capacity = %d, want %d", capacity, queueSize)
	}
	return got.pool
}

func startSubmit(ctx context.Context, jobs chan<- Job, job Job) <-chan error {
	finished := make(chan error, 1)
	go func() { finished <- Submit(ctx, jobs, job) }()
	return finished
}

func submit(t *testing.T, ctx context.Context, jobs chan<- Job, job Job) {
	t.Helper()
	if err := waitFor(t, startSubmit(ctx, jobs, job)); err != nil {
		t.Fatalf("Submit(%+v) returned %v", job, err)
	}
}

func sendJob(t *testing.T, ctx context.Context, jobs chan<- Job, job Job) {
	t.Helper()
	select {
	case jobs <- job:
	case <-ctx.Done():
		t.Fatalf("sending job %+v failed: %v", job, ctx.Err())
	case <-time.After(testTimeout):
		t.Fatalf("job %+v was not accepted before the timeout", job)
	}
}

func collectResults(t *testing.T, results <-chan Result) <-chan []Result {
	t.Helper()
	finished := make(chan []Result, 1)
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		var values []Result
		for {
			select {
			case value, ok := <-results:
				if !ok {
					finished <- values
					return
				}
				values = append(values, value)
			case <-stop:
				return
			}
		}
	}()
	return finished
}

func requireResultsClosed(t *testing.T, results <-chan Result) {
	t.Helper()
	for {
		select {
		case _, ok := <-results:
			if !ok {
				return
			}
		default:
			t.Fatal("Done closed before Results was closed")
		}
	}
}

func TestPoolResultsAndJobError(t *testing.T) {
	for _, queueSize := range []int{0, 2} {
		t.Run(strconv.Itoa(queueSize), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("job failure")
			pool := newPool(t, ctx, 2, queueSize, func(ctx context.Context, value int) (int, error) {
				if value < 0 {
					return 37, failure
				}
				return value * 2, nil
			})
			collected := collectResults(t, pool.Results)
			for _, job := range []Job{{ID: 10, Value: 2}, {ID: 20, Value: -1}, {ID: 30, Value: 0}} {
				sendJob(t, ctx, pool.Jobs, job)
			}
			close(pool.Jobs)
			got := waitFor(t, collected)
			if len(got) != 3 {
				t.Fatalf("results = %v, want exactly three results", got)
			}
			seen := make(map[int]bool)
			for _, result := range got {
				if seen[result.ID] {
					t.Fatalf("duplicate result ID %d", result.ID)
				}
				seen[result.ID] = true
				switch result.ID {
				case 10:
					if result.Err != nil || result.Value != 4 {
						t.Fatalf("result = %+v, want ID 10, value 4, nil error", result)
					}
				case 20:
					if result.Value != 37 || !errors.Is(result.Err, failure) {
						t.Fatalf("result = %+v, want ID 20, value 37, and the original failure", result)
					}
				case 30:
					if result.Err != nil || result.Value != 0 {
						t.Fatalf("result = %+v, want ID 30, value 0, nil error", result)
					}
				default:
					t.Fatalf("unexpected result ID %d", result.ID)
				}
			}
			waitFor(t, pool.Done)
			requireResultsClosed(t, pool.Results)
		})
	}
}

func TestPoolEmptyInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	pool := newPool(t, ctx, 3, 1, func(context.Context, int) (int, error) {
		calls.Add(1)
		return 0, nil
	})
	collected := collectResults(t, pool.Results)
	close(pool.Jobs)
	if got := waitFor(t, collected); len(got) != 0 {
		t.Fatalf("results = %v, want an empty result", got)
	}
	waitFor(t, pool.Done)
	requireResultsClosed(t, pool.Results)
	if calls.Load() != 0 {
		t.Fatalf("handler calls = %d, want 0", calls.Load())
	}
}

func TestPoolInvalidParameters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		workers int
		queue   int
	}{
		{name: "zero workers", workers: 0, queue: 1},
		{name: "negative workers", workers: -1, queue: 1},
		{name: "negative queue", workers: 1, queue: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := waitFor(t, createPool(context.Background(), tc.workers, tc.queue, func(context.Context, int) (int, error) {
				return 0, nil
			}))
			if got.pool != nil || got.err == nil {
				t.Fatalf("NewPool returned (%v, %v), want (nil, error)", got.pool, got.err)
			}
		})
	}
}

func TestNewPoolCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := waitFor(t, createPool(ctx, 1, 1, func(context.Context, int) (int, error) {
		return 0, nil
	}))
	if got.pool != nil || !errors.Is(got.err, ctx.Err()) {
		t.Fatalf("NewPool returned (%v, %v), want (nil, error matching %v)", got.pool, got.err, ctx.Err())
	}
}

func TestSubmitCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	jobs := make(chan Job, 1)
	if err := waitFor(t, startSubmit(ctx, jobs, Job{ID: 1})); !errors.Is(err, ctx.Err()) {
		t.Fatalf("Submit returned %v, want an error matching %v", err, ctx.Err())
	}
	if len(jobs) != 0 {
		t.Fatal("Submit enqueued a job with an already canceled context")
	}
	jobs <- Job{ID: 2}
	close(jobs)
}

func TestPoolBackpressureAndConcurrency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const workers, queueSize = 2, 2
	started := make(chan int, workers+queueSize)
	allowReturn := make(chan struct{})
	release := sync.OnceFunc(func() { close(allowReturn) })
	defer release()
	var active, peak atomic.Int32
	pool := newPool(t, ctx, workers, queueSize, func(ctx context.Context, value int) (int, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			previous := peak.Load()
			if current <= previous || peak.CompareAndSwap(previous, current) {
				break
			}
		}
		started <- value
		select {
		case <-allowReturn:
			return value * 10, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})
	collected := collectResults(t, pool.Results)
	for _, value := range []int{1, 2} {
		sendJob(t, ctx, pool.Jobs, Job{ID: value, Value: value})
	}
	waitFor(t, started)
	waitFor(t, started)
	for _, value := range []int{3, 4} {
		sendJob(t, ctx, pool.Jobs, Job{ID: value, Value: value})
	}
	if queued := len(pool.Jobs); queued != queueSize {
		t.Fatalf("queued jobs = %d, want %d while all workers are held", queued, queueSize)
	}
	select {
	case pool.Jobs <- Job{ID: 5, Value: 5}:
		t.Fatal("full queue accepted another job")
	default:
	}
	close(pool.Jobs)
	release()
	got := waitFor(t, collected)
	if len(got) != workers+queueSize {
		t.Fatalf("results = %v, want four accepted jobs", got)
	}
	seen := make(map[int]bool)
	for _, result := range got {
		if result.ID < 1 || result.ID > 4 || seen[result.ID] || result.Value != result.ID*10 || result.Err != nil {
			t.Fatalf("unexpected or repeated result %+v", result)
		}
		seen[result.ID] = true
	}
	waitFor(t, pool.Done)
	if peak.Load() > workers || active.Load() != 0 {
		t.Fatalf("peak/active handlers = %d/%d, want peak <= %d and active 0", peak.Load(), active.Load(), workers)
	}
}

func TestPoolCancellationWaitsForWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 4)
	canceled := make(chan struct{}, 4)
	allowReturn := make(chan struct{})
	release := sync.OnceFunc(func() { close(allowReturn) })
	defer release()
	var active atomic.Int32
	pool := newPool(t, ctx, 2, 2, func(ctx context.Context, value int) (int, error) {
		active.Add(1)
		defer active.Add(-1)
		started <- struct{}{}
		<-ctx.Done()
		canceled <- struct{}{}
		<-allowReturn
		return 0, ctx.Err()
	})
	collected := collectResults(t, pool.Results)
	for _, value := range []int{1, 2} {
		sendJob(t, ctx, pool.Jobs, Job{ID: value, Value: value})
	}
	waitFor(t, started)
	waitFor(t, started)
	for _, value := range []int{3, 4} {
		sendJob(t, ctx, pool.Jobs, Job{ID: value, Value: value})
	}
	if len(pool.Jobs) != 2 {
		t.Fatal("expected a full queue before cancellation")
	}
	cancel()
	waitFor(t, canceled)
	waitFor(t, canceled)
	select {
	case <-pool.Done:
		t.Fatal("Done closed while canceled handlers were held")
	default:
	}
	release()
	waitFor(t, pool.Done)
	if active.Load() != 0 {
		t.Fatalf("active handlers after Done = %d, want 0", active.Load())
	}
	requireResultsClosed(t, pool.Results)
	waitFor(t, collected)
	close(pool.Jobs)
}

func TestSubmitSuccess(t *testing.T) {
	for _, queueSize := range []int{0, 1} {
		t.Run(strconv.Itoa(queueSize), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			jobs := make(chan Job, queueSize)
			delivered := make(chan Job, 1)
			go func() {
				select {
				case job := <-jobs:
					delivered <- job
				case <-ctx.Done():
				}
			}()
			want := Job{ID: 7, Value: 42}
			if err := waitFor(t, startSubmit(ctx, jobs, want)); err != nil {
				t.Fatalf("Submit returned %v, want nil", err)
			}
			if got := waitFor(t, delivered); got != want {
				t.Fatalf("delivered job = %+v, want %+v", got, want)
			}
			close(jobs)
		})
	}
}

func TestSubmitCanceledWhileQueueFull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := make(chan Job, 1)
	defer close(jobs)
	original := Job{ID: 1, Value: 10}
	jobs <- original
	submitting := make(chan struct{}, 1)
	submitted := make(chan error, 1)
	go func() {
		submitting <- struct{}{}
		submitted <- Submit(ctx, jobs, Job{ID: 2, Value: 20})
	}()
	waitFor(t, submitting)
	cancel()
	if err := waitFor(t, submitted); !errors.Is(err, ctx.Err()) {
		t.Fatalf("Submit returned %v, want an error matching %v", err, ctx.Err())
	}
	if len(jobs) != 1 {
		t.Fatalf("queued jobs = %d, want 1", len(jobs))
	}
	if got := <-jobs; got != original {
		t.Fatalf("queued job = %+v, want the original %+v", got, original)
	}
}

func TestIntegrationPoolAndSubmit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool := newPool(t, ctx, 2, 1, func(ctx context.Context, value int) (int, error) {
		return value * 2, nil
	})
	collected := collectResults(t, pool.Results)
	for _, job := range []Job{{ID: 1, Value: 5}, {ID: 2, Value: 7}} {
		submit(t, ctx, pool.Jobs, job)
	}
	close(pool.Jobs)
	got := waitFor(t, collected)
	if len(got) != 2 {
		t.Fatalf("results = %v, want two results", got)
	}
	want := map[int]int{1: 10, 2: 14}
	for _, result := range got {
		value, ok := want[result.ID]
		if !ok || result.Value != value || result.Err != nil {
			t.Fatalf("unexpected result %+v", result)
		}
		delete(want, result.ID)
	}
	waitFor(t, pool.Done)
	requireResultsClosed(t, pool.Results)
}

func TestPoolRepeatedIDs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool := newPool(t, ctx, 2, 1, func(ctx context.Context, value int) (int, error) {
		return value * 2, nil
	})
	collected := collectResults(t, pool.Results)
	for _, value := range []int{5, -2} {
		sendJob(t, ctx, pool.Jobs, Job{ID: 7, Value: value})
	}
	close(pool.Jobs)
	got := waitFor(t, collected)
	if len(got) != 2 {
		t.Fatalf("results = %v, want one result for each of the two jobs", got)
	}
	want := map[int]bool{10: true, -4: true}
	for _, result := range got {
		if result.ID != 7 || result.Err != nil || !want[result.Value] {
			t.Fatalf("unexpected result %+v", result)
		}
		delete(want, result.Value)
	}
	waitFor(t, pool.Done)
	requireResultsClosed(t, pool.Results)
}

func TestPoolCancellationWithoutReadingResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	callbackStarted := make(chan struct{}, 1)
	cancellationObserved := make(chan struct{}, 1)
	allowReturn := make(chan struct{})
	release := sync.OnceFunc(func() { close(allowReturn) })
	defer release()
	pool := newPool(t, ctx, 1, 0, func(ctx context.Context, value int) (int, error) {
		if value >= 0 {
			return value, nil
		}
		callbackStarted <- struct{}{}
		<-ctx.Done()
		cancellationObserved <- struct{}{}
		<-allowReturn
		return 0, ctx.Err()
	})
	capacity := cap(pool.Results)
	for value := range capacity {
		sendJob(t, ctx, pool.Jobs, Job{ID: value, Value: value})
	}
	sendJob(t, ctx, pool.Jobs, Job{ID: capacity, Value: -1})
	waitFor(t, callbackStarted)
	if len(pool.Results) != capacity {
		t.Fatalf("buffered results = %d, want capacity %d before cancellation", len(pool.Results), capacity)
	}
	cancel()
	waitFor(t, cancellationObserved)
	release()
	waitFor(t, pool.Done)
	requireResultsClosed(t, pool.Results)
	close(pool.Jobs)
}
