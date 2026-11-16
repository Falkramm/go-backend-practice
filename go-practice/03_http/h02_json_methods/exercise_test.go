package h02

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

const noteJSON = `{"id":7,"title":"note","body":null,"pinned":false,"created_at":"2026-11-09T12:00:00Z","extra":"allowed"}`

func newClient(t *testing.T, httpClient *http.Client, baseURL string) *Client {
	t.Helper()
	client, err := NewClient(httpClient, baseURL)
	if err != nil || client == nil {
		t.Fatalf("NewClient = %v, %v; want a client and nil error", client, err)
	}
	return client
}

func responseServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func httpClient(server *httptest.Server) *http.Client {
	client := server.Client()
	client.Timeout = testTimeout
	return client
}

func checkNote(t *testing.T, got Note, err error, want Note) {
	t.Helper()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("note = %#v, error = %v; want %#v and nil error", got, err, want)
	}
}

func callMethod(client *Client, method string) error {
	ctx := context.Background()
	var err error
	switch method {
	case http.MethodPost:
		_, err = client.Create(ctx, NoteInput{Title: "Go"})
	case http.MethodGet:
		_, err = client.Get(ctx, 7)
	case http.MethodPut:
		_, err = client.Replace(ctx, 7, NoteInput{Title: "Go"})
	case http.MethodPatch:
		_, err = client.Patch(ctx, 7, map[string]any{"pinned": false})
	case http.MethodDelete:
		err = client.Delete(ctx, 7)
	default:
		panic("unsupported test method")
	}
	return err
}

func TestClientCRUDChainSendsMethodsHeadersAndJSON(t *testing.T) {
	type request struct {
		method      string
		path        string
		accept      string
		contentType string
		body        string
	}
	var mu sync.Mutex
	var requests []request
	createdAt := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	var note Note
	exists := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, request{
			method: r.Method, path: r.URL.Path, accept: r.Header.Get("Accept"),
			contentType: r.Header.Get("Content-Type"), body: string(body),
		})
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/api/notes" {
			var input NoteInput
			if err := json.Unmarshal(body, &input); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			note = Note{ID: 7, Title: input.Title, Body: input.Body, Pinned: input.Pinned, CreatedAt: createdAt}
			exists = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(note)
			return
		}
		if r.URL.Path != "/api/notes/7" || !exists {
			http.Error(w, "missing", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
		case http.MethodPut:
			var input NoteInput
			if err := json.Unmarshal(body, &input); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			note.Title, note.Body, note.Pinned = input.Title, input.Body, input.Pinned
		case http.MethodPatch:
			if err := json.Unmarshal(body, &note); err != nil {
				http.Error(w, "invalid patch", http.StatusBadRequest)
				return
			}
		case http.MethodDelete:
			exists = false
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(note)
	}))
	defer server.Close()
	client := newClient(t, httpClient(server), server.URL+"/api/")
	ctx := context.Background()
	want := Note{ID: 7, Title: "Go", CreatedAt: createdAt}
	got, err := client.Create(ctx, NoteInput{Title: "Go"})
	checkNote(t, got, err, want)
	got, err = client.Get(ctx, 7)
	checkNote(t, got, err, want)

	body := "second"
	want.Title, want.Body, want.Pinned = "updated", &body, true
	got, err = client.Replace(ctx, 7, NoteInput{Title: want.Title, Body: want.Body, Pinned: want.Pinned})
	checkNote(t, got, err, want)
	got, err = client.Patch(ctx, 7, map[string]any{})
	checkNote(t, got, err, want)

	want.Body = nil
	got, err = client.Patch(ctx, 7, map[string]any{"body": nil})
	checkNote(t, got, err, want)
	empty := ""
	want.Body, want.Pinned = &empty, false
	got, err = client.Patch(ctx, 7, map[string]any{"body": "", "pinned": false})
	checkNote(t, got, err, want)
	if err := client.Delete(ctx, 7); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := client.Get(ctx, 7); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("Get after Delete = %v; want an error containing 404", err)
	}

	expectedMethods := []string{"POST", "GET", "PUT", "PATCH", "PATCH", "PATCH", "DELETE", "GET"}
	expectedBodies := []map[string]any{
		{"title": "Go", "body": nil, "pinned": false},
		nil,
		{"title": "updated", "body": "second", "pinned": true},
		{},
		{"body": nil},
		{"body": "", "pinned": false},
		nil,
		nil,
	}
	mu.Lock()
	observed := append([]request(nil), requests...)
	mu.Unlock()
	if len(observed) != len(expectedMethods) {
		t.Fatalf("server received %d requests; want %d", len(observed), len(expectedMethods))
	}
	for i, received := range observed {
		wantPath := "/api/notes/7"
		if i == 0 {
			wantPath = "/api/notes"
		}
		if received.method != expectedMethods[i] || received.path != wantPath || received.accept != "application/json" {
			t.Errorf("request %d = %+v; want %s %s with JSON Accept", i, received, expectedMethods[i], wantPath)
		}
		if expectedBodies[i] == nil {
			if received.body != "" {
				t.Errorf("request %d body = %q; want an empty body", i, received.body)
			}
			continue
		}
		if received.contentType != "application/json" {
			t.Errorf("request %d Content-Type = %q; want application/json", i, received.contentType)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(received.body), &payload); err != nil {
			t.Fatalf("request %d contains invalid JSON: %v", i, err)
		}
		if !reflect.DeepEqual(payload, expectedBodies[i]) {
			t.Errorf("request %d JSON = %#v; want %#v", i, payload, expectedBodies[i])
		}
	}
}

