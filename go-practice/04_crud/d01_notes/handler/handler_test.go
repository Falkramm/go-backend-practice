package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/go-practice/04_crud/d01_notes/handler"
	"example.com/go-practice/04_crud/d01_notes/internal/testdb"
)

const requestTimeout = 3 * time.Second

type response struct {
	status int
	header http.Header
	body   []byte
}

func startServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler.New(db))
	t.Cleanup(server.Close)
	return server
}

func request(t *testing.T, server *httptest.Server, method, path, body, contentType string) response {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := server.Client()
	client.Timeout = requestTimeout
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read HTTP response: %v", err)
	}
	return response{res.StatusCode, res.Header.Clone(), data}
}

func expectStatus(t *testing.T, res response, status int) {
	t.Helper()
	if res.status != status {
		t.Fatalf("HTTP status = %d, body = %q; want %d", res.status, res.body, status)
	}
}

func expectJSON(t *testing.T, res response) {
	t.Helper()
	mediaType, _, err := mime.ParseMediaType(res.header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		t.Fatalf("Content-Type = %q; want application/json", res.header.Get("Content-Type"))
	}
}

func expectError(t *testing.T, res response, status int) {
	t.Helper()
	expectStatus(t, res, status)
	mediaType, _, err := mime.ParseMediaType(res.header.Get("Content-Type"))
	if err != nil || mediaType != "text/plain" {
		t.Fatalf("error Content-Type = %q; want text/plain", res.header.Get("Content-Type"))
	}
	if strings.TrimSpace(string(res.body)) == "" {
		t.Fatal("error response must have a nonempty message")
	}
}

func decodeNote(t *testing.T, res response, status int) handler.Note {
	t.Helper()
	expectStatus(t, res, status)
	expectJSON(t, res)
	var note handler.Note
	if err := json.Unmarshal(res.body, &note); err != nil {
		t.Fatalf("decode note: %v", err)
	}
	return note
}

func decodeList(t *testing.T, res response) []handler.Note {
	t.Helper()
	expectStatus(t, res, http.StatusOK)
	expectJSON(t, res)
	if !strings.HasPrefix(strings.TrimSpace(string(res.body)), "[") {
		t.Fatalf("list body = %q; want a JSON array", res.body)
	}
	var notes []handler.Note
	if err := json.Unmarshal(res.body, &notes); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return notes
}

func expectNote(t *testing.T, got, want handler.Note) {
	t.Helper()
	if got.ID != want.ID || got.Title != want.Title || got.Pinned != want.Pinned ||
		!reflect.DeepEqual(got.Body, want.Body) || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("note = %#v; want %#v", got, want)
	}
}

func expectStoredNote(t *testing.T, db *sql.DB, want handler.Note) {
	t.Helper()
	var got handler.Note
	var body sql.NullString
	err := db.QueryRowContext(context.Background(),
		"SELECT id, title, body, pinned, created_at FROM notes WHERE id = $1", want.ID,
	).Scan(&got.ID, &got.Title, &body, &got.Pinned, &got.CreatedAt)
	if err != nil {
		t.Fatalf("read stored note: %v", err)
	}
	if body.Valid {
		got.Body = &body.String
	}
	expectNote(t, got, want)
}

