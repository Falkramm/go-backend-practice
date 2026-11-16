// Package testdb creates isolated schemas in the local practice PostgreSQL.
package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func parseTestConfig(raw string) (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig(raw)
	if err != nil {
		return nil, errors.New("invalid integration database configuration")
	}
	if !isPracticeEndpoint(cfg.Host, cfg.Port) {
		return nil, errors.New("integration database must use loopback on port 15432")
	}
	for _, fallback := range cfg.Fallbacks {
		if !isPracticeEndpoint(fallback.Host, fallback.Port) {
			return nil, errors.New("integration database fallbacks must use loopback on port 15432")
		}
	}
	return cfg, nil
}

func isPracticeEndpoint(host string, port uint16) bool {
	if port != 15432 {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Open applies schemaPath in a fresh schema and drops the schema at test cleanup.
func Open(t *testing.T, schemaPath string) *sql.DB {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		raw = os.Getenv("DATABASE_URL")
	}
	if raw == "" {
		t.Skip("integration requires TEST_DATABASE_URL or DATABASE_URL for loopback PostgreSQL on port 15432")
	}
	cfg, err := parseTestConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*cfg)
	admin.SetMaxOpenConns(1)
	admin.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		t.Fatalf("connect to local practice PostgreSQL: %v", err)
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		admin.Close()
		t.Fatalf("create test schema identifier: %v", err)
	}
	schema := "test_crud_" + hex.EncodeToString(random)
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+identifier); err != nil {
		admin.Close()
		t.Fatalf("create isolated test schema: %v", err)
	}
	var db *sql.DB
	t.Cleanup(func() {
		if db != nil {
			if err := db.Close(); err != nil {
				t.Errorf("close application database pool: %v", err)
			}
		}
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := admin.ExecContext(cleanupContext, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("drop generated test schema: %v", err)
		}
		if err := admin.Close(); err != nil {
			t.Errorf("close schema management pool: %v", err)
		}
	})
	appConfig := cfg.Copy()
	appConfig.RuntimeParams["search_path"] = schema
	db = stdlib.OpenDB(*appConfig)
	db.SetMaxOpenConns(6)
	db.SetMaxIdleConns(6)
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect application pool to isolated schema: %v", err)
	}
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read test schema.sql: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(schemaSQL)); err != nil {
		t.Fatalf("apply test schema.sql: %v", err)
	}
	return db
}
