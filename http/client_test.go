package http

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Timeout != 2*time.Second {
		t.Errorf("expected default timeout 2s, got %v", c.Timeout)
	}

	tr, err := getTransport(c)
	if err != nil {
		t.Fatalf("expected transport to be *http.Transport: %v", err)
	}

	if tr.TLSHandshakeTimeout != 500*time.Millisecond {
		t.Errorf("expected TLSHandshakeTimeout 500ms, got %v", tr.TLSHandshakeTimeout)
	}

	if tr.ResponseHeaderTimeout != 1500*time.Millisecond {
		t.Errorf("expected ResponseHeaderTimeout 1.5s, got %v", tr.ResponseHeaderTimeout)
	}
}

func TestWithTLSKeyLogWriter(t *testing.T) {
	continueResponse := make(chan struct{})
	server := httptest.NewTLSServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		_, _ = io.WriteString(w, "first\n")
		w.(stdhttp.Flusher).Flush()
		<-continueResponse
		_, _ = io.WriteString(w, "second\n")
	}))
	defer server.Close()

	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	sharedTLSConfig := &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	defaultTLSConfig := stdhttp.DefaultTransport.(*stdhttp.Transport).TLSClientConfig
	var secrets bytes.Buffer
	client, err := NewClient(
		WithTimeout(0),
		func(c *stdhttp.Client) error {
			transport, err := getTransport(c)
			if err != nil {
				return err
			}
			transport.TLSClientConfig = sharedTLSConfig
			return nil
		},
		WithTLSKeyLogWriter(&secrets),
		WithClientTracerProvider(tracenoop.NewTracerProvider()),
		WithClientMeterProvider(metricnoop.NewMeterProvider()),
	)
	if err != nil {
		t.Fatal(err)
	}
	transport, err := getTransport(client)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := client.Transport.(*InstrumentedTransport); !ok {
		t.Fatal("client transport lost instrumentation wrapper")
	}
	if transport.TLSClientConfig == sharedTLSConfig {
		t.Fatal("client reused shared TLS config")
	}
	if transport.TLSClientConfig.RootCAs != roots || transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("client lost existing TLS settings")
	}
	if sharedTLSConfig.KeyLogWriter != nil {
		t.Fatal("client mutated shared TLS config")
	}
	if stdhttp.DefaultTransport.(*stdhttp.Transport).TLSClientConfig != defaultTLSConfig {
		t.Fatal("client mutated default transport TLS config")
	}

	response, err := client.Get(server.URL)
	if err != nil {
		close(continueResponse)
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	reader := bufio.NewReader(response.Body)
	first, err := reader.ReadString('\n')
	if err != nil || first != "first\n" {
		close(continueResponse)
		t.Fatalf("first streamed chunk = %q, %v", first, err)
	}
	close(continueResponse)
	rest, err := io.ReadAll(reader)
	if err != nil || string(rest) != "second\n" {
		t.Fatalf("remaining streamed body = %q, %v", rest, err)
	}
	keyLogLine := regexp.MustCompile(`(?m)^(CLIENT_RANDOM|CLIENT_HANDSHAKE_TRAFFIC_SECRET|CLIENT_TRAFFIC_SECRET_0) [0-9a-fA-F]+ [0-9a-fA-F]+$`)
	if !keyLogLine.Match(secrets.Bytes()) {
		t.Fatalf("missing NSS key log entry: %q", secrets.String())
	}

	sharedKeyLog := &bytes.Buffer{}
	loggedTLSConfig := &tls.Config{KeyLogWriter: sharedKeyLog}
	otherClient, err := NewClient(
		func(c *stdhttp.Client) error {
			transport, err := getTransport(c)
			if err != nil {
				return err
			}
			transport.TLSClientConfig = loggedTLSConfig
			return nil
		},
		WithTLSKeyLogWriter(nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	otherTransport, err := getTransport(otherClient)
	if err != nil {
		t.Fatal(err)
	}
	if otherTransport.TLSClientConfig.KeyLogWriter != nil {
		t.Fatal("nil writer enabled key logging")
	}
	if loggedTLSConfig.KeyLogWriter != sharedKeyLog {
		t.Fatal("nil writer mutated shared TLS config")
	}
}

func ExampleWithTLSKeyLogWriter() {
	keyLog, err := os.CreateTemp("", "tls-secrets-*.log") // Mode 0600.
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.Remove(keyLog.Name()) }()
	defer func() { _ = keyLog.Close() }()

	client, err := NewClient(WithTLSKeyLogWriter(keyLog))
	if err != nil {
		panic(err)
	}
	// Use client for HTTPS requests while keyLog remains open.
	fmt.Println(client.Timeout)
	// Output: 2s
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

	tr, err := getTransport(c)
	if err != nil {
		t.Fatalf("expected transport to be *http.Transport: %v", err)
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

	tr, err := getTransport(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tr.MaxIdleConns != 100 {
		t.Errorf("expected default MaxIdleConns 100, got %d", tr.MaxIdleConns)
	}

	c2, err := NewClient(WithMaxIdleConns(50))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tr2, err := getTransport(c2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tr2.MaxIdleConns != 50 {
		t.Errorf("expected MaxIdleConns 50, got %d", tr2.MaxIdleConns)
	}
}
