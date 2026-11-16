package h01

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

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

func testClient(server *httptest.Server) *http.Client {
	client := server.Client()
	client.Timeout = testTimeout
	return client
}

func copyValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	copy := make(url.Values, len(values))
	for key, entries := range values {
		copy[key] = append([]string(nil), entries...)
	}
	return copy
}

func TestGetItemsSendsQueryHeadersAndReusesClient(t *testing.T) {
	type request struct {
		method    string
		path      string
		rawQuery  string
		accept    string
		requestID string
	}
	requests := make(chan request, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed := request{
			method: r.Method, path: r.URL.Path, rawQuery: r.URL.RawQuery,
			accept: r.Header.Get("Accept"), requestID: r.Header.Get("X-Request-ID"),
		}
		select {
		case requests <- observed:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, `[{"id":7,"name":"one"},{"id":8,"name":"two"}]`)
	}))
	defer server.Close()
	client := testClient(server)
	originalTransport := client.Transport
	queries := []url.Values{
		{"search": {"tea & coffee/+?"}, "tag": {"go", "http"}, "empty": {""}},
		nil,
	}
	for _, query := range queries {
		before := copyValues(query)
		expectedQuery := query.Encode()
		items, err := GetItems(context.Background(), client, server.URL+"/v1/items?stale=remove", query, "request-17", 1024)
		if err != nil {
			t.Fatalf("GetItems: %v", err)
		}
		if !reflect.DeepEqual(items, []Item{{ID: 7, Name: "one"}, {ID: 8, Name: "two"}}) {
			t.Fatalf("items = %#v; want two typed items", items)
		}
		select {
		case got := <-requests:
			want := request{"GET", "/v1/items", expectedQuery, "application/json", "request-17"}
			if got != want {
				t.Fatalf("server received %+v; want %+v", got, want)
			}
		case <-time.After(testTimeout):
			t.Fatal("the server did not receive the request")
		}
		if !reflect.DeepEqual(query, before) {
			t.Fatalf("query changed from %#v to %#v", before, query)
		}
	}
	if client.Timeout != testTimeout || client.Transport != originalTransport {
		t.Fatal("GetItems changed the reusable client configuration")
	}
}

func TestGetItemsAcceptsSuccessfulStatusesAndEmptyArray(t *testing.T) {
	for _, status := range []int{200, 201, 206, 299} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			server := responseServer(t, status, "[]")
			items, err := GetItems(context.Background(), testClient(server), server.URL, nil, "", 2)
			if err != nil || len(items) != 0 {
				t.Fatalf("GetItems = %#v, %v; want an empty result and nil error", items, err)
			}
		})
	}
}

func TestGetItemsRequiresOneJSONArrayDocument(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"empty", 200, "", true},
		{"whitespace_only", 200, " \n\t", true},
		{"malformed", 200, "[{", true},
		{"object", 200, "{}", true},
		{"null", 200, "null", true},
		{"wrong_field_type", 200, `[{"id":"seven","name":"one"}]`, true},
		{"second_document", 200, "[] []", true},
		{"trailing_garbage", 200, "[] garbage", true},
		{"empty_204", 204, "", true},
		{"whitespace_tail", 200, "[] \n\t", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := responseServer(t, tc.status, tc.body)
			_, err := GetItems(context.Background(), testClient(server), server.URL, nil, "", 1024)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetItems error = %v; wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestGetItemsHTTPErrorIncludesStatus(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			server := responseServer(t, status, "failure body")
			_, err := GetItems(context.Background(), testClient(server), server.URL, nil, "", 8)
			if err == nil {
				t.Fatal("GetItems returned nil error for a failing HTTP status")
			}
			message := err.Error()
			if !strings.Contains(message, strconv.Itoa(status)) {
				t.Fatalf("HTTP error %q must contain status %d", message, status)
			}
		})
	}
}

