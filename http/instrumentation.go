package http

import (
	stdhttp "net/http"
	"time"

	"net/http/httptrace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedTransport wraps http.RoundTripper to inject trace context and attributes.
type InstrumentedTransport struct {
	Base           stdhttp.RoundTripper
	Tracer         trace.Tracer
	Meter          metric.Meter
	waitTime       metric.Float64Histogram
	activeRequests metric.Int64UpDownCounter
}

// RoundTrip implements http.RoundTripper.
func (t *InstrumentedTransport) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	// 1. Inject propagation headers
	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))

	// 2. Check for existing span
	ctx := req.Context()
	span := trace.SpanFromContext(ctx)

	// 3. Enrich if recording
	if span.IsRecording() {
		span.SetAttributes(clientRequestAttrs(req)...)
	}

	// 4. Trace Events & Wait Time
	// Wrap the context with a ClientTrace that logs events to the span (via otelhttptrace)
	// and measures connection wait time.
	ct := otelhttptrace.NewClientTrace(ctx)

	var getConnTime time.Time
	originalGetConn := ct.GetConn
	ct.GetConn = func(hostPort string) {
		getConnTime = time.Now()
		if originalGetConn != nil {
			originalGetConn(hostPort)
		}
	}

	originalGotConn := ct.GotConn
	ct.GotConn = func(info httptrace.GotConnInfo) {
		if !getConnTime.IsZero() && t.waitTime != nil {
			t.waitTime.Record(ctx, time.Since(getConnTime).Seconds(), metric.WithAttributes(
				attribute.Bool("reused", info.Reused),
			))
		}
		if originalGotConn != nil {
			originalGotConn(info)
		}
	}
	req = req.WithContext(httptrace.WithClientTrace(ctx, ct))

	// 5. Active Requests
	if t.activeRequests != nil {
		attrs := clientRequestAttrs(req)
		t.activeRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
		defer t.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs...))
	}

	// 6. Call Base
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

	return resp, err
}

// instrumentedHandler wraps http.Handler to extract trace context and start spans.
type instrumentedHandler struct {
	base           stdhttp.Handler
	tracer         trace.Tracer
	meter          metric.Meter
	activeRequests metric.Int64UpDownCounter
}

// ServeHTTP implements http.Handler.
func (h *instrumentedHandler) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	// 1. Extract propagation headers
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// 2. Start Span (Server Kind)
	// NOTE: The handler can overwrite the span name later in the request.
	spanName := "HTTP " + r.Method
	ctx, span := h.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// 3. Add Request Attributes
	span.SetAttributes(serverRequestAttrs(r)...)

	// 4. Active Requests
	if h.activeRequests != nil {
		attrs := serverRequestAttrs(r)
		h.activeRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
		defer h.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs...))
	}

	// 5. Wrap ResponseWriter to capture status code
	rr := &responseRecorder{ResponseWriter: w, statusCode: stdhttp.StatusOK}

	// 6. Serve
	h.base.ServeHTTP(rr, r.WithContext(ctx))

	// 7. Add Response Attributes
	span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(rr.statusCode))
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
