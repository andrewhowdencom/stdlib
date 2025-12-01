package http

import (
	"testing"
	"time"
)

func TestNewServer_Defaults(t *testing.T) {
	s, err := NewServer(":0", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.server.ReadTimeout != 2*time.Second {
		t.Errorf("expected default ReadTimeout 2s, got %v", s.server.ReadTimeout)
	}
	if s.server.WriteTimeout != 2*time.Second {
		t.Errorf("expected default WriteTimeout 2s, got %v", s.server.WriteTimeout)
	}
	if s.server.IdleTimeout != 2*time.Second {
		t.Errorf("expected default IdleTimeout 2s, got %v", s.server.IdleTimeout)
	}
}

func TestNewServer_Options(t *testing.T) {
	s, err := NewServer(":0", nil,
		WithReadTimeout(100*time.Millisecond),
		WithWriteTimeout(200*time.Millisecond),
		WithIdleTimeout(300*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.server.ReadTimeout != 100*time.Millisecond {
		t.Errorf("expected ReadTimeout 100ms, got %v", s.server.ReadTimeout)
	}
	if s.server.WriteTimeout != 200*time.Millisecond {
		t.Errorf("expected WriteTimeout 200ms, got %v", s.server.WriteTimeout)
	}
	if s.server.IdleTimeout != 300*time.Millisecond {
		t.Errorf("expected IdleTimeout 300ms, got %v", s.server.IdleTimeout)
	}
}
