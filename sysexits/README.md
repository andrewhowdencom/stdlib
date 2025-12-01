# sysexits

`sysexits` provides a way to map Go errors to standard UNIX exit codes.

## Why?

There is a convention on UNIX systems about what specific application codes mean. Originally from UNIX, these conventions are [known as "sysexits.h"](https://manpages.ubuntu.com/manpages/lunar/man3/sysexits.h.3head.html).

Go, by convention, [passes errors around to indicate a failure](https://go.dev/blog/go1.13-errors). It uses an error type to indicate a class of errors and the error value to indicate a specific error condition within an error type.

This package bridges these two worlds. It allows you to wrap Go errors with specific exit codes, so your CLI tools can return meaningful exit codes to the operating system, while still using standard Go error handling patterns.

## Usage

Here is a complete example of how to use `sysexits` in a CLI application:

```go
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/andrewhowdencom/stdlib/sysexits"
)

func main() {
	if err := run(); err != nil {
		// Check if the error is a Sysexit
		var exit sysexits.Sysexit
		if errors.As(err, &exit) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(exit.Code)
		}

		// Fallback for generic errors
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Simulate a "user not found" scenario
	if err := findUser("unknown_user"); err != nil {
		// Wrap the error with sysexits.NoUser (exit code 67)
		// We use %w to wrap the original error and also include the sysexit's message
		return fmt.Errorf("%w: %v", sysexits.NoUser, err)
	}
	return nil
}

func findUser(username string) error {
	return fmt.Errorf("user '%s' does not exist", username)
}
```
