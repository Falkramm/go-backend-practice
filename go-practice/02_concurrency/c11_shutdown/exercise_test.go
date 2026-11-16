package c11

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return ctx
}

func receive[T any](t *testing.T, results <-chan T) T {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(testTimeout):
		t.Fatal("operation did not finish within the test deadline")
		var zero T
		return zero
	}
}

func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(testTimeout):
		t.Fatal("handler did not reach the expected state")
	}
}

func newService(t *testing.T, workers, queueSize int, handle func(context.Context, int) error) *Service {
	t.Helper()
	service, err := NewService(workers, queueSize, handle)
	if err != nil || service == nil {
		t.Fatalf("NewService(%d, %d) = %v, %v; want a service and nil error", workers, queueSize, service, err)
	}
	return service
}

func startShutdown(service *Service, ctx context.Context) <-chan error {
	result := make(chan error, 1)
	go func() { result <- service.Shutdown(ctx) }()
	return result
}

func TestServiceRejectsInvalidConfiguration(t *testing.T) {
	handle := func(context.Context, int) error { return nil }
	cases := []struct {
		name      string
		workers   int
		queueSize int
		handle    func(context.Context, int) error
	}{
		{"zero_workers", 0, 1, handle},
		{"negative_workers", -1, 1, handle},
		{"negative_queue", 1, -1, handle},
		{"nil_handler", 1, 1, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewService(tc.workers, tc.queueSize, tc.handle)
			if err == nil || service != nil {
				t.Fatalf("NewService = %v, %v; want nil and an error", service, err)
			}
		})
	}
}

func TestServiceDrainsAcceptedWorkAndClosesAdmission(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var mu sync.Mutex
	var handled []int
	service := newService(t, 1, 2, func(ctx context.Context, id int) error {
		if id == 1 {
			close(started)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		mu.Lock()
		handled = append(handled, id)
		mu.Unlock()
		return nil
	})
	ctx := testContext(t)
	if err := service.Submit(ctx, 1); err != nil {
		t.Fatalf("Submit(1): %v", err)
	}
	awaitSignal(t, started)
	for _, id := range []int{2, 3} {
		if err := service.Submit(ctx, id); err != nil {
			t.Fatalf("Submit(%d): %v", id, err)
		}
	}

	shutdown := startShutdown(service, ctx)
	lateSubmit := make(chan error, 1)
	go func() { lateSubmit <- service.Submit(ctx, 4) }()
	if err := receive(t, lateSubmit); !errors.Is(err, ErrClosed) {
		t.Fatalf("Submit while draining = %v; want ErrClosed", err)
	}
	releaseOnce.Do(func() { close(release) })
	if err := receive(t, shutdown); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	mu.Lock()
	got := append([]int(nil), handled...)
	mu.Unlock()
	if !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatalf("handled IDs = %v; want [1 2 3]", got)
	}
}

func TestServiceEmptyAndRepeatedShutdown(t *testing.T) {
	service := newService(t, 2, 0, func(context.Context, int) error { return nil })
	ctx := testContext(t)
	for range 2 {
		if err := receive(t, startShutdown(service, ctx)); err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
	}
	if err := service.Submit(ctx, 1); !errors.Is(err, ErrClosed) {
		t.Fatalf("Submit after Shutdown = %v; want ErrClosed", err)
	}
}

func TestServiceShutdownDeadlineCancelsHandler(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	service := newService(t, 1, 1, func(ctx context.Context, id int) error {
		close(started)
		<-ctx.Done()
		close(finished)
		return ctx.Err()
	})
	if err := service.Submit(testContext(t), 1); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	awaitSignal(t, started)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := receive(t, startShutdown(service, ctx)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v; want context.DeadlineExceeded", err)
	}
	awaitSignal(t, finished)
}

func TestServiceShutdownCancellationCancelsHandler(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	service := newService(t, 1, 1, func(ctx context.Context, id int) error {
		close(started)
		<-ctx.Done()
		close(finished)
		return ctx.Err()
	})
	if err := service.Submit(testContext(t), 1); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	awaitSignal(t, started)
	ctx, cancel := context.WithCancel(context.Background())
	shutdown := startShutdown(service, ctx)
	cancel()
	if err := receive(t, shutdown); !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown = %v; want context.Canceled", err)
	}
	awaitSignal(t, finished)
}

func TestServiceCanceledSubmitDoesNotEnqueueWork(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	var mu sync.Mutex
	var handled []int
	service := newService(t, 1, 1, func(ctx context.Context, id int) error {
		if id == 1 {
			close(started)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		mu.Lock()
		handled = append(handled, id)
		mu.Unlock()
		return nil
	})
	ctx := testContext(t)
	if err := service.Submit(ctx, 1); err != nil {
		t.Fatalf("Submit(1): %v", err)
	}
	awaitSignal(t, started)
	if err := service.Submit(ctx, 2); err != nil {
		t.Fatalf("Submit(2): %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := service.Submit(canceled, 3); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Submit = %v; want context.Canceled", err)
	}
	// Canceling shutdown also releases the handler, so deferred release stays single-owner.
	shutdownCtx, stop := context.WithCancel(context.Background())
	shutdown := startShutdown(service, shutdownCtx)
	stop()
	if err := receive(t, shutdown); !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown = %v; want context.Canceled", err)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, id := range handled {
		if id == 3 {
			t.Fatal("a canceled Submit enqueued work")
		}
	}
}

