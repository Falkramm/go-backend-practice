package day15

import (
	"http"
	"context"
)

type Config struct {
	Timeout 			time.Duration
	MaxRetries 			int
	MaxIdleConnsPerHost int
	BaseBackoff 		time.Duration
	MaxBackoff 			time.Duration
}

type Client struct {
	hc 	*http.Client
	cfg Config
}

func NewClient(cfg Config) *Client {
	return &Client{
		hc: &http.Client{
			transport :http.Transport{
				MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
				MaxIdleConns: cfg.MaxIdleConnsPerHost * 4,
			},
			Timeout: cfg.Timeout,
		},
		cfg: cfg,
	}
}

func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastError error

	for attempt := range c.cfg.MaxRetries {
		// перезалить тело

		resp, err := c.hc.Do(req.WithContext(ctx))

		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		} else {
			return resp, nil
		}
	}
}


func main() {

}
