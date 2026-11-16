package parallelget

import (
	"context"
	"net/http"
)

type Result struct {
	URL        string
	StatusCode int
	Err        error
}

// FetchURLs возвращает результаты в порядке urls.
// ctx и client передаются вызывающим кодом.
func FetchURLs(ctx context.Context, client *http.Client, urls []string) []Result {
	return nil
}
