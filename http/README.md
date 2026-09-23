# http

`http` is a wrapper around the standard library `net/http` package with opinionated defaults optimized for internal
datacenter traffic.

## Why?

The standard `net/http` library defaults are designed for general internet usage, which often means no timeouts or very
long timeouts. In a high-performance internal network environment, these defaults can lead to resource exhaustion and
cascading failures when dependencies are slow or unresponsive.

This package provides:
*   **Aggressive Timeouts:** Defaults that assume a reliable, low-latency network (e.g., 2s total request timeout).
*   **Granular Control:** Options to configure specific timeouts (Connect, TLS Handshake, Response Header).
*   **TLS Key Logging:** An opt-in client option for decrypting your own HTTPS packet captures.
*   **Graceful Shutdown:** Built-in support for graceful server shutdown.
*   **Safe Defaults:** "Secure by default" configuration to prevent common pitfalls.

## Usage

### Client

Create a new client with default aggressive timeouts:

```go
package main

import (
	"log"
	"time"

	"github.com/andrewhowdencom/stdlib/http"
)

func main() {
	// Create a client with default settings (2s total timeout, etc.)
	client, err := http.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// You can also override specific defaults
	client, err = http.NewClient(
		http.WithTimeout(5 * time.Second),
		http.WithConnectTimeout(1 * time.Second),
	)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// Use standard http.Client methods
	resp, err := client.Get("http://example.com")
	if err != nil {
		log.Printf("request failed: %v", err)
		return
	}
	defer resp.Body.Close()
}
```

### TLS key logging

Pass a private file to `WithTLSKeyLogWriter` when you need to inspect your own HTTPS traffic in Wireshark. The option
uses the client's existing instrumented transport and leaves the default transport alone. See the compiled
[`ExampleWithTLSKeyLogWriter`](client_test.go) for a minimal example.

```go
keyLog, err := os.OpenFile("tls-secrets.log", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
if err != nil {
	return err
}
defer keyLog.Close()

client, err := http.NewClient(
	http.WithTimeout(0), // Allow a response stream to run longer than the default 2 seconds.
	http.WithResponseHeaderTimeout(10*time.Minute),
	http.WithTLSKeyLogWriter(keyLog),
)
if err != nil {
	return err
}
// Make HTTPS requests with client before keyLog.Close() runs.
```

Keep the writer open while requests are active. The key log and a packet capture together can reveal request and
response bodies, including prompts and authentication headers. Protect and delete both files accordingly. Passing
`nil` disables key logging. This option does not change request timeouts or buffer streamed responses.

### Server

Create and run a server with safe defaults and graceful shutdown:

```go
package main

import (
	"fmt"
	stdhttp "net/http"
	"time"

	"github.com/andrewhowdencom/stdlib/http"
)

func main() {
	handler := stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

	// Create a server with default timeouts
	srv, err := http.NewServer(":8080", handler)
	if err != nil {
		panic(err)
	}

	// Or configure with options
	srv, err = http.NewServer(":8080", handler,
		http.WithReadTimeout(5*time.Second),
		http.WithWriteTimeout(5*time.Second),
	)
	if err != nil {
		panic(err)
	}

	// Run starts the server and waits for SIGINT/SIGTERM for graceful shutdown
	if err := srv.Run(); err != nil {
		fmt.Printf("Server exited with error: %v\n", err)
	}
}
```