func TestClientRejectsUnexpectedHTTPStatus(t *testing.T) {
	cases := []struct {
		method string
		status int
	}{
		{"GET", 400}, {"GET", 404}, {"GET", 409},
		{"POST", 200}, {"GET", 201}, {"PUT", 201}, {"PATCH", 201}, {"DELETE", 200},
	}
	for _, tc := range cases {
		t.Run(tc.method+"_"+strconv.Itoa(tc.status), func(t *testing.T) {
			server := responseServer(t, tc.status, "failure")
			client := newClient(t, httpClient(server), server.URL)
			err := callMethod(client, tc.method)
			if err == nil || !strings.Contains(err.Error(), strconv.Itoa(tc.status)) {
				t.Fatalf("%s error = %v; want status %d", tc.method, err, tc.status)
			}
		})
	}
}

func TestClientRequiresOneJSONObjectResponse(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"empty", "", true},
		{"malformed", "{", true},
		{"null", "null", true},
		{"array", "[]", true},
		{"second_document", noteJSON + " {}", true},
		{"trailing_garbage", noteJSON + " trailing", true},
		{"whitespace_tail_and_extra_field", noteJSON + " \n\t", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := responseServer(t, http.StatusOK, tc.body)
			client := newClient(t, httpClient(server), server.URL)
			note, err := client.Get(context.Background(), 7)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Get error = %v; wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && (note.ID != 7 || note.Title != "note" || note.CreatedAt.IsZero()) {
				t.Fatalf("decoded note = %#v; want typed fields including created_at", note)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewClientRejectsInvalidConfiguration(t *testing.T) {
	cases := []struct {
		name      string
		nilClient bool
		timeout   time.Duration
		baseURL   string
	}{
		{"nil_client", true, testTimeout, "http://example.test"},
		{"zero_timeout", false, 0, "http://example.test"},
		{"negative_timeout", false, -time.Second, "http://example.test"},
		{"relative_base", false, testTimeout, "/api"},
		{"non_http_base", false, testTimeout, "ftp://example.test"},
		{"query_in_base", false, testTimeout, "http://example.test?x=1"},
		{"fragment_in_base", false, testTimeout, "http://example.test#part"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := &http.Client{Timeout: tc.timeout}
			if tc.nilClient {
				httpClient = nil
			}
			client, err := NewClient(httpClient, tc.baseURL)
			if err == nil || client != nil {
				t.Fatalf("NewClient = %v, %v; want nil and an error", client, err)
			}
		})
	}
}

func TestClientRejectsNonpositiveIDsBeforeHTTP(t *testing.T) {
	for _, id := range []int64{0, -1} {
		t.Run(strconv.FormatInt(id, 10), func(t *testing.T) {
			var requests atomic.Int32
			httpClient := &http.Client{Timeout: testTimeout, Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				requests.Add(1)
				return nil, errors.New("unexpected HTTP call")
			})}
			client := newClient(t, httpClient, "http://example.test")
			ctx := context.Background()
			results := []error{}
			_, err := client.Get(ctx, id)
			results = append(results, err)
			_, err = client.Replace(ctx, id, NoteInput{})
			results = append(results, err)
			_, err = client.Patch(ctx, id, map[string]any{})
			results = append(results, err)
			results = append(results, client.Delete(ctx, id))
			for i, result := range results {
				if result == nil {
					t.Errorf("operation %d accepted ID %d", i, id)
				}
			}
			if requests.Load() != 0 {
				t.Fatalf("invalid IDs caused %d HTTP requests", requests.Load())
			}
		})
	}
}

