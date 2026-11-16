package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var errNotImplemented = errors.New("not implemented")

func newHTTPServer(h http.Handler) *http.Server {
	// TODO: Set ReadHeaderTimeout, ReadTimeout, WriteTimeout and IdleTimeout.
	return &http.Server{Handler: h}
}

func runServer(ctx context.Context, listener net.Listener, h http.Handler, grace time.Duration) error {
	// TODO: Serve HTTP, drain after cancellation, then force-close on deadline.
	return errNotImplemented
}

func main() {
	// TODO: Read DATABASE_URL and HTTP_ADDR (default 127.0.0.1:8080).
	// TODO: Open sql.DB, PingContext and configure its connection pool.
	// TODO: Build handler.New(db).
	// TODO: net.Listen, signal.NotifyContext for SIGINT/SIGTERM and runServer.
	// TODO: Close the database after runServer finishes.
}
