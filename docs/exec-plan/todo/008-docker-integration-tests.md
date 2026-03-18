# 008: Docker Integration Test Infrastructure

**Objective:** Evolve the existing integration tests to run inside Docker containers via testcontainers-go, isolating from host git config and filesystem. Introduce `testing.Short()` split so unit tests run fast without Docker.

**Covers:** Test strategy from Phase 2 design doc.

## Context

Phase 1 integration tests (`integration_test.go`) build the `ww` binary and run it against `t.TempDir()` git repos on the host. This works for single-repo scenarios but Phase 2 workspace tests need:
- Isolation from host `~/.gitconfig` (which may affect branch defaults, merge behavior)
- Reproducible filesystem layout (workspace root with multiple child git repos)
- Clean git state (no ambient credentials or hooks)

### Approach: testcontainers-go inside `go test`

Rather than maintaining a separate test suite or Dockerfile-based runner, integration tests use **testcontainers-go** to spin up a Docker container from within `go test`. This keeps everything in Go's standard test framework — same assertions, same test discovery, same CI tooling.

### Test split: short vs long-running

| Target | Flag | Docker required | What runs |
|--------|------|-----------------|-----------|
| `make test` | `go test -short ./...` | No | Unit tests only |
| `make test-integration` | `go test -run Integration ./...` | Yes | Integration tests (Docker) |

Integration test functions are guarded by `testing.Short()`:
```go
func TestIntegration_...(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // ...
}
```

CI runs both as separate jobs/steps.

## Sub-tasks

- [ ] [parallel] **Add testcontainers-go dependency**: `go get github.com/testcontainers/testcontainers-go`. Create a test helper that starts a container with `git` + `ww` binary mounted/copied in, with `GIT_CONFIG_GLOBAL=/dev/null` and minimal git identity configured.
- [ ] [parallel] **Create workspace test helpers**: `testutil/workspace.go` providing:
  - `SetupWorkspace(t, opts)` — creates a temp dir with N child git repos, each with configurable branches/commits
  - `SetupNonGitWorkspace(t, opts)` — creates a non-git parent with git children
  - Returns struct with paths for assertions
  - These helpers run inside the container (called from integration test code)
- [ ] [parallel] **Refactor existing `integration_test.go`**: Add `testing.Short()` skip guard to all existing integration test functions. Verify they still pass as-is (before Docker migration).
- [ ] [depends on: testcontainers-go, workspace helpers] **Migrate existing integration tests to Docker**: Update tests to run `ww` commands inside the container instead of on the host. Existing single-repo tests should produce identical results.
- [ ] [depends on: Refactor] **Update Makefile**:
  - `make test` → `go test -short ./...` (was `go test ./...`)
  - `make test-integration` → `go test -run Integration ./...` (new target)
  - Keep a `make test-all` or similar that runs both
- [ ] [depends on: Update Makefile] **Update CI workflow (`.github/workflows/ci.yml`)**: Split into two jobs or steps:
  - Unit tests: `make test` (fast, no Docker)
  - Integration tests: `make test-integration` (requires Docker)

## Code Changes

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | Add testcontainers-go dependency |
| `testutil/container.go` | New — testcontainers-go helper (start container, copy binary, exec commands) |
| `testutil/workspace.go` | New — workspace structure test helpers |
| `integration_test.go` | Add `testing.Short()` guards, migrate to run inside Docker containers |
| `Makefile` | Change `test` to `-short`, add `test-integration` target |
| `.github/workflows/ci.yml` | Split unit/integration test jobs |

## Spec Changes

None — this is test infrastructure only.

## Design Notes

- **Why testcontainers-go over standalone Dockerfile**: Keeps test execution in `go test`. No separate test framework, no shell-based assertion libraries. Same `t.Error`/`t.Fatal` patterns.
- **Why `testing.Short()` split**: Developers get fast feedback with `make test` (no Docker needed). CI runs both. Integration tests are opt-in locally.
- Test helpers (`testutil/`) are designed to be reused by plans 009, 010, 011 for workspace-mode test scenarios.

## Verification

- `make test` passes without Docker running (unit tests only)
- `make test-integration` passes with Docker running (integration tests)
- Existing Phase 1 integration tests produce identical results inside Docker
- CI workflow has separate unit/integration steps, both green
