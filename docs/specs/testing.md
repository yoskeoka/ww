# Testing Specification

## Targets

| Command | Behavior |
|---|---|
| `make test` | Runs `go test -short ./...` and skips Docker-backed integration tests |
| `make test-all` | Runs `go test ./...` and includes Docker-backed integration tests |

## Docker Integration Harness

Integration tests execute `ww` and supporting shell commands inside a shared Docker container. Each test gets its own cloned repository from a shared seeded fixture so the suite can run in parallel without rebuilding the seed data.

The test helper must return combined command output for both success and failure cases. It must not depend on only one Docker exec stream encoding:

- If the container exec reader is already plain combined output, return that output as-is.
- If the container exec reader is Docker-multiplexed, demultiplex it before returning the combined output.

This compatibility requirement exists so `make test-all` remains stable across supported Docker and `testcontainers-go` combinations.

## Parallel vs Serial Test Classification

Integration tests are split into two categories to prevent Docker exec concurrency flakiness:

### Parallel-safe tests

Tests that are read-only, create-only, or assert only on error cases may call `t.Parallel()`. These tests create isolated fixtures (via `MkdirTemp`) and do not perform destructive operations (remove/clean) or assert on branch deletion state.

### Stateful (serial) tests

Tests that mutate worktree or branch state and then assert on cleanup side-effects must NOT call `t.Parallel()`. This includes:

- Tests that call `ww clean` or `ww remove` and assert on filesystem removal
- Tests that push and delete remote branches and assert on status (merged/stale/active)
- Tests that run `ww` from within a worktree directory (cross-worktree path resolution)

These tests run sequentially to avoid Docker exec contention when many complex multi-step operations execute simultaneously in the shared container.

### Rationale

Each test gets its own isolated temp directory inside the container, so tests do not share filesystem state. However, all tests share a single Docker container, and high-concurrency Docker exec calls (20+ simultaneous) cause intermittent failures: partial output, timing-sensitive assertions failing, or exec call contention. Serializing the most exec-heavy tests (clean/remove with 10+ exec calls each) keeps the suite reliable while preserving parallelism for simpler tests.
