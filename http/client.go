package http

import (
	"errors"
	"net"
	stdhttp "net/http"
	"time"
)

// ClientOption is a function that configures a http.Client.
type ClientOption func(*stdhttp.Client) error

// defaultClientOptions defines the aggressive defaults for the client.
var defaultClientOptions = []ClientOption{
	WithTimeout(2 * time.Second),
	WithConnectTimeout(500 * time.Millisecond),
	WithTLSHandshakeTimeout(500 * time.Millisecond),
	WithResponseHeaderTimeout(1500 * time.Millisecond),
	// MaxIdleConns: 100 is the standard library default, but we make it explicit here.
	WithMaxIdleConns(100),
	WithIdleConnTimeout(90 * time.Second),
	WithExpectContinueTimeout(1 * time.Second),
}

// WithTimeout sets the total request timeout (Client.Timeout).
func WithTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		c.Timeout = d
		return nil
	}
}

// WithConnectTimeout sets the connection timeout (Dialer.Timeout).
func WithConnectTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, ok := c.Transport.(*stdhttp.Transport)
		if !ok {
			return errors.New("transport is not *http.Transport")
		}
		// Re-create the Dialer with the new timeout, preserving KeepAlive
		t.DialContext = (&net.Dialer{
			Timeout:   d,
			KeepAlive: 30 * time.Second,
		}).DialContext
		return nil
	}
}

// WithTLSHandshakeTimeout sets the TLS handshake timeout.
func WithTLSHandshakeTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, ok := c.Transport.(*stdhttp.Transport)
		if !ok {
			return errors.New("transport is not *http.Transport")
		}
		t.TLSHandshakeTimeout = d
		return nil
	}
}

// WithResponseHeaderTimeout sets the response header timeout.
func WithResponseHeaderTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, ok := c.Transport.(*stdhttp.Transport)
		if !ok {
			return errors.New("transport is not *http.Transport")
		}
		t.ResponseHeaderTimeout = d
		return nil
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(n int) ClientOption {
	return func(c *stdhttp.Client) error {
		t, ok := c.Transport.(*stdhttp.Transport)
		if !ok {
			return errors.New("transport is not *http.Transport")
		}
		t.MaxIdleConns = n
		return nil
	}
}

// WithIdleConnTimeout sets the idle connection timeout.
func WithIdleConnTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, ok := c.Transport.(*stdhttp.Transport)
		if !ok {
			return errors.New("transport is not *http.Transport")
		}
		t.IdleConnTimeout = d
		return nil
	}
}

// WithExpectContinueTimeout sets the Expect-Continue timeout.
func WithExpectContinueTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, ok := c.Transport.(*stdhttp.Transport)
		if !ok {
			return errors.New("transport is not *http.Transport")
		}
		t.ExpectContinueTimeout = d
		return nil
	}
}

// NewClient returns a new http.Client with sane defaults for internal traffic.
// Defaults are defined in defaultClientOptions.
func NewClient(opts ...ClientOption) (*stdhttp.Client, error) {
	// Initialize Transport with base values that are not timeouts
	t := &stdhttp.Transport{
		Proxy:             stdhttp.ProxyFromEnvironment,
		ForceAttemptHTTP2: true,
	}

	c := &stdhttp.Client{
		Transport: t,
	}

	// Apply defaults
	for _, opt := range defaultClientOptions {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	// Apply user overrides
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}