func TestServiceSubmitShutdownRacePreservesAcceptedIDs(t *testing.T) {
	var mu sync.Mutex
	handled := make(map[int]int)
	service := newService(t, 4, 32, func(ctx context.Context, id int) error {
		mu.Lock()
		handled[id]++
		mu.Unlock()
		return nil
	})
	ctx := testContext(t)
	if err := service.Submit(ctx, 0); err != nil {
		t.Fatalf("initial Submit: %v", err)
	}
	type submission struct {
		id  int
		err error
	}
	const callers = 32
	start := make(chan struct{})
	results := make(chan submission, callers)
	for id := 1; id <= callers; id++ {
		go func() {
			<-start
			results <- submission{id: id, err: service.Submit(ctx, id)}
		}()
	}
	shutdown := make(chan error, 1)
	go func() {
		<-start
		shutdown <- service.Shutdown(ctx)
	}()
	close(start)
	accepted := map[int]int{0: 1}
	for range callers {
		result := receive(t, results)
		if result.err == nil {
			accepted[result.id]++
		} else if !errors.Is(result.err, ErrClosed) {
			t.Fatalf("Submit(%d) = %v; want nil or ErrClosed", result.id, result.err)
		}
	}
	if err := receive(t, shutdown); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(handled, accepted) {
		t.Fatalf("handled IDs = %v; accepted IDs = %v", handled, accepted)
	}
}

func TestServiceSIGTERM(t *testing.T) {
	directory := t.TempDir()
	journalPath := filepath.Join(directory, "handled.txt")
	logPath := filepath.Join(directory, "process.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logFile.Close() })
	readyReader, readyWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = readyReader.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestServiceSIGTERMHelper$")
	cmd.Env = append(os.Environ(), "GO_C11_HELPER=1", "GO_C11_JOURNAL="+journalPath)
	cmd.ExtraFiles = []*os.File{readyWriter}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		cancel()
		_ = readyWriter.Close()
		t.Fatal(err)
	}
	_ = readyWriter.Close()
	var waitErr error
	exited := make(chan struct{})
	go func() {
		waitErr = cmd.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-exited:
		default:
			_ = cmd.Process.Kill()
			select {
			case <-exited:
			case <-time.After(testTimeout):
				t.Error("helper process did not exit after cleanup")
			}
		}
	})
	readiness := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(readyReader).ReadString('\n')
		if err == nil && line != "ready\n" {
			err = fmt.Errorf("unexpected readiness %q", line)
		}
		readiness <- err
	}()
	if err := receive(t, readiness); err != nil {
		output, _ := os.ReadFile(logPath)
		t.Fatalf("helper readiness: %v\n%s", err, output)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-ctx.Done():
		t.Fatalf("helper did not finish after SIGTERM: %v", ctx.Err())
	}
	if waitErr != nil {
		output, _ := os.ReadFile(logPath)
		t.Fatalf("helper exit: %v\n%s", waitErr, output)
	}
	journal, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	var handled []int
	for _, field := range strings.Fields(string(journal)) {
		id, err := strconv.Atoi(field)
		if err != nil {
			t.Fatal(err)
		}
		handled = append(handled, id)
	}
	if !reflect.DeepEqual(handled, []int{1, 2, 3}) {
		t.Fatalf("helper journal = %v; want [1 2 3]", handled)
	}
}

func TestServiceSIGTERMHelper(t *testing.T) {
	if os.Getenv("GO_C11_HELPER") != "1" {
		return
	}
	stopContext, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	ready := os.NewFile(3, "ready")
	if ready == nil {
		t.Fatal("missing readiness pipe")
	}
	defer ready.Close()
	release := make(chan struct{})
	started := make(chan struct{})
	service := newService(t, 1, 2, func(ctx context.Context, id int) error {
		if id == 1 {
			close(started)
		}
		select {
		case <-release:
		case <-ctx.Done():
			return ctx.Err()
		}
		file, err := os.OpenFile(os.Getenv("GO_C11_JOURNAL"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, writeErr := fmt.Fprintln(file, id)
		closeErr := file.Close()
		return errors.Join(writeErr, closeErr)
	})
	ctx := testContext(t)
	if err := service.Submit(ctx, 1); err != nil {
		t.Fatal(err)
	}
	awaitSignal(t, started)
	for _, id := range []int{2, 3} {
		if err := service.Submit(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ready.WriteString("ready\n"); err != nil {
		t.Fatal(err)
	}
	_ = ready.Close()
	<-stopContext.Done()
	grace, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	shutdown := startShutdown(service, grace)
	if err := service.Submit(grace, 99); !errors.Is(err, ErrClosed) {
		t.Fatalf("Submit after SIGTERM = %v; want ErrClosed", err)
	}
	close(release)
	if err := receive(t, shutdown); err != nil {
		t.Fatalf("Shutdown after SIGTERM: %v", err)
	}
}
