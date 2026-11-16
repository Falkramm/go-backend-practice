package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/go-practice/04_crud/d02_catalog/handler"
	"example.com/go-practice/04_crud/internal/testdb"
)

type category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type product struct {
	ID           int64   `json:"id"`
	SKU          string  `json:"sku"`
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Name         string  `json:"name"`
	PriceCents   int64   `json:"price_cents"`
	Stock        int64   `json:"stock"`
	Description  *string `json:"description"`
}

type productInput struct {
	SKU         string  `json:"sku,omitempty"`
	CategoryID  int64   `json:"category_id"`
	Name        string  `json:"name"`
	PriceCents  int64   `json:"price_cents"`
	Stock       int64   `json:"stock"`
	Description *string `json:"description"`
}

type reply struct {
	status int
	body   []byte
}

func setup(t *testing.T) (*sql.DB, *httptest.Server) {
	t.Helper()
	db := testdb.Open(t, "../schema.sql")
	server := httptest.NewServer(handler.New(db))
	server.Client().Timeout = 3 * time.Second
	t.Cleanup(server.Close)
	return db, server
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func send(server *httptest.Server, method, path string, input any) (reply, error) {
	var body bytes.Buffer
	if input != nil {
		if err := json.NewEncoder(&body).Encode(input); err != nil {
			return reply{}, err
		}
	}
	req, err := http.NewRequest(method, server.URL+path, &body)
	if err != nil {
		return reply{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := server.Client().Do(req)
	if err != nil {
		return reply{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	return reply{status: res.StatusCode, body: data}, err
}

func call(t *testing.T, server *httptest.Server, method, path string, input any, status int) reply {
	t.Helper()
	res, err := send(server, method, path, input)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if res.status != status {
		t.Fatalf("%s %s: status=%d body=%q, want %d", method, path, res.status, res.body, status)
	}
	if status >= 400 && strings.TrimSpace(string(res.body)) == "" {
		t.Fatal("error response must contain a message")
	}
	if status == http.StatusNoContent && len(res.body) != 0 {
		t.Fatalf("204 body=%q, want empty", res.body)
	}
	return res
}

func decode[T any](t *testing.T, res reply) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(res.body, &value); err != nil {
		t.Fatalf("decode %q: %v", res.body, err)
	}
	return value
}

func path(resource string, id int64) string {
	return "/" + resource + "/" + strconv.FormatInt(id, 10)
}

func createCategory(t *testing.T, server *httptest.Server, name string) category {
	t.Helper()
	got := decode[category](t, call(t, server, "POST", "/categories", map[string]string{"name": name}, 201))
	if got.ID <= 0 || got.Name != name {
		t.Fatalf("created category=%+v, want generated ID and name %q", got, name)
	}
	return got
}

func expectStoredProduct(t *testing.T, db *sql.DB, want product) {
	t.Helper()
	var got product
	err := db.QueryRowContext(testContext(t), `SELECT p.id,p.sku,p.category_id,c.name,p.name,p.price_cents,p.stock,p.description
		FROM products p JOIN categories c ON c.id=p.category_id WHERE p.id=$1`, want.ID).
		Scan(&got.ID, &got.SKU, &got.CategoryID, &got.CategoryName, &got.Name, &got.PriceCents, &got.Stock, &got.Description)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("stored product=%+v error=%v, want %+v", got, err, want)
	}
}

func expectProduct(t *testing.T, got, want product) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("product=%+v, want %+v", got, want)
	}
}

func rowCount(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(testContext(t), query).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestCatalogCRUDAndJoin(t *testing.T) {
	db, server := setup(t)
	for _, resource := range []string{"categories", "products"} {
		res := call(t, server, "GET", "/"+resource, nil, 200)
		if strings.TrimSpace(string(res.body)) != "[]" {
			t.Fatalf("initial %s=%s, want []", resource, res.body)
		}
	}
	cat := createCategory(t, server, "Tools")
	if got := decode[category](t, call(t, server, "GET", path("categories", cat.ID), nil, 200)); got != cat {
		t.Fatalf("category=%+v, want %+v", got, cat)
	}
	cat.Name = "Hardware"
	if got := decode[category](t, call(t, server, "PUT", path("categories", cat.ID), map[string]string{"name": cat.Name}, 200)); got != cat {
		t.Fatalf("replaced category=%+v, want %+v", got, cat)
	}
	if got := decode[[]category](t, call(t, server, "GET", "/categories", nil, 200)); !reflect.DeepEqual(got, []category{cat}) {
		t.Fatalf("categories=%+v, want %+v", got, []category{cat})
	}
	text := "durable"
	input := productInput{SKU: "hammer-1", CategoryID: cat.ID, Name: "Hammer", PriceCents: 1250, Stock: 8, Description: &text}
	got := decode[product](t, call(t, server, "POST", "/products", input, 201))
	want := product{ID: got.ID, SKU: input.SKU, CategoryID: cat.ID, CategoryName: cat.Name, Name: input.Name, PriceCents: input.PriceCents, Stock: input.Stock, Description: &text}
	if got.ID <= 0 {
		t.Fatalf("generated product ID=%d", got.ID)
	}
	expectProduct(t, got, want)
	expectStoredProduct(t, db, want)
	expectProduct(t, decode[product](t, call(t, server, "GET", path("products", got.ID), nil, 200)), want)
	cat.Name = "Workshop"
	call(t, server, "PUT", path("categories", cat.ID), map[string]string{"name": cat.Name}, 200)
	want.CategoryName = cat.Name
	expectProduct(t, decode[product](t, call(t, server, "GET", path("products", got.ID), nil, 200)), want)
	expectStoredProduct(t, db, want)

	other := createCategory(t, server, "Garden")
	input = productInput{SKU: "spade-1", CategoryID: other.ID, Name: "Spade", PriceCents: 2000, Stock: 3}
	want.SKU, want.CategoryID, want.CategoryName, want.Name = input.SKU, other.ID, other.Name, input.Name
	want.PriceCents, want.Stock, want.Description = input.PriceCents, input.Stock, nil
	expectProduct(t, decode[product](t, call(t, server, "PUT", path("products", want.ID), input, 200)), want)
	expectStoredProduct(t, db, want)

	for _, change := range []struct {
		input       map[string]any
		description *string
	}{
		{map[string]any{"description": text}, &text},
		{map[string]any{}, &text},
		{map[string]any{"description": nil}, nil},
		{map[string]any{"description": ""}, new(string)},
	} {
		want.Description = change.description
		expectProduct(t, decode[product](t, call(t, server, "PATCH", path("products", want.ID), change.input, 200)), want)
		expectStoredProduct(t, db, want)
	}
	want.PriceCents, want.Stock = 0, 0
	expectProduct(t, decode[product](t, call(t, server, "PATCH", path("products", want.ID), map[string]int{"price_cents": 0, "stock": 0}, 200)), want)
	expectStoredProduct(t, db, want)

	call(t, server, "DELETE", path("products", want.ID), nil, 204)
	call(t, server, "GET", path("products", want.ID), nil, 404)
	if got := rowCount(t, db, "SELECT count(*) FROM products"); got != 0 {
		t.Fatalf("products after delete=%d, want 0", got)
	}
	call(t, server, "DELETE", path("categories", cat.ID), nil, 204)
	call(t, server, "DELETE", path("categories", other.ID), nil, 204)
	call(t, server, "GET", path("categories", cat.ID), nil, 404)
	if got := rowCount(t, db, "SELECT count(*) FROM categories"); got != 0 {
		t.Fatalf("categories after delete=%d, want 0", got)
	}
}

func TestProductFiltersPaginationAndStableSort(t *testing.T) {
	db, server := setup(t)
	_, err := db.ExecContext(testContext(t), `
		INSERT INTO categories(id,name) VALUES(1,'first'),(2,'second');
		INSERT INTO products(id,sku,category_id,name,price_cents,stock) VALUES
		(30,'p30',1,'alpha',100,1),(10,'p10',1,'same',100,1),
		(20,'p20',2,'beta',150,1),(40,'p40',1,'same',200,1)`)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		ids   []int64
	}{
		{"", []int64{10, 20, 30, 40}},
		{"?limit=1&offset=1", []int64{20}},
		{"?category_id=1&sort=price_cents", []int64{10, 30, 40}},
		{"?category_id=1&sort=name", []int64{30, 10, 40}},
		{"?category_id=1&sort=price_cents&limit=1&offset=1", []int64{30}},
		{"?sort=price_cents", []int64{10, 30, 20, 40}},
		{"?offset=100", []int64{}},
		{"?category_id=9999", []int64{}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			products := decode[[]product](t, call(t, server, "GET", "/products"+tc.query, nil, 200))
			if products == nil {
				t.Fatal("product list must be an array")
			}
			ids := make([]int64, 0, len(products))
			for _, item := range products {
				ids = append(ids, item.ID)
				if item.CategoryName == "" {
					t.Fatal("product must include joined category_name")
				}
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("IDs=%v, want %v", ids, tc.ids)
			}
		})
	}
}

