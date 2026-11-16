package handler_test

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"testing"

	"example.com/go-practice/04_crud/d01_notes/internal/testdb"
)

func TestDatabaseErrorIsLogged(t *testing.T) {
	db := testdb.Open(t)
	if _, err := db.ExecContext(testContext(t), "DROP TABLE notes"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	server := startServer(t, db)
	res := request(t, server, "GET", "/notes/1", "", "")
	expectError(t, res, http.StatusInternalServerError)
	if strings.TrimSpace(string(res.body)) != "internal server error" {
		t.Fatalf("response body = %q; want generic error", res.body)
	}
	for _, detail := range []string{"GET /notes/1", "42P01"} {
		if !strings.Contains(output.String(), detail) {
			t.Errorf("log = %q; want %q", output.String(), detail)
		}
	}
}
