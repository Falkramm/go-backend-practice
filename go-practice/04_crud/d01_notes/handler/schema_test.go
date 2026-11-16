package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/go-practice/04_crud/d01_notes/internal/testdb"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSchemaContract(t *testing.T) {
	db := testdb.Open(t)
	ctx := schemaContext(t)
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('notes') IS NOT NULL").Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("schema.sql must create notes in the isolated schema")
	}

	expected := map[string]struct{ kind, nullable string }{
		"id":         {kind: "bigint", nullable: "NO"},
		"title":      {kind: "text", nullable: "NO"},
		"body":       {kind: "text", nullable: "YES"},
		"pinned":     {kind: "boolean", nullable: "NO"},
		"created_at": {kind: "timestamp with time zone", nullable: "NO"},
	}
	rows, err := db.QueryContext(ctx, "SELECT column_name,data_type,is_nullable FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='notes'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := make(map[string]bool)
	for rows.Next() {
		var name, kind, nullable string
		if err := rows.Scan(&name, &kind, &nullable); err != nil {
			t.Fatal(err)
		}
		want, required := expected[name]
		if required {
			seen[name] = true
			if kind != want.kind || nullable != want.nullable {
				t.Errorf("column %s=(%s,%s), want (%s,%s)", name, kind, nullable, want.kind, want.nullable)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	for name := range expected {
		if !seen[name] {
			t.Errorf("missing column %s", name)
		}
	}
	if t.Failed() {
		return
	}

	var primary bool
	err = db.QueryRowContext(ctx, `SELECT EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage k USING (constraint_catalog,constraint_schema,constraint_name)
        WHERE tc.table_schema=current_schema() AND tc.table_name='notes'
        AND tc.constraint_type='PRIMARY KEY' AND k.column_name='id')`).Scan(&primary)
	if err != nil || !primary {
		t.Fatalf("notes.id primary key=%v, error=%v", primary, err)
	}

	var before, after, created time.Time
	if err := db.QueryRowContext(ctx, "SELECT clock_timestamp()").Scan(&before); err != nil {
		t.Fatal(err)
	}
	var id int64
	var pinned, bodyNull bool
	err = db.QueryRowContext(ctx, "INSERT INTO notes(title) VALUES($1) RETURNING id,pinned,body IS NULL,created_at", "defaults").Scan(&id, &pinned, &bodyNull, &created)
	if err != nil {
		t.Fatalf("insert default fields: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT clock_timestamp()").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if id <= 0 || pinned || !bodyNull || created.Before(before) || created.After(after) {
		t.Fatalf("invalid defaults id=%d pinned=%v bodyNull=%v created=%v", id, pinned, bodyNull, created)
	}

	cases := []struct {
		name, query, code string
		args              []any
	}{
		{name: "empty title", query: "INSERT INTO notes(title) VALUES($1)", code: "23514", args: []any{""}},
		{name: "whitespace title", query: "INSERT INTO notes(title) VALUES($1)", code: "23514", args: []any{"   "}},
		{name: "null title", query: "INSERT INTO notes(title) VALUES(NULL)", code: "23502"},
		{name: "null pinned", query: "INSERT INTO notes(title,pinned) VALUES('null pinned',NULL)", code: "23502"},
		{name: "null timestamp", query: "INSERT INTO notes(title,created_at) VALUES('null timestamp',NULL)", code: "23502"},
		{name: "duplicate id", query: "INSERT INTO notes(id,title) VALUES($1,'duplicate')", code: "23505", args: []any{id}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.ExecContext(ctx, tc.query, tc.args...)
			var pgError *pgconn.PgError
			if !errors.As(err, &pgError) || pgError.Code != tc.code {
				t.Fatalf("constraint error=%v, want SQLSTATE %s", err, tc.code)
			}
		})
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO notes(title,body) VALUES('empty body','')"); err != nil {
		t.Fatalf("empty body must be valid: %v", err)
	}
}

func schemaContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
