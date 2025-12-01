package http

import (
	"net/http"
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

	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected transport to be *http.Transport")
	}

	if tr.TLSHandshakeTimeout != 500*time.Millisecond {
		t.Errorf("expected TLSHandshakeTimeout 500ms, got %v", tr.TLSHandshakeTimeout)
	}

	if tr.ResponseHeaderTimeout != 1500*time.Millisecond {
		t.Errorf("expected ResponseHeaderTimeout 1.5s, got %v", tr.ResponseHeaderTimeout)
	}
}

func TestNewClient_Options(t *testing.T) {
	c, err := NewClient(
		WithTimeout(5*time.Second),
		WithTLSHandshakeTimeout(1*time.Second),
		WithResponseHeaderTimeout(3*time.Second),
		WithConnectTimeout(1*time.Second),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", c.Timeout)
	}

	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected transport to be *http.Transport")
	}

	if tr.TLSHandshakeTimeout != 1*time.Second {
		t.Errorf("expected TLSHandshakeTimeout 1s, got %v", tr.TLSHandshakeTimeout)
	}

	if tr.ResponseHeaderTimeout != 3*time.Second {
		t.Errorf("expected ResponseHeaderTimeout 3s, got %v", tr.ResponseHeaderTimeout)
	}
}

func TestNewClient_MaxIdleConns(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr := c.Transport.(*http.Transport)
	if tr.MaxIdleConns != 100 {
		t.Errorf("expected default MaxIdleConns 100, got %d", tr.MaxIdleConns)
	}

	c2, err := NewClient(WithMaxIdleConns(50))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr2 := c2.Transport.(*http.Transport)
	if tr2.MaxIdleConns != 50 {
		t.Errorf("expected MaxIdleConns 50, got %d", tr2.MaxIdleConns)
	}
}
