package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/go-interview/catalog"
	"example.com/go-interview/orders"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	mode := flag.String("case", "catalog", "catalog or orders")
	addr := flag.String("addr", "127.0.0.1:8082", "HTTP address")
	flag.Parse()
	if err := run(*mode, *addr); err != nil {
		log.Fatal(err)
	}
}

func run(mode, addr string) error {
	if mode != "catalog" && mode != "orders" {
		return fmt.Errorf("unknown case %q", mode)
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://student@127.0.0.1:15432/go_interview_" + mode + "?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(6)
	db.SetMaxIdleConns(3)
	pingContext, cancelPing := context.WithTimeout(context.Background(), 3*time.Second)
	err = db.PingContext(pingContext)
	cancelPing()
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	var h http.Handler
	if mode == "catalog" {
		h = catalog.New(db)
	} else {
		h = orders.New(db)
	}
	server := &http.Server{
		Addr: addr, Handler: h,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       time.Minute,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	served := make(chan error, 1)
	go func() { served <- server.ListenAndServe() }()
	log.Printf("%s on http://%s", mode, addr)
	select {
	case err := <-served:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}
