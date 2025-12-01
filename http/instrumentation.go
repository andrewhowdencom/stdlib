package http

import (
	stdhttp "net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedTransport wraps http.RoundTripper to inject trace context and attributes.
type InstrumentedTransport struct {
	Base     stdhttp.RoundTripper
	Tracer   trace.Tracer
	Meter    metric.Meter
	duration metric.Float64Histogram
}

// RoundTrip implements http.RoundTripper.
func (t *InstrumentedTransport) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	start := time.Now()

	// 1. Inject propagation headers
	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))

	// 2. Check for existing span
	ctx := req.Context()
	span := trace.SpanFromContext(ctx)

	// 3. Enrich if recording
	if span.IsRecording() {
		span.SetAttributes(clientRequestAttrs(req)...)
	}

	// 4. Call Base
	// Ensure Base is not nil
	rt := t.Base
	if rt == nil {
		rt = stdhttp.DefaultTransport
	}
	resp, err := rt.RoundTrip(req)

	// 5. Enrich response
	if span.IsRecording() {
		if err != nil {
			span.RecordError(err)
		}
		if resp != nil {
			span.SetAttributes(clientResponseAttrs(resp)...)
		}
	}

	// 6. Record Metrics
	if t.duration != nil {
		attrs := clientRequestAttrs(req)
		if resp != nil {
			attrs = append(attrs, clientResponseAttrs(resp)...)
		}
		t.duration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
	}

	return resp, err
}

// instrumentedHandler wraps http.Handler to extract trace context and start spans.
type instrumentedHandler struct {
	base     stdhttp.Handler
	tracer   trace.Tracer
	meter    metric.Meter
	duration metric.Float64Histogram
}

// ServeHTTP implements http.Handler.
func (h *instrumentedHandler) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	start := time.Now()

	// 1. Extract propagation headers
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// 2. Start Span (Server Kind)
	// NOTE: The handler can overwrite the span name later in the request.
	spanName := "HTTP " + r.Method
	ctx, span := h.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// 3. Add Request Attributes
	span.SetAttributes(serverRequestAttrs(r)...)

	// 4. Wrap ResponseWriter to capture status code
	rr := &responseRecorder{ResponseWriter: w, statusCode: stdhttp.StatusOK}

	// 5. Serve
	h.base.ServeHTTP(rr, r.WithContext(ctx))

	// 6. Add Response Attributes
	span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(rr.statusCode))

	// 7. Record Metrics
	if h.duration != nil {
		attrs := serverRequestAttrs(r)
		attrs = append(attrs, semconv.HTTPResponseStatusCodeKey.Int(rr.statusCode))
		h.duration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
	}
}

type responseRecorder struct {
	stdhttp.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Helpers for extracting attributes

func clientRequestAttrs(req *stdhttp.Request) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(req.Method),
	}
	if req.URL != nil {
		attrs = append(attrs, semconv.URLPathKey.String(req.URL.Path))
		attrs = append(attrs, semconv.URLSchemeKey.String(req.URL.Scheme))
		attrs = append(attrs, semconv.ServerAddressKey.String(req.URL.Hostname()))
	}
	return attrs
}

func clientResponseAttrs(resp *stdhttp.Response) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPResponseStatusCodeKey.Int(resp.StatusCode),
	}
}

func serverRequestAttrs(req *stdhttp.Request) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(req.Method),
		semconv.URLPathKey.String(req.URL.Path),
		semconv.URLSchemeKey.String(req.URL.Scheme),
		semconv.UserAgentOriginalKey.String(req.UserAgent()),
	}
	return attrs
}