func TestNotesCRUDWithPostgreSQL(t *testing.T) {
	db := testdb.Open(t)
	server := startServer(t, db)
	if notes := decodeList(t, request(t, server, "GET", "/notes", "", "")); len(notes) != 0 {
		t.Fatalf("initial list = %#v; want an empty array", notes)
	}

	createdResponse := request(t, server, "POST", "/notes",
		`{"title":"Go","body":"practice HTTP","pinned":true}`, "application/json")
	created := decodeNote(t, createdResponse, http.StatusCreated)
	if created.ID <= 0 || created.CreatedAt.IsZero() {
		t.Fatalf("created note lacks generated fields: %#v", created)
	}
	body := "practice HTTP"
	want := handler.Note{ID: created.ID, Title: "Go", Body: &body, Pinned: true, CreatedAt: created.CreatedAt}
	expectNote(t, created, want)
	expectStoredNote(t, db, want)
	path := "/notes/" + strconv.FormatInt(created.ID, 10)
	if createdResponse.header.Get("Location") != path {
		t.Fatalf("Location = %q; want %q", createdResponse.header.Get("Location"), path)
	}

	res := request(t, server, "GET", path, "", "")
	expectNote(t, decodeNote(t, res, http.StatusOK), want)

	want.Title, want.Body, want.Pinned = "HTTP", nil, true
	res = request(t, server, "PUT", path, `{"title":"HTTP","body":null,"pinned":true}`, "application/json")
	expectNote(t, decodeNote(t, res, http.StatusOK), want)
	expectStoredNote(t, db, want)

	draft := "draft"
	empty := ""
	patches := []struct {
		payload string
		body    *string
		pinned  bool
	}{
		{`{"body":"draft"}`, &draft, true},
		{`{"body":null}`, nil, true},
		{`{"body":"","pinned":false}`, &empty, false},
		{`{}`, &empty, false},
	}
	for _, patch := range patches {
		want.Body, want.Pinned = patch.body, patch.pinned
		res = request(t, server, "PATCH", path, patch.payload, "application/json")
		expectNote(t, decodeNote(t, res, http.StatusOK), want)
		expectStoredNote(t, db, want)
	}

	res = request(t, server, "DELETE", path, "", "")
	expectStatus(t, res, http.StatusNoContent)
	if len(res.body) != 0 {
		t.Fatalf("DELETE body = %q; want an empty body", res.body)
	}
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT count(*) FROM notes").Scan(&count); err != nil || count != 0 {
		t.Fatalf("remaining rows = %d, error = %v; want zero", count, err)
	}
	expectError(t, request(t, server, "GET", path, "", ""), http.StatusNotFound)
	if notes := decodeList(t, request(t, server, "GET", "/notes", "", "")); len(notes) != 0 {
		t.Fatalf("final list = %#v; want an empty array", notes)
	}
}

