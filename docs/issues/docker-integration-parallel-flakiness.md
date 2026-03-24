# Docker integration tests are flaky under parallel execution

**Type:** reliability | **Priority:** Medium

## Problem

The Docker-backed integration suite has shown intermittent failures that do not reproduce reliably in isolation.

**Current evidence:**
- `make test-all` has failed with integration-test-only regressions that passed when the affected tests were re-run individually.
- The suite uses a shared Docker container for all integration tests and many tests call `t.Parallel()`.
- The failing cases so far have been stateful tests that create, remove, or clean git worktrees and then assert on filesystem state and branch cleanup.

**Known unknowns:**
- The exact root cause has not been isolated yet.
- It is not yet proven whether the interference is from shared container state, git timing, helper behavior, or test assumptions that are too strict under parallel execution.

**Files:**
- `integration_test.go` - many Docker integration tests run in parallel and assert on mutable repo/worktree state
- `internal/testutil/container.go` - all integration tests share one running container via `globalEnv`
- `docs/specs/testing.md` - documents the shared-container harness

## Proposed Approaches

### Option 1: Serialize only stateful integration tests

Keep the shared container for speed, but remove `t.Parallel()` from tests that:
- mutate worktree or branch state
- assert cleanup side effects
- operate on workspace-wide paths like `.worktrees/`

This is the lowest-risk option and should preserve most of the current runtime benefit.

### Option 2: Introduce a "parallel-safe" vs "stateful" test split

Define two categories inside `integration_test.go`:
- parallel-safe tests: read-mostly or single-repo isolated cases
- stateful tests: serialized cases that mutate shared or timing-sensitive state

This keeps the fast path parallel while making isolation expectations explicit in the test suite structure.

### Option 3: Improve per-test isolation inside the shared container

If selective serialization is insufficient, keep one container but strengthen isolation:
- guarantee unique per-test directories and namespacing for all fixture paths
- reduce shared mutable helpers or global fixture reuse where timing can leak across tests
- batch assertions to avoid repeated visibility checks across Docker exec boundaries

This aims to improve reliability without paying the full cost of one container per test.

### Option 4: Use per-test containers only as a last resort

If the flake source is proven to depend on container-global state, move the problematic subset to per-test containers.

This should be treated as a last resort because it will likely increase `make test-all` runtime materially.

## Recommended Next Step

Start with Option 1 and Option 2:
- identify which integration tests are stateful
- serialize only those tests
- keep the rest parallel
- re-run the suite repeatedly to confirm the flake rate drops

If flakes remain, instrument the failing paths and then evaluate Option 3 before considering per-test containers.

## Success Criteria

- `make test-all` is stable across repeated runs
- integration runtime stays close to the current shared-container baseline
- the suite documents which tests are intentionally serialized and why
