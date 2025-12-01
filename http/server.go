package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Server wraps net/http.Server to provide defaults and graceful shutdown.
type Server struct {
	server          *stdhttp.Server
	tracer          trace.Tracer
	meter           metric.Meter
	openConnections metric.Int64UpDownCounter
	activeRequests  metric.Int64UpDownCounter
}

// ServerOption configures the Server.
type ServerOption func(*Server) error

// defaultServerOptions defines the aggressive defaults for the server.
var defaultServerOptions = []ServerOption{
	WithReadTimeout(2 * time.Second),
	WithWriteTimeout(2 * time.Second),
	WithIdleTimeout(2 * time.Second),
}

// WithServerTracerProvider configures the server with a specific tracer provider.
func WithServerTracerProvider(tp trace.TracerProvider) ServerOption {
	return func(s *Server) error {
		s.tracer = tp.Tracer(instrumentationName)
		return nil
	}
}

// WithServerMeterProvider configures the server with a specific meter provider.
func WithServerMeterProvider(mp metric.MeterProvider) ServerOption {
	return func(s *Server) error {
		s.meter = mp.Meter(instrumentationName)
		return nil
	}
}

// WithReadTimeout sets the ReadTimeout.
func WithReadTimeout(d time.Duration) ServerOption {
	return func(s *Server) error {
		s.server.ReadTimeout = d
		return nil
	}
}

// WithWriteTimeout sets the WriteTimeout.
func WithWriteTimeout(d time.Duration) ServerOption {
	return func(s *Server) error {
		s.server.WriteTimeout = d
		return nil
	}
}

// WithIdleTimeout sets the IdleTimeout.
func WithIdleTimeout(d time.Duration) ServerOption {
	return func(s *Server) error {
		s.server.IdleTimeout = d
		return nil
	}
}

// NewServer creates a new Server with defaults.
// Defaults are defined in defaultServerOptions.
func NewServer(addr string, handler stdhttp.Handler, opts ...ServerOption) (*Server, error) {
	srv := &stdhttp.Server{
		Addr:    addr,
		Handler: handler,
	}

	s := &Server{server: srv}

	// Apply defaults
	for _, opt := range defaultServerOptions {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	// Apply user overrides
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	// Finalize instrumentation
	if s.tracer == nil {
		s.tracer = otel.GetTracerProvider().Tracer(instrumentationName)
	}
	if s.meter == nil {
		s.meter = otel.GetMeterProvider().Meter(instrumentationName)
	}

	var err error
	s.openConnections, err = s.meter.Int64UpDownCounter("http.server.open_connections")
	if err != nil {
		return nil, err
	}
	s.activeRequests, err = s.meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		return nil, err
	}

	s.server.ConnState = func(c net.Conn, cs stdhttp.ConnState) {
		switch cs {
		case stdhttp.StateNew:
			s.openConnections.Add(context.Background(), 1)
		case stdhttp.StateClosed, stdhttp.StateHijacked:
			s.openConnections.Add(context.Background(), -1)
		}
	}

	// Wrap handler
	if srv.Handler == nil {
		srv.Handler = stdhttp.DefaultServeMux
	}
	s.server.Handler = &instrumentedHandler{
		base:           srv.Handler,
		tracer:         s.tracer,
		meter:          s.meter,
		activeRequests: s.activeRequests,
	}

	return s, nil
}

// Run starts the server and waits for a signal to shutdown.
func (s *Server) Run() error {
	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		// Graceful shutdown
		// We'll use a timeout for the shutdown itself.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Ask the server to shutdown gracefully.
		if err := s.server.Shutdown(ctx); err != nil {
			// We return that error.
			return fmt.Errorf("could not stop server gracefully: %w (signal: %v)", err, sig)
		}
	}

	return nil
}