func TestProductDefaultLimit(t *testing.T) {
	db, server := setup(t)
	var categoryID int64
	if err := db.QueryRowContext(testContext(t), "INSERT INTO categories(name) VALUES('defaults') RETURNING id").Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	_, err := db.ExecContext(testContext(t), `INSERT INTO products(sku,category_id,name,price_cents,stock)
		SELECT 'sku-'||n,$1,'product '||n,n,1 FROM generate_series(1,25) n ORDER BY n`, categoryID)
	if err != nil {
		t.Fatal(err)
	}
	got := decode[[]product](t, call(t, server, "GET", "/products", nil, 200))
	if len(got) != 20 || got[0].SKU != "sku-1" || got[19].SKU != "sku-20" {
		t.Fatalf("default list=%+v, want first 20 products by ID", got)
	}
}

func TestCatalogValidationAndConflicts(t *testing.T) {
	db, server := setup(t)
	first := createCategory(t, server, "first")
	second := createCategory(t, server, "second")
	valid := productInput{SKU: "sku-a", CategoryID: first.ID, Name: "A", PriceCents: 100, Stock: 2}
	original := decode[product](t, call(t, server, "POST", "/products", valid, 201))
	otherInput := valid
	otherInput.SKU = "sku-b"
	other := decode[product](t, call(t, server, "POST", "/products", otherInput, 201))
	for _, tc := range []struct {
		name, method, path string
		input              any
		status             int
	}{
		{"duplicate category", "POST", "/categories", map[string]string{"name": first.Name}, 409},
		{"rename conflict", "PUT", path("categories", second.ID), map[string]string{"name": first.Name}, 409},
		{"empty category", "POST", "/categories", map[string]string{"name": "  "}, 400},
		{"duplicate sku", "POST", "/products", valid, 409},
		{"replace sku conflict", "PUT", path("products", other.ID), valid, 409},
		{"referenced category", "DELETE", path("categories", first.ID), nil, 409},
		{"missing category", "PATCH", path("products", original.ID), map[string]int64{"category_id": 99999}, 409},
		{"negative patch price", "PATCH", path("products", original.ID), map[string]int{"price_cents": -1}, 400},
		{"negative patch stock", "PATCH", path("products", original.ID), map[string]int{"stock": -1}, 400},
		{"bad product id", "GET", "/products/nope", nil, 400},
		{"zero category id", "GET", "/categories/0", nil, 400},
		{"bad limit", "GET", "/products?limit=0", nil, 400},
		{"large limit", "GET", "/products?limit=101", nil, 400},
		{"bad offset", "GET", "/products?offset=-1", nil, 400},
		{"bad filter", "GET", "/products?category_id=0", nil, 400},
		{"bad sort", "GET", "/products?sort=price_cents%3BDROP+TABLE+products", nil, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call(t, server, tc.method, tc.path, tc.input, tc.status)
			expectStoredProduct(t, db, original)
			expectStoredProduct(t, db, other)
		})
	}
	for _, change := range []struct {
		name string
		edit func(*productInput)
	}{
		{"blank sku", func(p *productInput) { p.SKU = " " }},
		{"blank name", func(p *productInput) { p.Name = " " }},
		{"negative price", func(p *productInput) { p.PriceCents = -1 }},
		{"negative stock", func(p *productInput) { p.Stock = -1 }},
		{"zero category", func(p *productInput) { p.CategoryID = 0 }},
	} {
		t.Run(change.name, func(t *testing.T) {
			input := valid
			change.edit(&input)
			call(t, server, "POST", "/products", input, 400)
		})
	}
	missing := valid
	missing.SKU, missing.CategoryID = "missing-category", 99999
	call(t, server, "POST", "/products", missing, 409)
	for _, tc := range []struct{ method, resource string }{
		{"GET", "categories"}, {"PUT", "categories"}, {"DELETE", "categories"},
		{"GET", "products"}, {"PUT", "products"}, {"PATCH", "products"}, {"DELETE", "products"},
	} {
		var input any = valid
		if tc.resource == "categories" {
			input = map[string]string{"name": "missing"}
		}
		call(t, server, tc.method, path(tc.resource, 99999), input, 404)
	}
	if got := rowCount(t, db, "SELECT count(*) FROM products"); got != 2 {
		t.Fatalf("products after invalid requests=%d, want 2", got)
	}
	if got := rowCount(t, db, "SELECT count(*) FROM categories"); got != 2 {
		t.Fatalf("categories after invalid requests=%d, want 2", got)
	}
}

