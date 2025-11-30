package http

import (
	"time"

	stdhttp "net/http"
)

// ClientOption is a function that configures a http.Client.
type ClientOption func(*stdhttp.Client) error

// WithTimeout sets the timeout for the client.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		c.Timeout = d
		return nil
	}
}

// NewClient returns a new http.Client with sane defaults for internal traffic.
// Default timeout is 2 seconds.
func NewClient(opts ...ClientOption) (*stdhttp.Client, error) {
	c := &stdhttp.Client{
		Timeout: 2 * time.Second,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}
