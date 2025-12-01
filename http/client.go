package http

import (
	"errors"
	"net"
	stdhttp "net/http"
	"time"
)

// ClientOption is a function that configures a http.Client.
type ClientOption func(*stdhttp.Client) error

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

// NewClient returns a new http.Client with sane defaults for internal traffic.
// Default timeouts:
// - Total: 2s
// - Connect: 500ms
// - TLS Handshake: 500ms
// - Response Header: 1.5s
func NewClient(opts ...ClientOption) (*stdhttp.Client, error) {
	// Custom Transport with aggressive defaults
	t := &stdhttp.Transport{
		Proxy: stdhttp.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   500 * time.Millisecond,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   500 * time.Millisecond,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 1500 * time.Millisecond,
	}

	c := &stdhttp.Client{
		Transport: t,
		Timeout:   2 * time.Second,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}
