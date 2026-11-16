package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

type serverRun struct {
	listener net.Listener
	url      string
	stop     context.CancelFunc
	finished chan struct{}
	err      error
}

func startServer(t *testing.T, h http.Handler, grace time.Duration) *serverRun {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &serverRun{
		listener: listener,
		url:      "http://" + listener.Addr().String(),
		stop:     cancel,
		finished: make(chan struct{}),
	}
	go func() {
		run.err = runServer(ctx, listener, h, grace)
		close(run.finished)
	}()
	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
		select {
		case <-run.finished:
		case <-time.After(testTimeout):
			t.Error("runServer did not finish during cleanup")
		}
	})
	return run
}

type response struct {
	status int
	body   string
	err    error
}

func startRequest(t *testing.T, endpoint string) <-chan response {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	transport := &http.Transport{DisableKeepAlives: true}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Timeout: testTimeout, Transport: transport}
	result := make(chan response, 1)
	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			result <- response{err: err}
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			result <- response{err: err}
			return
		}
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		result <- response{status: resp.StatusCode, body: string(body), err: errors.Join(readErr, closeErr)}
	}()
	return result
}

func awaitHandler(t *testing.T, started <-chan struct{}, run *serverRun) {
	t.Helper()
	select {
	case <-started:
	case <-run.finished:
		t.Fatalf("runServer returned before serving a request: %v", run.err)
	case <-time.After(testTimeout):
		t.Fatal("the HTTP handler did not start")
	}
}

func awaitAdmissionClosed(t *testing.T, run *serverRun) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(testTimeout)
	defer deadline.Stop()
	for {
		conn, err := net.DialTimeout("tcp", run.listener.Addr().String(), 20*time.Millisecond)
		if err != nil {
			return
		}
		_ = conn.Close()
		select {
		case <-run.finished:
			t.Fatalf("runServer returned while the active handler was held: %v", run.err)
		case <-deadline.C:
			t.Fatal("shutdown did not close admission")
		case <-ticker.C:
		}
	}
}

func TestHTTPServerHasFiniteTimeouts(t *testing.T) {
	srv := newHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	if srv == nil || srv.Handler == nil {
		t.Fatal("newHTTPServer must return a server with the supplied handler")
	}
	for name, timeout := range map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	} {
		if timeout <= 0 {
			t.Errorf("%s = %s, want a positive timeout", name, timeout)
		}
	}
}

func TestRunServerDrainsActiveRequest(t *testing.T) {
	started := make(chan struct{})
	var handlerContext context.Context
	release := make(chan struct{})
	released := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerContext = r.Context()
		close(started)
		select {
		case <-release:
			_, _ = io.WriteString(w, "done")
		case <-r.Context().Done():
		}
	})
	run := startServer(t, h, time.Second)
	t.Cleanup(func() {
		if !released {
			close(release)
		}
	})
	request := startRequest(t, run.url)
	awaitHandler(t, started, run)
	run.stop()
	awaitAdmissionClosed(t, run)
	if err := handlerContext.Err(); err != nil {
		t.Fatalf("the active request was canceled during its grace period: %v", err)
	}
	select {
	case <-run.finished:
		t.Fatalf("runServer returned before the accepted request finished: %v", run.err)
	default:
	}
	close(release)
	released = true
	select {
	case got := <-request:
		if got.err != nil || got.status != http.StatusOK || got.body != "done" {
			t.Fatalf("accepted request = %+v, want HTTP 200 and done", got)
		}
	case <-time.After(testTimeout):
		t.Fatal("accepted request did not finish")
	}
	select {
	case <-run.finished:
		if run.err != nil {
			t.Fatalf("graceful runServer returned %v, want nil", run.err)
		}
	case <-time.After(testTimeout):
		t.Fatal("graceful shutdown did not finish")
	}
}

func TestRunServerDeadlineCancelsActiveRequest(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	released := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(canceled)
		<-release
		close(finished)
	})
	run := startServer(t, h, 25*time.Millisecond)
	t.Cleanup(func() {
		if !released {
			close(release)
		}
	})
	request := startRequest(t, run.url)
	awaitHandler(t, started, run)
	run.stop()
	select {
	case <-canceled:
	case <-run.finished:
		t.Fatalf("runServer returned before canceling its active request: %v", run.err)
	case <-time.After(testTimeout):
		t.Fatal("the active request context was not canceled")
	}
	select {
	case <-run.finished:
		t.Fatalf("runServer returned while handler cleanup was held: %v", run.err)
	default:
	}
	close(release)
	released = true
	select {
	case <-run.finished:
		if !errors.Is(run.err, context.DeadlineExceeded) {
			t.Fatalf("runServer returned %v, want a shutdown deadline error", run.err)
		}
	case <-time.After(testTimeout):
		t.Fatal("shutdown did not respect its budget")
	}
	select {
	case <-finished:
	case <-time.After(testTimeout):
		t.Fatal("the active request did not finish cleanup")
	}
	select {
	case <-request:
	case <-time.After(testTimeout):
		t.Fatal("the HTTP client did not finish after forced close")
	}
}
