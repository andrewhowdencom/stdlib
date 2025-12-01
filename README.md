# stdlib
A set of standard libraries in Go that I like reusing.

## Development

### Requirements

*   [Go](https://go.dev/) (see `go.mod` for version)
*   [Task](https://taskfile.dev/)
*   [golangci-lint](https://golangci-lint.run/)

### Common Tasks

This project uses `Taskfile` to manage development tasks.

*   `task setup`: Install dependencies.
*   `task generate`: Generate code.
*   `task build`: Build the code.
*   `task test`: Run tests.
*   `task lint`: Run linters.
*   `task validate`: Run lint, test, and build (recommended before commit).
