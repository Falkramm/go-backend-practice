package testdb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
)

func TestParseTestConfigAllowsOnlyPracticeEndpoint(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		ok   bool
	}{
		{name: "IPv4 loopback", dsn: "postgres://student@127.0.0.1:15432/go_practice?sslmode=disable", ok: true},
		{name: "localhost", dsn: "postgres://student@localhost:15432/go_practice?sslmode=disable", ok: true},
		{name: "IPv6 loopback", dsn: "postgres://student@[::1]:15432/go_practice?sslmode=disable", ok: true},
		{name: "remote host", dsn: "postgres://student@192.0.2.1:15432/go_practice?sslmode=disable"},
		{name: "default port", dsn: "postgres://student@127.0.0.1:5432/go_practice?sslmode=disable"},
		{name: "Unix socket", dsn: "host=/tmp port=15432 user=student dbname=go_practice sslmode=disable"},
		{name: "remote fallback", dsn: "host=127.0.0.1,192.0.2.1 port=15432,15432 user=student dbname=go_practice sslmode=disable"},
		{name: "wrong fallback port", dsn: "host=127.0.0.1,localhost port=15432,5432 user=student dbname=go_practice sslmode=disable"},
		{name: "invalid DSN", dsn: "postgres://[invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseTestConfig(tc.dsn)
			if tc.ok {
				if err != nil || cfg == nil {
					t.Fatalf("allowed endpoint rejected: %v", err)
				}
				return
			}
			if err == nil || cfg != nil {
				t.Fatal("unsafe or invalid endpoint accepted")
			}
		})
	}
}

func TestOpenCreatesIsolatedSchemasAndCleansUp(t *testing.T) {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		raw = os.Getenv("DATABASE_URL")
	}
	if raw == "" {
		t.Skip("integration requires a local practice database DSN")
	}
	cfg, err := parseTestConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	schemaFile := filepath.Join(t.TempDir(), "schema.sql")
	if err := os.WriteFile(schemaFile, []byte("CREATE TABLE fixture (id integer PRIMARY KEY);"), 0600); err != nil {
		t.Fatal(err)
	}
	var schemas []string
	for _, name := range []string{"first", "second"} {
		if !t.Run(name, func(t *testing.T) {
			db := Open(t, schemaFile)
			if maximum := db.Stats().MaxOpenConnections; maximum != 6 {
				t.Fatalf("max open connections = %d; want 6", maximum)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var schema string
			if err := db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(schema, "test_crud_") {
				t.Fatalf("unexpected schema %q", schema)
			}
			for _, previous := range schemas {
				if schema == previous {
					t.Fatal("schema was reused")
				}
			}
			schemas = append(schemas, schema)
			var fixtureExists bool
			if err := db.QueryRowContext(ctx, "SELECT to_regclass('fixture') IS NOT NULL").Scan(&fixtureExists); err != nil || !fixtureExists {
				t.Fatalf("schema fixture present = %v, error = %v; want true", fixtureExists, err)
			}
			var probeExists bool
			if err := db.QueryRowContext(ctx, "SELECT to_regclass('isolation_probe') IS NOT NULL").Scan(&probeExists); err != nil {
				t.Fatal(err)
			}
			if probeExists {
				t.Fatal("isolated schema contains a previous probe")
			}
			if _, err := db.ExecContext(ctx, "CREATE TABLE isolation_probe (id integer PRIMARY KEY)"); err != nil {
				t.Fatal(err)
			}
		}) {
			return
		}
	}
	admin := stdlib.OpenDB(*cfg)
	defer admin.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, schema := range schemas {
		var exists bool
		if err := admin.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)", schema).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Errorf("schema %q remained after cleanup", schema)
		}
	}
}
