package handler_test

import (
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

	"example.com/go-practice/04_crud/d03_orders/handler"
	"example.com/go-practice/04_crud/internal/testdb"
)

const requestTimeout = 5 * time.Second

type itemInput struct {
	ProductID int64 `json:"product_id"`
	Qty       int64 `json:"qty"`
}

type item struct {
	ProductID  int64  `json:"product_id"`
	Qty        int64  `json:"qty"`
	PriceCents *int64 `json:"price_cents"`
}

type order struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Items  []item `json:"items"`
}

type response struct {
	status int
	body   []byte
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	t.Cleanup(cancel)
	return ctx
}

func startServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler.New(db))
	server.Client().Timeout = requestTimeout
	t.Cleanup(server.Close)
	return server
}

func perform(server *httptest.Server, method, path, body string) (response, error) {
	req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		return response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := server.Client().Do(req)
	if err != nil {
		return response{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	return response{status: res.StatusCode, body: data}, err
}

func request(t *testing.T, server *httptest.Server, method, path, body string) response {
	t.Helper()
	res, err := perform(server, method, path, body)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return res
}

func expectStatus(t *testing.T, res response, status int) {
	t.Helper()
	if res.status != status {
		t.Fatalf("HTTP status = %d, body = %q; want %d", res.status, res.body, status)
	}
}

func decodeOrder(t *testing.T, res response, status int) order {
	t.Helper()
	expectStatus(t, res, status)
	var value order
	if err := json.Unmarshal(res.body, &value); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	return value
}

func expectOrder(t *testing.T, got, want order) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %#v; want %#v", got, want)
	}
}

func payload(t *testing.T, items ...itemInput) string {
	t.Helper()
	data, err := json.Marshal(struct {
		Items []itemInput `json:"items"`
	}{Items: items})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func orderPath(id int64) string { return "/orders/" + strconv.FormatInt(id, 10) }

func seedProduct(t *testing.T, db *sql.DB, name string, price, stock int64) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowContext(testContext(t),
		"INSERT INTO products (name, price_cents, stock) VALUES ($1,$2,$3) RETURNING id",
		name, price, stock).Scan(&id)
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	return id
}

func seedOrder(t *testing.T, db *sql.DB, items ...itemInput) order {
	t.Helper()
	var id int64
	if err := db.QueryRowContext(testContext(t), "INSERT INTO orders DEFAULT VALUES RETURNING id").Scan(&id); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	for _, entry := range items {
		_, err := db.ExecContext(testContext(t),
			"INSERT INTO order_items (order_id, product_id, qty) VALUES ($1,$2,$3)",
			id, entry.ProductID, entry.Qty)
		if err != nil {
			t.Fatalf("seed order item: %v", err)
		}
	}
	return storedOrder(t, db, id)
}

func storedOrder(t *testing.T, db *sql.DB, id int64) order {
	t.Helper()
	value := order{ID: id, Items: make([]item, 0)}
	if err := db.QueryRowContext(testContext(t), "SELECT status FROM orders WHERE id=$1", id).Scan(&value.Status); err != nil {
		t.Fatalf("read stored order: %v", err)
	}
	rows, err := db.QueryContext(testContext(t),
		"SELECT product_id, qty, price_cents FROM order_items WHERE order_id=$1 ORDER BY product_id", id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var entry item
		if err := rows.Scan(&entry.ProductID, &entry.Qty, &entry.PriceCents); err != nil {
			t.Fatal(err)
		}
		value.Items = append(value.Items, entry)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return value
}

func expectStock(t *testing.T, db *sql.DB, productID, want int64) {
	t.Helper()
	var got int64
	if err := db.QueryRowContext(testContext(t), "SELECT stock FROM products WHERE id=$1", productID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("product %d stock = %d; want %d", productID, got, want)
	}
}

func expectCount(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRowContext(testContext(t), query).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s = %d; want %d", query, got, want)
	}
}