func TestNotesPaginationWithPostgreSQL(t *testing.T) {
	db := testdb.Open(t)
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO notes (id, title, body, pinned, created_at)
		OVERRIDING SYSTEM VALUE
		VALUES (30, 'third', NULL, false, $1),
		       (10, 'first', NULL, false, $1),
		       (20, 'second', NULL, false, $1)`, time.Now().UTC())
	if err != nil {
		t.Fatalf("seed notes for pagination: %v", err)
	}
	server := startServer(t, db)
	cases := []struct {
		query string
		ids   []int64
	}{
		{"", []int64{10, 20, 30}},
		{"?limit=1&offset=1", []int64{20}},
		{"?limit=100&offset=0", []int64{10, 20, 30}},
		{"?limit=100&offset=5", []int64{}},
	}
	for _, tc := range cases {
		notes := decodeList(t, request(t, server, "GET", "/notes"+tc.query, "", ""))
		ids := make([]int64, 0, len(notes))
		for _, note := range notes {
			ids = append(ids, note.ID)
		}
		if !reflect.DeepEqual(ids, tc.ids) {
			t.Fatalf("list %q IDs = %v; want %v", tc.query, ids, tc.ids)
		}
	}
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT count(*) FROM notes").Scan(&count); err != nil || count != 3 {
		t.Fatalf("rows after List = %d, error = %v; want 3", count, err)
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func seedNote(t *testing.T, db *sql.DB) handler.Note {
	t.Helper()
	var note handler.Note
	err := db.QueryRowContext(testContext(t), `
		INSERT INTO notes (title, body, pinned) VALUES ('original', 'original body', true)
		RETURNING id, title, body, pinned, created_at`).
		Scan(&note.ID, &note.Title, &note.Body, &note.Pinned, &note.CreatedAt)
	if err != nil {
		t.Fatalf("seed note: %v", err)
	}
	return note
}

func TestRejectsInvalidJSONWithoutChangingNotes(t *testing.T) {
	db := testdb.Open(t)
	original := seedNote(t, db)
	server := startServer(t, db)
	payloads := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"malformed", "{"},
		{"array", "[]"},
		{"null_object", "null"},
		{"wrong_title_type", `{"title":7}`},
		{"wrong_body_type", `{"title":"Go","body":7}`},
		{"wrong_pinned_type", `{"title":"Go","pinned":"false"}`},
	}
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		for _, payload := range payloads {
			t.Run(method+"_"+payload.name, func(t *testing.T) {
				path := "/notes"
				if method != "POST" {
					path += "/" + strconv.FormatInt(original.ID, 10)
				}
				expectError(t, request(t, server, method, path, payload.body, "application/json"), http.StatusBadRequest)
				expectStoredNote(t, db, original)
			})
		}
	}
	var count int
	if err := db.QueryRowContext(testContext(t), "SELECT count(*) FROM notes").Scan(&count); err != nil || count != 1 {
		t.Fatalf("rows after invalid requests = %d, error = %v; want 1", count, err)
	}
}

func TestOrdinaryJSONDecoding(t *testing.T) {
	db := testdb.Open(t)
	original := seedNote(t, db)
	server := startServer(t, db)
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		for _, contentType := range []string{"", "text/plain", "application/json; charset=utf-8"} {
			t.Run(method+"_"+contentType, func(t *testing.T) {
				path := "/notes"
				status := http.StatusCreated
				if method != "POST" {
					path += "/" + strconv.FormatInt(original.ID, 10)
					status = http.StatusOK
				}
				res := request(t, server, method, path,
					`{"title":"Go","body":null,"pinned":false,"unknown":true}`, contentType)
				note := decodeNote(t, res, status)
				if note.Title != "Go" || note.Body != nil || note.Pinned {
					t.Fatalf("decoded note = %#v", note)
				}
				expectStoredNote(t, db, note)
			})
		}
	}
}

func TestPatchNullTitleAndPinnedAreOmitted(t *testing.T) {
	db := testdb.Open(t)
	original := seedNote(t, db)
	server := startServer(t, db)
	path := "/notes/" + strconv.FormatInt(original.ID, 10)
	res := request(t, server, "PATCH", path, `{"title":null,"pinned":null}`, "application/json")
	expectNote(t, decodeNote(t, res, http.StatusOK), original)
	expectStoredNote(t, db, original)
}

func TestJSONContentTypeParametersAndWhitespaceTail(t *testing.T) {
	db := testdb.Open(t)
	server := startServer(t, db)
	res := request(t, server, "POST", "/notes", "{\"title\":\"Go\",\"body\":null,\"pinned\":false} \n\t",
		"application/json; charset=utf-8")
	created := decodeNote(t, res, http.StatusCreated)
	expectStoredNote(t, db, created)
}

func TestRejectsBlankTitle(t *testing.T) {
	db := testdb.Open(t)
	original := seedNote(t, db)
	server := startServer(t, db)
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		for _, title := range []string{"", "   ", "\t\n"} {
			t.Run(method+"_"+strconv.Quote(title), func(t *testing.T) {
				path := "/notes"
				if method != "POST" {
					path += "/" + strconv.FormatInt(original.ID, 10)
				}
				payload, err := json.Marshal(map[string]string{"title": title})
				if err != nil {
					t.Fatal(err)
				}
				expectError(t, request(t, server, method, path, string(payload), "application/json"), http.StatusBadRequest)
				expectStoredNote(t, db, original)
			})
		}
	}
}

func TestPostAndPutRequireTitle(t *testing.T) {
	db := testdb.Open(t)
	original := seedNote(t, db)
	server := startServer(t, db)
	for _, method := range []string{"POST", "PUT"} {
		for _, payload := range []string{`{}`, `{"body":"x","pinned":true}`, `{"title":null}`} {
			t.Run(method+"_"+payload, func(t *testing.T) {
				path := "/notes"
				if method == "PUT" {
					path += "/" + strconv.FormatInt(original.ID, 10)
				}
				expectError(t, request(t, server, method, path, payload, "application/json"), http.StatusBadRequest)
				expectStoredNote(t, db, original)
			})
		}
	}
	var count int
	if err := db.QueryRowContext(testContext(t), "SELECT count(*) FROM notes").Scan(&count); err != nil || count != 1 {
		t.Fatalf("rows after requests without title = %d, error = %v; want 1", count, err)
	}
}

func TestRejectsInvalidIDsAndPagination(t *testing.T) {
	db := testdb.Open(t)
	server := startServer(t, db)
	cases := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/notes/0", ""},
		{"GET", "/notes/-1", ""},
		{"GET", "/notes/not-a-number", ""},
		{"GET", "/notes/999999999999999999999", ""},
		{"DELETE", "/notes/0", ""},
		{"PUT", "/notes/0", `{"title":"Go"}`},
		{"PATCH", "/notes/0", `{}`},
		{"GET", "/notes?limit=0", ""},
		{"GET", "/notes?limit=101", ""},
		{"GET", "/notes?limit=-1", ""},
		{"GET", "/notes?limit=wrong", ""},
		{"GET", "/notes?offset=-1", ""},
		{"GET", "/notes?offset=wrong", ""},
	}
	for _, tc := range cases {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			expectError(t, request(t, server, tc.method, tc.path, tc.body, "application/json"), http.StatusBadRequest)
		})
	}
}

func TestListDefaultPagination(t *testing.T) {
	db := testdb.Open(t)
	_, err := db.ExecContext(testContext(t), "INSERT INTO notes (title) SELECT 'note ' || n FROM generate_series(1, 25) n")
	if err != nil {
		t.Fatal(err)
	}
	server := startServer(t, db)
	notes := decodeList(t, request(t, server, "GET", "/notes", "", ""))
	if len(notes) != 20 || notes[0].Title != "note 1" || notes[19].Title != "note 20" {
		t.Fatalf("default list = %#v; want first 20 notes", notes)
	}
}

func TestReplaceResetsOmittedFieldsAndPreservesTitle(t *testing.T) {
	db := testdb.Open(t)
	want := seedNote(t, db)
	server := startServer(t, db)
	path := "/notes/" + strconv.FormatInt(want.ID, 10)
	want.Title, want.Body, want.Pinned = "  keep spaces  ", nil, false
	res := request(t, server, "PUT", path, `{"title":"  keep spaces  "}`, "application/json")
	expectNote(t, decodeNote(t, res, http.StatusOK), want)
	expectStoredNote(t, db, want)
}

func TestStoresSQLPunctuationAsData(t *testing.T) {
	db := testdb.Open(t)
	server := startServer(t, db)
	input := map[string]any{"title": "Robert'); DROP TABLE notes; --", "body": "quote ' and placeholders $1 ; --"}
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	created := decodeNote(t, request(t, server, "POST", "/notes", string(payload), "application/json"), http.StatusCreated)
	if created.Title != input["title"] || created.Body == nil || *created.Body != input["body"] {
		t.Fatalf("stored SQL punctuation = %#v", created)
	}
	expectStoredNote(t, db, created)
	var count int
	if err := db.QueryRowContext(testContext(t), "SELECT count(*) FROM notes").Scan(&count); err != nil || count != 1 {
		t.Fatalf("notes count = %d, error = %v; want 1", count, err)
	}
}

func TestMissingNoteReturns404(t *testing.T) {
	db := testdb.Open(t)
	server := startServer(t, db)
	for _, tc := range []struct{ method, body string }{
		{"GET", ""}, {"PUT", `{"title":"missing"}`}, {"PATCH", `{"pinned":false}`}, {"DELETE", ""},
	} {
		t.Run(tc.method, func(t *testing.T) {
			expectError(t, request(t, server, tc.method, "/notes/987654321", tc.body, "application/json"), http.StatusNotFound)
		})
	}
}

func TestDatabaseFailureReturnsSafeError(t *testing.T) {
	db := testdb.Open(t)
	if _, err := db.ExecContext(testContext(t), "DROP TABLE notes"); err != nil {
		t.Fatal(err)
	}
	server := startServer(t, db)
	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/notes", `{"title":"Go"}`},
		{"GET", "/notes", ""},
		{"GET", "/notes/1", ""},
		{"PUT", "/notes/1", `{"title":"Go"}`},
		{"PATCH", "/notes/1", `{}`},
		{"DELETE", "/notes/1", ""},
	} {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			res := request(t, server, tc.method, tc.path, tc.body, "application/json")
			expectError(t, res, http.StatusInternalServerError)
			for _, detail := range []string{"SQLSTATE", "relation", "42P01", "does not exist"} {
				if strings.Contains(string(res.body), detail) {
					t.Fatalf("internal response exposed database details: %s", res.body)
				}
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	db := testdb.Open(t)
	server := startServer(t, db)
	expectStatus(t, request(t, server, "POST", "/notes/7", `{}`, "application/json"), http.StatusMethodNotAllowed)
}