func TestUpsertKeepsIDAndReplacesWholeState(t *testing.T) {
	db, server := setup(t)
	first := createCategory(t, server, "first")
	second := createCategory(t, server, "second")
	url := "/products/by-sku/same-sku"
	input := productInput{CategoryID: first.ID, Name: "original", PriceCents: 100, Stock: 2}
	got := decode[product](t, call(t, server, "PUT", url, input, 200))
	want := product{ID: got.ID, SKU: "same-sku", CategoryID: first.ID, CategoryName: first.Name, Name: input.Name, PriceCents: 100, Stock: 2}
	if want.ID <= 0 {
		t.Fatal("upsert must generate a product ID")
	}
	expectProduct(t, got, want)
	expectProduct(t, decode[product](t, call(t, server, "PUT", url, input, 200)), want)
	text := "new"
	input = productInput{CategoryID: second.ID, Name: "replacement", PriceCents: 250, Stock: 0, Description: &text}
	want.CategoryID, want.CategoryName, want.Name = second.ID, second.Name, input.Name
	want.PriceCents, want.Stock, want.Description = 250, 0, &text
	expectProduct(t, decode[product](t, call(t, server, "PUT", url, input, 200)), want)
	expectStoredProduct(t, db, want)
	input.Description, want.Description = nil, nil
	expectProduct(t, decode[product](t, call(t, server, "PUT", url, input, 200)), want)
	expectStoredProduct(t, db, want)
	if got := rowCount(t, db, "SELECT count(*) FROM products"); got != 1 {
		t.Fatalf("rows after repeated upsert=%d, want 1", got)
	}
}