func TestGetItemsBodyLimitBoundary(t *testing.T) {
	const payload = `[{"id":1,"name":"one"}]`
	cases := []struct {
		name  string
		body  string
		limit int64
		large bool
	}{
		{"exact_limit", payload, int64(len(payload)), false},
		{"one_byte_over", payload, int64(len(payload)) - 1, true},
		{"whitespace_counts", "[] ", 2, true},
		{"maximum_positive_limit", payload, 1<<63 - 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := responseServer(t, http.StatusOK, tc.body)
			_, err := GetItems(context.Background(), testClient(server), server.URL, nil, "", tc.limit)
			if tc.large {
				if err == nil {
					t.Fatal("GetItems accepted a response exceeding the body limit")
				}
			} else if err != nil {
				t.Fatalf("GetItems rejected an allowed body size: %v", err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGetItemsRejectsInvalidConfigurationBeforeHTTP(t *testing.T) {
	cases := []struct {
		name      string
		nilClient bool
		timeout   time.Duration
		limit     int64
	}{
		{"nil_client", true, testTimeout, 10},
		{"zero_timeout", false, 0, 10},
		{"negative_timeout", false, -time.Second, 10},
		{"zero_limit", false, testTimeout, 0},
		{"negative_limit", false, testTimeout, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			client := &http.Client{Timeout: tc.timeout, Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("unexpected HTTP call")
			})}
			if tc.nilClient {
				client = nil
			}
			_, err := GetItems(context.Background(), client, "http://example.test/items", nil, "", tc.limit)
			if err == nil {
				t.Fatal("GetItems accepted invalid configuration")
			}
			if calls.Load() != 0 {
				t.Fatal("GetItems performed HTTP before rejecting invalid configuration")
			}
		})
	}
}

func TestGetItemsRejectsInvalidEndpointBeforeHTTP(t *testing.T) {
	for _, endpoint := range []string{"", "/relative/items", "://broken"} {
		t.Run(endpoint, func(t *testing.T) {
			var calls atomic.Int32
			client := &http.Client{Timeout: testTimeout, Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("unexpected HTTP call")
			})}
			_, err := GetItems(context.Background(), client, endpoint, nil, "", 1024)
			if err == nil || calls.Load() != 0 {
				t.Fatalf("GetItems error = %v, HTTP calls = %d; want an error before HTTP", err, calls.Load())
			}
		})
	}
}

type trackedBody struct {
	reader io.Reader
	read   int64
	closed bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += int64(n)
	return n, err
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

type failedReader struct{ err error }

func (r failedReader) Read([]byte) (int, error) { return 0, r.err }

func TestGetItemsClosesBodiesAndBoundsReads(t *testing.T) {
	readErr := errors.New("body read failed")
	cases := []struct {
		name       string
		status     int
		reader     io.Reader
		limit      int64
		wantErr    bool
		errorCause error
	}{
		{"success", 200, strings.NewReader("[]"), 2, false, nil},
		{"invalid_json", 200, strings.NewReader("{"), 16, true, nil},
		{"http_error", 500, strings.NewReader("failure body beyond limit"), 8, true, nil},
		{"oversized_success", 200, strings.NewReader("[]     "), 2, true, nil},
		{"read_error", 200, failedReader{readErr}, 8, true, readErr},
		{"http_read_error", 500, failedReader{readErr}, 8, true, readErr},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := &trackedBody{reader: tc.reader}
			client := &http.Client{Timeout: testTimeout, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tc.status, Status: fmt.Sprintf("%d %s", tc.status, http.StatusText(tc.status)),
					Header: make(http.Header), Body: body, ContentLength: -1, Request: r,
				}, nil
			})}
			_, err := GetItems(context.Background(), client, "http://example.test/items", nil, "", tc.limit)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetItems error = %v; wantErr %v", err, tc.wantErr)
			}
			if tc.errorCause != nil && !errors.Is(err, tc.errorCause) {
				t.Fatalf("GetItems error = %v; want cause %v", err, tc.errorCause)
			}
			if tc.status >= 400 && tc.errorCause == nil && (err == nil || !strings.Contains(err.Error(), strconv.Itoa(tc.status))) {
				t.Fatalf("HTTP error = %v; want status %d", err, tc.status)
			}
			if !body.closed {
				t.Fatal("response body was left open")
			}
			if body.read > tc.limit+1 {
				t.Fatalf("read %d bytes; budget is %d", body.read, tc.limit+1)
			}
		})
	}
}

func TestGetItemsPreservesTransportErrorCause(t *testing.T) {
	cause := errors.New("transport failed")
	client := &http.Client{Timeout: testTimeout, Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, cause
	})}
	_, err := GetItems(context.Background(), client, "http://example.test/items", nil, "", 1024)
	if !errors.Is(err, cause) {
		t.Fatalf("GetItems error = %v; want transport cause %v", err, cause)
	}
}

func TestGetItemsPreservesCanceledContext(t *testing.T) {
	server := responseServer(t, http.StatusOK, "[]")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetItems(ctx, testClient(server), server.URL, nil, "", 1024)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetItems error = %v; want context.Canceled", err)
	}
}