func TestDraftCRUD(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p1 := seedProduct(t, db, "first", 150, 10)
	p2 := seedProduct(t, db, "second", 900, 3)
	server := startServer(t, db)
	res := request(t, server, "GET", "/orders", "")
	expectStatus(t, res, 200)
	if strings.TrimSpace(string(res.body)) != "[]" {
		t.Fatalf("initial list = %q; want []", res.body)
	}
	created := decodeOrder(t, request(t, server, "POST", "/orders",
		payload(t, itemInput{p2, 2}, itemInput{p1, 1})), 201)
	if created.ID <= 0 {
		t.Fatalf("created order ID = %d; want positive", created.ID)
	}
	want := order{ID: created.ID, Status: "draft", Items: []item{{ProductID: p1, Qty: 1}, {ProductID: p2, Qty: 2}}}
	expectOrder(t, created, want)
	expectOrder(t, storedOrder(t, db, want.ID), want)
	path := orderPath(want.ID)
	expectOrder(t, decodeOrder(t, request(t, server, "GET", path, ""), 200), want)
	res = request(t, server, "GET", "/orders", "")
	expectStatus(t, res, 200)
	var list []order
	if err := json.Unmarshal(res.body, &list); err != nil || !reflect.DeepEqual(list, []order{want}) {
		t.Fatalf("orders = %s, error = %v; want %#v", res.body, err, []order{want})
	}
	want.Items = []item{{ProductID: p2, Qty: 1}}
	expectOrder(t, decodeOrder(t, request(t, server, "PUT", path, payload(t, itemInput{p2, 1})), 200), want)
	expectOrder(t, storedOrder(t, db, want.ID), want)
	expectCount(t, db, "SELECT count(*) FROM order_items", 1)
	res = request(t, server, "DELETE", path, "")
	expectStatus(t, res, 204)
	if len(res.body) != 0 {
		t.Fatalf("DELETE response = %q; want empty", res.body)
	}
	expectCount(t, db, "SELECT count(*) FROM orders", 0)
	expectCount(t, db, "SELECT count(*) FROM order_items", 0)
	expectStock(t, db, p1, 10)
	expectStock(t, db, p2, 3)
	for _, tc := range []struct{ method, suffix, body string }{
		{"GET", "", ""}, {"PUT", "", payload(t, itemInput{p1, 1})},
		{"DELETE", "", ""}, {"POST", "/confirm", ""},
	} {
		expectStatus(t, request(t, server, tc.method, path+tc.suffix, tc.body), 404)
	}
}

func TestOrderListKeepsItemsGrouped(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p1 := seedProduct(t, db, "shared", 150, 10)
	p2 := seedProduct(t, db, "second", 250, 10)
	p3 := seedProduct(t, db, "third", 350, 10)
	first := seedOrder(t, db, itemInput{p2, 2}, itemInput{p1, 1})
	second := seedOrder(t, db, itemInput{p3, 3}, itemInput{p1, 4})
	server := startServer(t, db)
	res := request(t, server, "GET", "/orders", "")
	expectStatus(t, res, 200)
	var got []order
	if err := json.Unmarshal(res.body, &got); err != nil {
		t.Fatalf("decode order list: %v", err)
	}
	want := []order{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order list = %#v; want %#v", got, want)
	}
}

func TestInvalidDraftsLeaveNoRows(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p := seedProduct(t, db, "one", 150, 5)
	server := startServer(t, db)
	for _, body := range []string{
		`{}`, `{"items":[]}`, `{"items":null}`,
		payload(t, itemInput{p, 0}), payload(t, itemInput{p, -1}),
		payload(t, itemInput{0, 1}), payload(t, itemInput{-1, 1}),
		payload(t, itemInput{p, 1}, itemInput{p, 2}),
	} {
		t.Run(body, func(t *testing.T) {
			expectStatus(t, request(t, server, "POST", "/orders", body), 400)
			expectCount(t, db, "SELECT count(*) FROM orders", 0)
			expectCount(t, db, "SELECT count(*) FROM order_items", 0)
			expectStock(t, db, p, 5)
		})
	}
}

func TestDraftForeignKeyFailuresRollback(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p := seedProduct(t, db, "one", 150, 5)
	original := seedOrder(t, db, itemInput{p, 1})
	server := startServer(t, db)
	bad := payload(t, itemInput{p, 4}, itemInput{999999, 1})
	for _, tc := range []struct{ method, path string }{{"POST", "/orders"}, {"PUT", orderPath(original.ID)}} {
		expectStatus(t, request(t, server, tc.method, tc.path, bad), 409)
		expectCount(t, db, "SELECT count(*) FROM orders", 1)
		expectCount(t, db, "SELECT count(*) FROM order_items", 1)
		expectOrder(t, storedOrder(t, db, original.ID), original)
		expectStock(t, db, p, 5)
	}
}