func TestConcurrentUpsertCreatesOneWholeProduct(t *testing.T) {
	db, server := setup(t)
	first := createCategory(t, server, "first")
	second := createCategory(t, server, "second")
	blocker, err := db.BeginTx(testContext(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback()
	if _, err := blocker.ExecContext(testContext(t), "LOCK TABLE products IN SHARE MODE"); err != nil {
		t.Fatal(err)
	}
	var blockerPID int
	if err := blocker.QueryRowContext(testContext(t), "SELECT pg_backend_pid()").Scan(&blockerPID); err != nil {
		t.Fatal(err)
	}
	text := "second body"
	inputs := []productInput{
		{CategoryID: first.ID, Name: "first payload", PriceCents: 10, Stock: 1},
		{CategoryID: second.ID, Name: "second payload", PriceCents: 20, Stock: 2, Description: &text},
	}
	type result struct {
		response reply
		err      error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for _, input := range inputs {
		go func(input productInput) {
			<-start
			res, err := send(server, "PUT", "/products/by-sku/concurrent", input)
			results <- result{response: res, err: err}
		}(input)
	}
	close(start)
	waitForUpsertWriters(t, db, blockerPID)
	if err := blocker.Rollback(); err != nil {
		t.Fatalf("release upsert blocker: %v", err)
	}
	var id int64
	for range 2 {
		res := <-results
		if res.err != nil || res.response.status != 200 {
			t.Fatalf("concurrent upsert status=%d body=%q error=%v", res.response.status, res.response.body, res.err)
		}
		got := decode[product](t, res.response)
		if got.ID <= 0 || (id != 0 && got.ID != id) {
			t.Fatalf("concurrent IDs=%d and %d, want one generated ID", id, got.ID)
		}
		id = got.ID
	}
	stored := decode[product](t, call(t, server, "GET", path("products", id), nil, 200))
	wantFirst := product{ID: id, SKU: "concurrent", CategoryID: first.ID, CategoryName: first.Name, Name: inputs[0].Name, PriceCents: 10, Stock: 1}
	wantSecond := product{ID: id, SKU: "concurrent", CategoryID: second.ID, CategoryName: second.Name, Name: inputs[1].Name, PriceCents: 20, Stock: 2, Description: &text}
	if !reflect.DeepEqual(stored, wantFirst) && !reflect.DeepEqual(stored, wantSecond) {
		t.Fatalf("upsert mixed payloads: %+v", stored)
	}
	expectStoredProduct(t, db, stored)
	if got := rowCount(t, db, "SELECT count(*) FROM products"); got != 1 {
		t.Fatalf("concurrent upsert rows=%d, want 1", got)
	}
}

func waitForUpsertWriters(t *testing.T, db *sql.DB, blockerPID int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int
		err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND $1=ANY(pg_blocking_pids(pid))`, blockerPID).Scan(&count)
		if err != nil {
			t.Fatalf("observe simultaneous upsert writers: %v", err)
		}
		if count == 2 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("blocked upsert writers=%d, want 2", count)
		case <-ticker.C:
		}
	}
}

func TestCatalogDatabaseConstraints(t *testing.T) {
	db, _ := setup(t)
	ctx := testContext(t)
	var categoryID int64
	err := db.QueryRowContext(ctx, "INSERT INTO categories(name) VALUES('first') RETURNING id").Scan(&categoryID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, "INSERT INTO products(sku,category_id,name,price_cents,stock) VALUES('first',$1,'product',10,1)", categoryID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		args  []any
	}{
		{"INSERT INTO categories(name) VALUES('first')", nil},
		{"INSERT INTO products(sku,category_id,name,price_cents,stock) VALUES('first',$1,'duplicate',1,1)", []any{categoryID}},
		{"INSERT INTO products(sku,category_id,name,price_cents,stock) VALUES('bad-category',99999,'bad',1,1)", nil},
		{"INSERT INTO products(sku,category_id,name,price_cents,stock) VALUES('bad-price',$1,'bad',-1,1)", []any{categoryID}},
		{"INSERT INTO products(sku,category_id,name,price_cents,stock) VALUES('bad-stock',$1,'bad',1,-1)", []any{categoryID}},
		{"DELETE FROM categories WHERE id=$1", []any{categoryID}},
	} {
		if _, err := db.ExecContext(ctx, tc.query, tc.args...); err == nil {
			t.Errorf("database accepted constraint violation: %s", tc.query)
		}
	}
	if got := rowCount(t, db, "SELECT count(*) FROM products"); got != 1 {
		t.Fatalf("rows after constraint checks=%d, want 1", got)
	}
}
