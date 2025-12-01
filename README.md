# stdlib

`stdlib` is a collection of opinionated Go libraries designed for reuse across multiple projects.

## Why?

In many organizations and projects, there's a need for standard patterns and practices. This repository collects a set of
libraries that enforce these patterns, providing "safe defaults" and consistent behavior.

Key philosophies include:
*   **Safe Defaults:** Libraries should be secure and reliable by default (e.g., sensible timeouts).
*   **Opinionated:** We make choices so you don't have to, prioritizing consistency over infinite flexibility.
*   **Internal Focus:** Optimized for high-performance internal datacenter traffic.

## Packages

This repository contains the following packages:

*   **[http](./http/README.md):** A wrapper around the standard `net/http` library. It enforces aggressive timeouts
    suitable for internal datacenter communication (e.g., 2s total timeout) and provides graceful shutdown capabilities.
*   **[sysexits](./sysexits/README.md):** A library to map Go errors to standard UNIX exit codes. It allows CLI tools
    to return meaningful exit codes to the operating system while using standard Go error handling.

## Installation

You can install individual packages using `go get`.

For the `http` package:

```bash
go get github.com/andrewhowdencom/stdlib/http
```

For the `sysexits` package:

```bash
go get github.com/andrewhowdencom/stdlib/sysexits
```

## License

This project is licensed under the [GNU Affero General Public License v3.0](./LICENSE).
