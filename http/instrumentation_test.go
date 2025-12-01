package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestClientInstrumentation(t *testing.T) {
	// Set global propagator for test
	otel.SetTextMapPropagator(propagation.TraceContext{})

	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))

	mockTransport := &mockRoundTripper{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("traceparent") == "" {
				t.Error("Traceparent header missing")
			}
			return &http.Response{
				StatusCode: 200,
				Request:    req,
			}, nil
		},
	}

	client, err := NewClient(
		func(c *http.Client) error {
			c.Transport = &InstrumentedTransport{
				Base: mockTransport,
			}
			return nil
		},
		WithClientTracerProvider(tp),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent-span")
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com/foo", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}
	s := spans[0]
	if s.Name != "parent-span" {
		t.Errorf("Expected span name parent-span, got %s", s.Name)
	}

	attrs := s.Attributes
	if !hasAttr(attrs, semconv.HTTPRequestMethodKey.String("GET")) {
		t.Error("Missing http.request.method=GET")
	}
	if !hasAttr(attrs, semconv.URLPathKey.String("/foo")) {
		t.Error("Missing url.path=/foo")
	}
	if !hasAttr(attrs, semconv.HTTPResponseStatusCodeKey.Int(200)) {
		t.Error("Missing http.response.status_code=200")
	}
}

func TestServerInstrumentation(t *testing.T) {
	// Set global propagator for test
	otel.SetTextMapPropagator(propagation.TraceContext{})

	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})

	srv, err := NewServer(":0", handler, WithServerTracerProvider(tp))
	if err != nil {
		t.Fatal(err)
	}

	// Access internal server handler
	serverHandler := srv.server.Handler

	req := httptest.NewRequest("POST", "/bar", nil)
	w := httptest.NewRecorder()

	serverHandler.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}
	s := spans[0]
	if s.Name != "HTTP POST" {
		t.Errorf("Expected span name HTTP POST, got %s", s.Name)
	}

	attrs := s.Attributes
	if !hasAttr(attrs, semconv.HTTPRequestMethodKey.String("POST")) {
		t.Error("Missing http.request.method=POST")
	}
	if !hasAttr(attrs, semconv.URLPathKey.String("/bar")) {
		t.Error("Missing url.path=/bar")
	}
	if !hasAttr(attrs, semconv.HTTPResponseStatusCodeKey.Int(201)) {
		t.Error("Missing http.response.status_code=201")
	}
}

func hasAttr(attrs []attribute.KeyValue, want attribute.KeyValue) bool {
	for _, a := range attrs {
		if a.Key == want.Key && a.Value.Emit() == want.Value.Emit() {
			return true
		}
	}
	return false
}

type mockRoundTripper struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}
