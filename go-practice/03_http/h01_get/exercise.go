package h01

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
)

var (
	ErrBodyTooLarge   = errors.New("response body too large")
	errNotImplemented = errors.New("not implemented")
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func GetItems(ctx context.Context, client *http.Client, endpoint string, query url.Values, requestID string, maxBodyBytes int64) ([]Item, error) {
	if client == nil || client.Timeout <= 0 {
		return nil, errors.New("invalid client")
	}

	if maxBodyBytes <= 0 {
		return nil, errors.New("invalid max body bytes")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("get items: %w", ctx.Err())
	}

	u, err := url.Parse(endpoint)

	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	if !u.IsAbs() || u.Host == "" {
		return nil, errors.New("invalid url")
	}

	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		u.String(),
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", requestID)

	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	defer resp.Body.Close()

	readLimit := maxBodyBytes
	if readLimit < math.MaxInt64 {
		readLimit++
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, readLimit))

	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("want status code: %v; got: %v", http.StatusOK, resp.StatusCode)
	}

	if int64(len(body)) > maxBodyBytes {
		return nil, errors.New("invalid body size")
	}

	var items []Item

	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	if items == nil {
		return nil, errors.New("invalid json body")
	}

	return items, nil
}
