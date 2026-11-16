package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/go-practice/04_crud/internal/testdb"
)

func lockProduct(t *testing.T, db *sql.DB, productID int64) (*sql.Tx, int) {
	t.Helper()
	tx, err := db.BeginTx(testContext(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	var id int64
	var pid int
	err = tx.QueryRowContext(testContext(t),
		"SELECT id, pg_backend_pid() FROM products WHERE id=$1 FOR UPDATE", productID).Scan(&id, &pid)
	if err != nil || id != productID {
		t.Fatalf("lock product %d: id=%d, error=%v", productID, id, err)
	}
	return tx, pid
}

func waitForBlockedQueries(t *testing.T, db *sql.DB, blockerPID, want int) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count, pid int
		err := db.QueryRowContext(ctx, `WITH RECURSIVE blocked AS (
			SELECT pid FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND $1=ANY(pg_blocking_pids(pid))
			UNION
			SELECT activity.pid FROM pg_stat_activity activity
			JOIN blocked ON blocked.pid=ANY(pg_blocking_pids(activity.pid))
			WHERE activity.datname=current_database() AND activity.wait_event_type='Lock'
		) SELECT count(*), COALESCE(min(pid),0) FROM blocked`, blockerPID).Scan(&count, &pid)
		if err != nil {
			t.Fatalf("observe PostgreSQL lock wait: %v", err)
		}
		if count >= want {
			return pid
		}
		select {
		case <-ctx.Done():
			t.Fatalf("blocked queries = %d; want %d: %v", count, want, ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForQueryCancellation(t *testing.T, db *sql.DB, pid int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := db.QueryRowContext(ctx,
			"SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid=$1 AND wait_event_type='Lock')", pid).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}
		if !blocked {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("SQL remained blocked after cancellation")
		case <-ticker.C:
		}
	}
}

type pendingResponse struct {
	response response
	err      error
}

func startConfirmation(server *httptest.Server, id int64) <-chan pendingResponse {
	done := make(chan pendingResponse, 1)
	go func() {
		res, err := perform(server, "POST", orderPath(id)+"/confirm", "")
		done <- pendingResponse{response: res, err: err}
	}()
	return done
}

func awaitResponse(t *testing.T, pending <-chan pendingResponse) response {
	t.Helper()
	select {
	case result := <-pending:
		if result.err != nil {
			t.Fatalf("confirm request: %v", result.err)
		}
		return result.response
	case <-time.After(requestTimeout):
		t.Fatal("confirmation did not finish")
		return response{}
	}
}

func TestTwoOrdersCompeteForLastUnit(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p := seedProduct(t, db, "last unit", 150, 1)
	first := seedOrder(t, db, itemInput{p, 1})
	second := seedOrder(t, db, itemInput{p, 1})
	server := startServer(t, db)
	blocker, pid := lockProduct(t, db, p)
	defer blocker.Rollback()
	one := startConfirmation(server, first.ID)
	two := startConfirmation(server, second.ID)
	waitForBlockedQueries(t, db, pid, 2)
	if err := blocker.Rollback(); err != nil {
		t.Fatal(err)
	}
	r1, r2 := awaitResponse(t, one), awaitResponse(t, two)
	if !((r1.status == 200 && r2.status == 409) || (r1.status == 409 && r2.status == 200)) {
		t.Fatalf("concurrent statuses = %d, %d; want one 200 and one 409", r1.status, r2.status)
	}
	winner, loser := first, second
	if r2.status == 200 {
		winner, loser = second, first
	}
	price := int64(150)
	winner.Status = "confirmed"
	winner.Items[0].PriceCents = &price
	expectOrder(t, storedOrder(t, db, winner.ID), winner)
	expectOrder(t, storedOrder(t, db, loser.ID), loser)
	expectStock(t, db, p, 0)
	expectCount(t, db, "SELECT count(*) FROM orders WHERE status='confirmed'", 1)
}

func TestConcurrentConfirmationDeductsOnce(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p := seedProduct(t, db, "one", 150, 5)
	want := seedOrder(t, db, itemInput{p, 2})
	server := startServer(t, db)
	blocker, pid := lockProduct(t, db, p)
	defer blocker.Rollback()
	one := startConfirmation(server, want.ID)
	two := startConfirmation(server, want.ID)
	waitForBlockedQueries(t, db, pid, 2)
	if err := blocker.Rollback(); err != nil {
		t.Fatal(err)
	}
	price := int64(150)
	want.Status = "confirmed"
	want.Items[0].PriceCents = &price
	expectOrder(t, decodeOrder(t, awaitResponse(t, one), 200), want)
	expectOrder(t, decodeOrder(t, awaitResponse(t, two), 200), want)
	expectOrder(t, storedOrder(t, db, want.ID), want)
	expectStock(t, db, p, 3)
}

func TestCanceledConfirmationRollsBackAndReleasesLocks(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p1 := seedProduct(t, db, "first", 150, 5)
	p2 := seedProduct(t, db, "second", 250, 5)
	original := seedOrder(t, db, itemInput{p1, 1}, itemInput{p2, 1})
	server := startServer(t, db)
	blocker, blockerPID := lockProduct(t, db, p2)
	defer blocker.Rollback()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", server.URL+orderPath(original.ID)+"/confirm", nil)
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() {
		res, err := server.Client().Do(req)
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
		t.Fatal(err)
	}
	expectOrder(t, storedOrder(t, db, original.ID), original)
	expectStock(t, db, p1, 5)
	expectStock(t, db, p2, 5)
	if err := db.PingContext(testContext(t)); err != nil {
		t.Fatalf("pool after cancellation: %v", err)
	}
	confirmed := decodeOrder(t, request(t, server, "POST", orderPath(original.ID)+"/confirm", ""), 200)
	if confirmed.Status != "confirmed" {
		t.Fatalf("confirmation after cancellation returned %#v", confirmed)
	}
	expectStock(t, db, p1, 4)
	expectStock(t, db, p2, 4)
}
