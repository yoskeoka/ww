# 008: Docker Integration Test Infrastructure

**Objective:** Set up a Docker-based integration test environment for Phase 2 workspace tests. Workspace structures (parent + multiple child repos + worktrees) need isolation from host git config and filesystem layout.

**Covers:** Test strategy from Phase 2 design doc.

## Context

Phase 1 integration tests (`integration_test.go`) build the `ww` binary and run it against `t.TempDir()` git repos. This works for single-repo scenarios but Phase 2 workspace tests need:
- Isolation from host `~/.gitconfig` (which may affect branch defaults, merge behavior)
- Reproducible filesystem layout (workspace root with multiple child git repos)
- Clean git state (no ambient credentials or hooks)

A single Dockerfile provides this isolation. No docker-compose needed — there are no service dependencies.

## Sub-tasks

- [ ] [parallel] **Create `Dockerfile.test`**: Multi-stage build:
  - Build stage: Go toolchain, compile `ww` binary and test binary
  - Test stage: Minimal image with `git` + compiled test binary
  - Set `GIT_CONFIG_GLOBAL=/dev/null` and configure minimal git identity
- [ ] [parallel] **Create workspace test helpers**: `testutil/workspace.go` (or `_test.go` helper) providing:
  - `SetupWorkspace(t, opts)` — creates a temp dir with N child git repos, each with configurable branches/commits
  - `SetupNonGitWorkspace(t, opts)` — creates a non-git parent with git children
  - Returns struct with paths for assertions
- [ ] [depends on: Dockerfile, test helpers] **Add `make test-docker` target**: Makefile target that builds and runs the Docker test image
- [ ] [depends on: make test-docker] **Update CI workflow (`.github/workflows/ci.yml`)**: Add a step that runs `make test-docker` (or runs Docker tests alongside existing tests)

## Code Changes

| File | Change |
|------|--------|
| `Dockerfile.test` | New — Docker image for integration tests |
| `testutil/workspace.go` | New — workspace structure test helpers |
| `Makefile` | Add `test-docker` target |
| `.github/workflows/ci.yml` | Add Docker test step |

## Spec Changes

None — this is test infrastructure only.

## Design Notes

- Unit tests (`workspace/`, `git/`, `worktree/`) continue using `t.TempDir()` directly — Docker is for integration tests only.
- Test helpers are designed to be reused by plans 009, 010, 011.
- `Dockerfile.test` uses multi-stage build to keep the test image small.

## Verification

- `make test-docker` runs successfully
- CI workflow passes with Docker test step
- Test helpers can create workspace structures (verified by a smoke test)
