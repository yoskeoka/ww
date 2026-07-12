# Integration tests: testcontainers-go with testing.Short() split

## Status

Superseded on 2026-04-20 by [0012](0012-host-native-integration-harness.md).

## Context

Phase 2 workspace tests need isolation from host git config and filesystem. Two approaches were considered:

- **Standalone Docker runner**: Dockerfile.test that builds and runs tests inside a container, invoked via `make test-docker`. Requires a separate test execution model — assertion libraries (bats, shell scripts) or a compiled test binary copied into the image.
- **testcontainers-go inside `go test`**: Integration tests use testcontainers-go to spin up containers from within `go test`. Tests stay in Go's standard framework with `t.Error`/`t.Fatal` assertions.

The standalone approach was rejected because `ww` is a single-binary CLI with no service dependencies — maintaining a separate test suite and assertion framework adds complexity without proportional benefit.

## Decision

- Use **testcontainers-go** to run integration tests inside Docker containers from `go test`.
- Split tests with `testing.Short()`: `make test` runs `go test -short` (unit only, no Docker), `make test-integration` runs integration tests (Docker required).
- CI runs both as separate jobs.

## Consequences

- **Positive**: Single test framework (`go test`), no shell-based assertion tooling, developers get fast `make test` without Docker, CI has clear separation.
- **Negative**: testcontainers-go adds a dependency. Acceptable — it's test-only and well-maintained.
