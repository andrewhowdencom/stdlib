package http

import (
	"testing"
	"time"
)

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Timeout != 2*time.Second {
		t.Errorf("expected default timeout 2s, got %v", c.Timeout)
	}
}

func TestNewClient_WithTimeout(t *testing.T) {
	c, err := NewClient(WithTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", c.Timeout)
	}
}
