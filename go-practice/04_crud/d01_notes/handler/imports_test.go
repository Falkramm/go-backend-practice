package handler_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHandlerUsesOnlyAllowedImports(t *testing.T) {
	allowed := map[string]bool{
		"bytes":         true,
		"database/sql":  true,
		"encoding/json": true,
		"errors":        true,
		"io":            true,
		"log":           true,
		"net/http":      true,
		"strconv":       true,
		"strings":       true,
		"time":          true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if !allowed[path] {
				t.Errorf("%s imports %q outside the allowed handler libraries", name, path)
			}
		}
	}
}
