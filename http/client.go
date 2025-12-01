package http

import (
	"errors"
	"net"
	stdhttp "net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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

const instrumentationName = "github.com/andrewhowdencom/stdlib/http"

// getTransport returns the underlying *http.Transport from the client.
// It handles both direct *http.Transport and wrapped *InstrumentedTransport.
func getTransport(c *stdhttp.Client) (*stdhttp.Transport, error) {
	if t, ok := c.Transport.(*stdhttp.Transport); ok {
		return t, nil
	}
	if it, ok := c.Transport.(*InstrumentedTransport); ok {
		if t, ok := it.Base.(*stdhttp.Transport); ok {
			return t, nil
		}
	}
	return nil, errors.New("transport is not *http.Transport")
}

// ensureInstrumentedTransport ensures the client transport is wrapped in InstrumentedTransport.
func ensureInstrumentedTransport(c *stdhttp.Client) *InstrumentedTransport {
	if it, ok := c.Transport.(*InstrumentedTransport); ok {
		return it
	}
	it := &InstrumentedTransport{
		Base: c.Transport,
	}
	c.Transport = it
	return it
}

// WithClientTracerProvider configures the client with a specific tracer provider.
func WithClientTracerProvider(tp trace.TracerProvider) ClientOption {
	return func(c *stdhttp.Client) error {
		it := ensureInstrumentedTransport(c)
		it.Tracer = tp.Tracer(instrumentationName)
		return nil
	}
}

// WithClientMeterProvider configures the client with a specific meter provider.
func WithClientMeterProvider(mp metric.MeterProvider) ClientOption {
	return func(c *stdhttp.Client) error {
		it := ensureInstrumentedTransport(c)
		it.Meter = mp.Meter(instrumentationName)
		var err error
		it.duration, err = it.Meter.Float64Histogram("http.client.request.duration", metric.WithUnit("s"))
		return err
	}
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
		t, err := getTransport(c)
		if err != nil {
			return err
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
		t, err := getTransport(c)
		if err != nil {
			return err
		}
		t.TLSHandshakeTimeout = d
		return nil
	}
}

// WithResponseHeaderTimeout sets the response header timeout.
func WithResponseHeaderTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, err := getTransport(c)
		if err != nil {
			return err
		}
		t.ResponseHeaderTimeout = d
		return nil
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(n int) ClientOption {
	return func(c *stdhttp.Client) error {
		t, err := getTransport(c)
		if err != nil {
			return err
		}
		t.MaxIdleConns = n
		return nil
	}
}

// WithIdleConnTimeout sets the idle connection timeout.
func WithIdleConnTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, err := getTransport(c)
		if err != nil {
			return err
		}
		t.IdleConnTimeout = d
		return nil
	}
}

// WithExpectContinueTimeout sets the Expect-Continue timeout.
func WithExpectContinueTimeout(d time.Duration) ClientOption {
	return func(c *stdhttp.Client) error {
		t, err := getTransport(c)
		if err != nil {
			return err
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
