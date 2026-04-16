# Agent Instructions

## Project Overview

hetzner-dnsapi-proxy is a Go HTTP server that proxies DNS update requests
to the Hetzner Cloud API. It supports multiple DNS API protocols (lego
HTTP request, ACMEDNS, DirectAdmin Legacy, plain HTTP).

## Building and Testing

- `make build` - Build the binary (CGO_ENABLED=0, linux/amd64)
- `make test` - Run unit tests (Ginkgo, in `pkg/`)
- `make functest` - Run functional tests (Ginkgo, in `tests/`)
- `make fmt` - Format code with gofumpt
- `make lint` - Lint with golangci-lint
- `make vendor` - Tidy and vendor dependencies

Always run `make lint`, `make test`, and `make functest` before submitting
changes. Run `make vendor` after modifying dependencies.

## Code Style

- Use plain ASCII characters only - no em dashes, smart quotes, or Unicode
  punctuation. Prefer single dashes (`-`) over em dashes.
- Code is formatted with gofumpt (`make fmt`).
- Linting config is in `.golangci.yml` - uses a comprehensive set of
  linters including gosec, govet with shadow detection, and ginkgolinter.
- Imports should group local packages (`github.com/0xfelix/hetzner-dnsapi-proxy`)
  separately (enforced by goimports).
- Dependencies are vendored in `vendor/`.

## Testing

- Tests use Ginkgo v2 and Gomega.
- Unit tests live alongside code in `pkg/`.
- Functional tests live in `tests/` with shared helpers in `tests/lib*`.

## Project Structure

- `main.go` - Entry point, config parsing, server lifecycle
- `pkg/app/` - HTTP router setup
- `pkg/config/` - Configuration (file and env parsing)
- `pkg/middleware/` - HTTP middleware (auth, domain checks, DNS update/cleanup)
- `pkg/middleware/update/` - Record creation (cloud/ backend)
- `pkg/middleware/clean/` - Record cleanup (cloud/ backend)
- `pkg/hetzner/` - Hetzner Cloud API client helpers
- `pkg/data/` - Shared data types
- `pkg/ratelimit/` - Per-client-IP rate limiting (token bucket)
- `pkg/sanitize/` - Input sanitization

## Git

- Sign off all commits with `-s` (e.g. `git commit -s -m "..."`).
- Keep commit messages concise.
