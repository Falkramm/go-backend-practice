package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/go-practice/04_crud/d01_notes/internal/testdb"
)

func lockNote(t *testing.T, ctx context.Context, db *sql.DB, id int64) (*sql.Tx, int) {
	t.Helper()
	blocker, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = blocker.Rollback() })
	var lockedID int64
	var blockerPID int
	err = blocker.QueryRowContext(ctx, "SELECT id, pg_backend_pid() FROM notes WHERE id=$1 FOR UPDATE", id).
		Scan(&lockedID, &blockerPID)
	if err != nil || lockedID != id {
		t.Fatalf("lock fixture row: id=%d, error=%v", lockedID, err)
	}
	return blocker, blockerPID
}

func waitForBlockedQueries(t *testing.T, db *sql.DB, blockerPID, count int) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting, pid int
		err := db.QueryRowContext(ctx, `WITH RECURSIVE blocked AS (
			SELECT pid FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND $1=ANY(pg_blocking_pids(pid))
			UNION
			SELECT activity.pid FROM pg_stat_activity activity
			JOIN blocked ON blocked.pid=ANY(pg_blocking_pids(activity.pid))
			WHERE activity.datname=current_database() AND activity.wait_event_type='Lock'
		) SELECT count(*), COALESCE(min(pid), 0) FROM blocked`, blockerPID).Scan(&waiting, &pid)
		if err != nil {
			t.Fatalf("observe blocked PostgreSQL queries: %v", err)
		}
		if waiting >= count {
			return pid
		}
		select {
		case <-ctx.Done():
			t.Fatalf("blocked queries = %d; want %d: %v", waiting, count, ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForQueryCancellation(t *testing.T, db *sql.DB, waitingPID int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var stillWaiting bool
		err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid=$1 AND wait_event_type='Lock')", waitingPID).Scan(&stillWaiting)
		if err != nil {
			t.Fatalf("observe server-side query cancellation: %v", err)
		}
		if !stillWaiting {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("canceled PostgreSQL query remained blocked")
		case <-ticker.C:
		}
	}
}

func TestRequestCancellationStopsSQLAndPreservesNote(t *testing.T) {
	db := testdb.Open(t)
	original := seedNote(t, db)
	server := startServer(t, db)
	blocker, blockerPID := lockNote(t, testContext(t), db, original.ID)
	defer blocker.Rollback()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := "/notes/" + strconv.FormatInt(original.ID, 10)
	req, err := http.NewRequestWithContext(ctx, "PUT", server.URL+path,
		strings.NewReader(`{"title":"must stay unapplied","body":null,"pinned":false}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := server.Client()
	client.Timeout = requestTimeout
	finished := make(chan error, 1)
	go func() {
		res, err := client.Do(req)
		if res != nil {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
		}
		finished <- err
	}()
	waitingPID := waitForBlockedQueries(t, db, blockerPID, 1)
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("client error = %v; want context.Canceled", err)
		}
	case <-time.After(requestTimeout):
		t.Fatal("HTTP request did not finish after cancellation")
	}
	waitForQueryCancellation(t, db, waitingPID)
	if err := blocker.Rollback(); err != nil {
		t.Fatalf("release blocker: %v", err)
	}
	if err := db.PingContext(testContext(t)); err != nil {
		t.Fatalf("pool after cancellation: %v", err)
	}
	expectStoredNote(t, db, original)
	expectNote(t, decodeNote(t, request(t, server, "GET", path, "", ""), http.StatusOK), original)
}

func TestConcurrentPatchesPreserveDisjointChanges(t *testing.T) {
	db := testdb.Open(t)
	want := seedNote(t, db)
	server := startServer(t, db)
	blocker, blockerPID := lockNote(t, testContext(t), db, want.ID)
	defer blocker.Rollback()
	path := "/notes/" + strconv.FormatInt(want.ID, 10)
	client := server.Client()
	client.Timeout = requestTimeout
	type result struct {
		response response
		err      error
	}
	results := make(chan result, 2)
	ctx := testContext(t)
	for _, payload := range []string{`{"title":"concurrent title"}`, `{"pinned":false}`} {
		req, err := http.NewRequestWithContext(ctx, "PATCH", server.URL+path, strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		go func(req *http.Request) {
			res, err := client.Do(req)
			if err != nil {
				results <- result{err: err}
				return
			}
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			results <- result{response: response{res.StatusCode, res.Header.Clone(), body}, err: err}
		}(req)
	}
	waitForBlockedQueries(t, db, blockerPID, 2)
	if err := blocker.Rollback(); err != nil {
		t.Fatalf("release blocker: %v", err)
	}
	for range 2 {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("concurrent PATCH: %v", result.err)
			}
			decodeNote(t, result.response, http.StatusOK)
		case <-time.After(requestTimeout):
			t.Fatal("concurrent PATCH did not finish")
		}
	}
	want.Title, want.Pinned = "concurrent title", false
	expectStoredNote(t, db, want)
	expectNote(t, decodeNote(t, request(t, server, "GET", path, "", ""), http.StatusOK), want)
}