func TestClientPatchEncodingErrorHappensBeforeHTTP(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, noteJSON)
	}))
	defer server.Close()
	client := newClient(t, httpClient(server), server.URL)
	_, err := client.Patch(context.Background(), 7, map[string]any{"body": make(chan int)})
	var unsupported *json.UnsupportedTypeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Patch error = %v; want json.UnsupportedTypeError", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("encoding failure caused %d HTTP requests", requests.Load())
	}
}

type trackedBody struct {
	reader io.Reader
	closed bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	return b.reader.Read(p)
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

type failedReader struct{ err error }

func (r failedReader) Read([]byte) (int, error) { return 0, r.err }

func TestClientClosesBodiesAndPreservesReadErrors(t *testing.T) {
	readErr := errors.New("response read failed")
	cases := []struct {
		name       string
		method     string
		status     int
		reader     io.Reader
		wantErr    bool
		errorCause error
	}{
		{"create", "POST", 201, strings.NewReader(noteJSON), false, nil},
		{"get", "GET", 200, strings.NewReader(noteJSON), false, nil},
		{"replace", "PUT", 200, strings.NewReader(noteJSON), false, nil},
		{"patch", "PATCH", 200, strings.NewReader(noteJSON), false, nil},
		{"delete", "DELETE", 204, strings.NewReader(""), false, nil},
		{"invalid_json", "GET", 200, strings.NewReader("{"), true, nil},
		{"http_error", "GET", 409, strings.NewReader("conflict"), true, nil},
		{"read_error", "GET", 200, failedReader{readErr}, true, readErr},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := &trackedBody{reader: tc.reader}
			httpClient := &http.Client{Timeout: testTimeout, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tc.status, Header: make(http.Header),
					Body: body, ContentLength: -1, Request: r,
				}, nil
			})}
			client := newClient(t, httpClient, "http://example.test")
			err := callMethod(client, tc.method)
			if (err != nil) != tc.wantErr {
				t.Fatalf("%s error = %v; wantErr %v", tc.method, err, tc.wantErr)
			}
			if tc.errorCause != nil && !errors.Is(err, tc.errorCause) {
				t.Fatalf("error = %v; want cause %v", err, tc.errorCause)
			}
			if !body.closed {
				t.Fatal("response body was left open")
			}
		})
	}
}

func TestClientPreservesTransportAndContextCauses(t *testing.T) {
	cause := errors.New("transport failed")
	transportClient := &http.Client{Timeout: testTimeout, Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, cause
	})}
	client := newClient(t, transportClient, "http://example.test")
	if _, err := client.Get(context.Background(), 7); !errors.Is(err, cause) {
		t.Fatalf("Get error = %v; want transport cause", err)
	}
	server := responseServer(t, http.StatusOK, noteJSON)
	client = newClient(t, httpClient(server), server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Get(ctx, 7); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v; want context.Canceled", err)
	}
}
