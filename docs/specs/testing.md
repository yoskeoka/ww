# Testing Specification

## Targets

| Command | Behavior |
|---|---|
| `make test` | Runs `go test -short ./...` and skips Docker-backed integration tests |
| `make test-all` | Runs `go test ./...` and includes Docker-backed integration tests |

## Docker Integration Harness

Integration tests execute `ww` and supporting shell commands inside a shared Docker container.

The test helper must return combined command output for both success and failure cases. It must not depend on only one Docker exec stream encoding:

- If the container exec reader is already plain combined output, return that output as-is.
- If the container exec reader is Docker-multiplexed, demultiplex it before returning the combined output.

This compatibility requirement exists so `make test-all` remains stable across supported Docker and `testcontainers-go` combinations.
