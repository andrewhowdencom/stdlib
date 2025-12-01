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
