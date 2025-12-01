package http

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server wraps net/http.Server to provide defaults and graceful shutdown.
type Server struct {
	server *stdhttp.Server
}

// ServerOption configures the Server.
type ServerOption func(*Server) error

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

// defaultServerOptions defines the aggressive defaults for the server.
var defaultServerOptions = []ServerOption{
	WithReadTimeout(2 * time.Second),
	WithWriteTimeout(2 * time.Second),
	WithIdleTimeout(2 * time.Second),
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