func TestInsufficientSecondItemRollsBackConfirmation(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p1 := seedProduct(t, db, "available", 150, 10)
	p2 := seedProduct(t, db, "unavailable", 900, 0)
	original := seedOrder(t, db, itemInput{p1, 3}, itemInput{p2, 1})
	server := startServer(t, db)
	expectStatus(t, request(t, server, "POST", orderPath(original.ID)+"/confirm", ""), 409)
	expectOrder(t, storedOrder(t, db, original.ID), original)
	expectStock(t, db, p1, 10)
	expectStock(t, db, p2, 0)
}

func TestConfirmationSnapshotsPricesAndIsIdempotent(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p1 := seedProduct(t, db, "one", 150, 10)
	p2 := seedProduct(t, db, "two", 900, 3)
	want := seedOrder(t, db, itemInput{p2, 1}, itemInput{p1, 2})
	server := startServer(t, db)
	price1, price2 := int64(150), int64(900)
	want.Status = "confirmed"
	want.Items = []item{{ProductID: p1, Qty: 2, PriceCents: &price1}, {ProductID: p2, Qty: 1, PriceCents: &price2}}
	path := orderPath(want.ID)
	expectOrder(t, decodeOrder(t, request(t, server, "POST", path+"/confirm", ""), 200), want)
	expectOrder(t, storedOrder(t, db, want.ID), want)
	expectStock(t, db, p1, 8)
	expectStock(t, db, p2, 2)
	if _, err := db.ExecContext(testContext(t), "UPDATE products SET price_cents=5000 WHERE id=$1", p1); err != nil {
		t.Fatal(err)
	}
	expectOrder(t, decodeOrder(t, request(t, server, "POST", path+"/confirm", ""), 200), want)
	expectOrder(t, decodeOrder(t, request(t, server, "GET", path, ""), 200), want)
	for _, tc := range []struct{ method, body string }{{"PUT", payload(t, itemInput{p1, 1})}, {"DELETE", ""}} {
		expectStatus(t, request(t, server, tc.method, path, tc.body), 409)
	}
	expectOrder(t, storedOrder(t, db, want.ID), want)
	expectStock(t, db, p1, 8)
	expectStock(t, db, p2, 2)
}

func TestCommitFailureRollsBackConfirmation(t *testing.T) {
	db := testdb.Open(t, "../schema.sql")
	p := seedProduct(t, db, "one", 150, 5)
	original := seedOrder(t, db, itemInput{p, 2})
	_, err := db.ExecContext(testContext(t), `
		CREATE FUNCTION reject_confirmation_commit() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.status = 'confirmed' THEN
				RAISE EXCEPTION 'fixture commit rejected';
			END IF;
			RETURN NEW;
		END $$;
		CREATE CONSTRAINT TRIGGER reject_confirmation_commit
		AFTER UPDATE ON orders DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION reject_confirmation_commit()`)
	if err != nil {
		t.Fatalf("create deferred commit-failure fixture: %v", err)
	}
	probe, err := db.BeginTx(testContext(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Rollback()
	if _, err := probe.ExecContext(testContext(t), "UPDATE orders SET status='confirmed' WHERE id=$1", original.ID); err != nil {
		t.Fatalf("fixture failed before Commit: %v", err)
	}
	if err := probe.Commit(); err == nil || !strings.Contains(err.Error(), "fixture commit rejected") {
		t.Fatalf("fixture Commit error = %v; want deferred rejection", err)
	}
	server := startServer(t, db)
	expectStatus(t, request(t, server, "POST", orderPath(original.ID)+"/confirm", ""), 500)
	expectOrder(t, storedOrder(t, db, original.ID), original)
	expectStock(t, db, p, 5)
	if err := db.PingContext(testContext(t)); err != nil {
		t.Fatalf("pool after Commit failure: %v", err)
	}
}
