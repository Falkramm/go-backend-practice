package h02

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var errNotImplemented = errors.New("not implemented")

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      *string   `json:"body"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
}

type NoteInput struct {
	Title  string  `json:"title"`
	Body   *string `json:"body"`
	Pinned bool    `json:"pinned"`
}

type Client struct {
	hc      *http.Client
	baseUrl string
}

func NewClient(httpClient *http.Client, baseURL string) (*Client, error) {
	if httpClient == nil || httpClient.Timeout <= 0 {
		return nil, errors.New("invalid http client")
	}
	return &Client{
		hc:      httpClient,
		baseUrl: strings.TrimRight(baseURL, "/"),
	}, nil
}

func (c *Client) Create(ctx context.Context, input NoteInput) (Note, error) {
	if ctx.Err() != nil {
		return Note{}, fmt.Errorf("create: %w", ctx.Err())
	}

	u, err := url.Parse(c.baseUrl + "/notes")
	if err != nil {
		return Note{}, fmt.Errorf("create: %w", err)
	}

	data, err := json.Marshal(input)

	if err != nil {
		return Note{}, fmt.Errorf("create: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u.String(),
		bytes.NewReader(data),
	)

	if err != nil {
		return Note{}, fmt.Errorf("create: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)

	if err != nil {
		return Note{}, fmt.Errorf("create: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return Note{}, fmt.Errorf("got status code: %v; want: %v", resp.StatusCode, http.StatusCreated)
	}

	var res Note
	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&res); err != nil {
		return Note{}, fmt.Errorf("create: %w", err)
	}

	return res, nil
}

func (c *Client) Get(ctx context.Context, id int64) (Note, error) {
	if ctx.Err() != nil {
		return Note{}, fmt.Errorf("get: %w", ctx.Err())
	}

	u, err := url.Parse(c.baseUrl + fmt.Sprintf("/notes/%d", id))
	if err != nil {
		return Note{}, fmt.Errorf("get: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		u.String(),
		nil,
	)
	if err != nil {
		return Note{}, fmt.Errorf("get: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)

	if err != nil {
		return Note{}, fmt.Errorf("get: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Note{}, fmt.Errorf("statuc code got: %v; want: %v", resp.StatusCode, http.StatusOK)
	}

	var res Note
	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&res); err != nil {
		return Note{}, fmt.Errorf("get: %w", err)
	}

	return res, nil

}

func (c *Client) Replace(ctx context.Context, id int64, input NoteInput) (Note, error) {
	if ctx.Err() != nil {
		return Note{}, fmt.Errorf("replace: %w", ctx.Err())
	}

	u, err := url.Parse(c.baseUrl + fmt.Sprintf("/notes/%d", id))

	if err != nil {
		return Note{}, fmt.Errorf("replace: %w", err)
	}

	data, err := json.Marshal(input)

	if err != nil {
		return Note{}, fmt.Errorf("replace: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		u.String(),
		bytes.NewReader(data),
	)

	if err != nil {
		return Note{}, fmt.Errorf("replace: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)

	if err != nil {
		return Note{}, fmt.Errorf("replace: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Note{}, fmt.Errorf("statuc code got: %v; want: %v", resp.StatusCode, http.StatusOK)
	}

	var res Note
	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&res); err != nil {
		return Note{}, fmt.Errorf("replace: %w", err)
	}

	return res, nil
}

func (c *Client) Patch(ctx context.Context, id int64, changes map[string]any) (Note, error) {
	if ctx.Err() != nil {
		return Note{}, fmt.Errorf("replace: %w", ctx.Err())
	}

	u, err := url.Parse(c.baseUrl + fmt.Sprintf("/notes/%d", id))

	if err != nil {
		return Note{}, fmt.Errorf("patch: %w", err)
	}

	data, err := json.Marshal(changes)

	if err != nil {
		return Note{}, fmt.Errorf("patch: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		u.String(),
		bytes.NewReader(data),
	)

	if err != nil {
		return Note{}, fmt.Errorf("patch: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)

	if err != nil {
		return Note{}, fmt.Errorf("patch: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Note{}, fmt.Errorf("status got: %v; want: %v", resp.StatusCode, http.StatusOK)
	}

	var res Note
	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&res); err != nil {
		return Note{}, fmt.Errorf("patch: %w", err)
	}

	return res, nil
}

func (c *Client) Delete(ctx context.Context, id int64) error {
	if ctx.Err() != nil {
		return fmt.Errorf("delete: %w", ctx.Err())
	}

	u, err := url.Parse(c.baseUrl + fmt.Sprintf("/notes/%d", id))

	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		u.String(),
		nil,
	)

	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)

	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("status got: %v; want: %v", resp.StatusCode, http.StatusNoContent)
	}

	return nil
}
