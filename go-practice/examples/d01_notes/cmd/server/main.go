package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"example.com/go-practice/examples/d01_notes/handler"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func newHTTPServer(h http.Handler) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       time.Minute,
	}
}

func runServer(ctx context.Context, listener net.Listener, h http.Handler, grace time.Duration) error {
	if listener == nil || h == nil || grace <= 0 {
		return errors.New("listener, handler and positive grace period are required")
	}

	var admission sync.RWMutex
	var active sync.WaitGroup
	stopping := false
	tracked := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admission.RLock()
		if stopping {
			admission.RUnlock()
			http.Error(w, "server is stopping", http.StatusServiceUnavailable)
			return
		}
		active.Add(1)
		admission.RUnlock()
		defer active.Done()
		h.ServeHTTP(w, r)
	})

	server := newHTTPServer(tracked)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()

	var result error
	select {
	case err := <-served:
		if !errors.Is(err, http.ErrServerClosed) {
			result = fmt.Errorf("serve HTTP: %w", err)
		}
		_ = server.Close()

	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), grace)
		result = server.Shutdown(shutdownContext)
		cancel()
		if result != nil {
			_ = server.Close()
		}
		err := <-served
		if result == nil && !errors.Is(err, http.ErrServerClosed) {
			result = fmt.Errorf("serve HTTP: %w", err)
		}
	}

	// Prevent later requests from adding work after Wait begins.
	admission.Lock()
	stopping = true
	admission.Unlock()
	active.Wait()
	return result
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8081"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(6)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingContext, cancelPing := context.WithTimeout(ctx, 3*time.Second)
	err = db.PingContext(pingContext)
	cancelPing()
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	h := handler.New(db)
	log.Printf("listening on %s", listener.Addr())
	return runServer(ctx, listener, h, 5*time.Second)
}

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
