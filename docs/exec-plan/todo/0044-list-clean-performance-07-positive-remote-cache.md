# List/Clean Performance 07: Conservative Positive Remote Cache
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Avoid repeated live `git ls-remote --heads` calls when recent evidence already
proves that every relevant tracked branch exists on the same remote. Cache only
positive presence for a fixed short lifetime; never persist absence or a final
`stale`/`cleanable` decision.

Completion means repeated `ww list` / `ww clean --dry-run` calls within 30
seconds can skip unchanged positive remote probes, while a missing branch,
expired entry, changed remote identity, corrupt cache, or network failure follows
the current live-query/error path. A stale positive may conservatively delay
`stale` classification for at most the TTL, but it can never make an active
branch cleanable.

Addresses: N/A - performance follow-up to the completed measurement issue
The completed measurement issue is retrievable with
`git log --all -- docs/issues/done/0037-list-clean-performance-budget.md`.

## Design Decision

- Cache only branch names observed as present from a successful complete
  `ls-remote --heads <remote>` response.
- Use a fixed 30-second TTL; do not add configuration or a new CLI flag.
- A hit is usable only when every currently requested candidate branch is in
  the unexpired positive set. If any candidate is absent from the cached set,
  run the live full-remote query and replace the positive set.
- Never cache absence. Never serve an expired entry after a live-query failure.
  Preserve the current error rather than silently using stale data.
- Key entries by repository common-dir identity, remote name, and a hash of the
  effective configured remote URL values. Never persist raw remote URLs because
  they may contain credentials.
- Reuse the `os.UserCacheDir()/ww` store from plan 0043; do not use config or
  temp directories in normal execution.

This asymmetric policy is safe for cleanup: a deleted remote branch that is
still in a positive cache remains temporarily `active`, whereas caching a false
absence could incorrectly produce `stale`. Record the positive-only/TTL/error
decision in a new ADR during execution.

## Static Analysis Findings

- `listRepo` batches branch existence into one `ListRemoteBranches` call per
  unique remote and repeats those live calls on every CLI invocation.
- In a workspace with multiple real remotes, network/credential latency remains
  additive before plan 0041 and consumes concurrent slots after it.
- Plan 0039 removes per-branch local config subprocesses but intentionally keeps
  live remote enumeration unchanged.
- PR #280 uses local bare remotes, so it detects cache overhead/regression but
  understates network benefit. A controlled delayed-remote test and warm-cache
  benchmark are required alongside the existing budget.

## Existing Implementation References

- `worktree/worktree.go`, lines 456-540: status aggregation, unique-remote map,
  and `ListRemoteBranches` call site shared by list/clean.
- `git/git.go`
  - `BranchRemote`, lines 258-266: repository tracking identity input.
  - `HasRemote` / `RemoteBranchExists` / `ListRemoteBranches`, lines 278-326:
    existing remote URL/existence and complete branch-set queries.
  - `GitCommonDir`, lines 546-552: repository identity input.
- `docs/design-decisions/adr/0005-worktree-status-precedence.md`, lines 1-23:
  conservative merged/stale/active cleanup policy.
- `docs/specs/cli-commands.md`, lines 377-441 and 503 onward: stale and
  cleanable behavior that must remain safe.
- `docs/specs/git-operations.md`, lines 138-153: current live remote presence
  contract.
- `docs/specs/cache.md` (created by plan 0043): persistent-store location,
  trust, and fail-open rules.
- `cmd/ww/performance_test.go`, lines 1-77, and
  `internal/testutil/performance_fixture.go`, lines 53-165: PR #280's local
  remote fixture and command boundary.

## Code Change Map

- `docs/design-decisions/README.md` (MODIFY)
  - Index the conservative positive-remote cache decision.
- `docs/design-decisions/adr/0018-positive-remote-cache.md` (NEW)
  - Record positive-only evidence, 30-second TTL, full live refresh on any
    missing candidate, no expired fallback, and credential-safe identity keys.
- `docs/specs/cache.md` (MODIFY)
  - Define remote cache key/value/TTL, positive-only safety, and invalidation.
- `docs/specs/git-operations.md` (MODIFY)
  - State when a recent positive set may replace a live `ls-remote`, and that
    absence/expiry still requires the existing live query.
- `docs/specs/testing.md` (MODIFY)
  - Add warm positive-cache coverage and a deterministic delayed-remote test;
    keep existing budgeted benchmarks cold.
- `internal/cache/remote.go` (NEW)
  - Store observed-at timestamps and positive branch sets using plan 0043's
    versioned/atomic store.
  - Hash common-dir identity and configured remote URL values; store no URL,
    credential, token, command output, absence, or status decision.
  - Accept a clock seam for deterministic TTL tests.
- `internal/cache/remote_test.go` (NEW)
  - Cover full hit, one requested branch absent, exact TTL boundary, expiry,
    remote URL/name/common-dir changes, clock behavior, corrupt/tampered data,
    credential non-persistence, and concurrent writers.
- `git/git.go` (MODIFY)
  - Add a narrowly scoped helper to read all effective URL values needed for a
    credential-safe identity hash without exposing them in errors/cache files.
  - Keep the uncached `ListRemoteBranches` behavior available.
- `git/git_test.go` (MODIFY)
  - Cover multiple remote URLs, URL changes, unusual branch names, and live
    query failure behavior.
- `worktree/worktree.go` (MODIFY)
  - Inject an optional positive-remote cache into repository status evaluation.
  - Skip live enumeration only when every candidate branch has fresh positive
    evidence; otherwise query live, classify from that response, and replace
    the positive set.
  - Preserve output order, merged precedence, base-unknown degradation, and
    errors.
- `worktree/worktree_test.go` (MODIFY)
  - Prove cached positives stay active, deleted branches are at worst delayed,
    cache misses/expiry detect stale live, and no cache state can create a false
    merged/stale/cleanable result.
- `cmd/ww/main.go` (MODIFY)
  - Pass plan 0043's best-effort cache store into list/status managers without
    changing commands that do not need remote status.
- `internal/testutil/performance_fixture.go` (MODIFY)
  - Add an opt-in deterministic remote-delay wrapper/profile for investigation;
    retain local bare remotes and default counts for budgeted PR #280 runs.
- `cmd/ww/performance_test.go` (MODIFY)
  - Keep cold budgeted benchmarks comparable and add/extend warm variants that
    prime positive remote evidence outside the timer.
- `integration_test.go` (MODIFY)
  - Execute separate CLI processes proving hit/miss/expiry, branch deletion
    delay followed by live stale detection, URL changes, remote errors, and
    read-only/unavailable cache fallback.

## Spec Changes

- `docs/specs/cache.md`: add the 30-second positive-only remote cache contract.
- `docs/specs/git-operations.md`: allow a fresh complete positive set to avoid
  `ls-remote`; continue requiring live evidence for absence and after expiry.
- `docs/specs/testing.md`: separate cold budget protection from warm/delayed-
  remote benefit measurement.
- `docs/specs/cli-commands.md`: clarify, if necessary, that remote deletion may
  remain `active` for at most the positive-cache TTL. It must never become
  prematurely cleanable.

## Sub-tasks

1. Specify conservative semantics
   - Add ADR/spec changes first, including TTL boundary, credentials, error
     behavior, and the bounded active-delay statement.
2. Add positive cache and status integration
   - Implement identity hashing, clocked entries, all-candidates hit logic, and
     live refresh without caching absence.
   - Preserve uncached behavior when the store is missing or unusable.
3. Prove safety and benefit
   - Test deletion, URL changes, expiry, network errors, tampering, and
     concurrent processes.
   - Compare cold, warm, and controlled-delayed-remote measurements with the
     same PR #280 fixture scale.
   - Require a warm/delayed improvement outside variation and no meaningful
     cold regression before landing.

## Dependencies and Parallelism

- Depends on plan 0039 for batched candidate/tracking metadata and plan 0043
  for persistent storage, schema, atomicity, permissions, and test isolation.
- Should be evaluated after plan 0041 because repository concurrency changes
  the wall-clock impact of remote latency.
- ADR/spec work and cache unit tests may proceed in parallel. Status integration
  waits for the cache key/value interface.

## Verification

- `go test ./internal/cache ./git ./worktree ./cmd/ww`
- `go test -race ./internal/cache ./worktree ./cmd/ww`
- Targeted integration tests for fresh hit, one-branch miss, exact expiry,
  deletion delay, URL change, remote failure, and unavailable cache.
- `go test ./...`
- `make test`
- `make test-all`
- `make lint`
- `make perf-check`
- Run cold, warm, and delayed-remote profiles with identical PR #280
  repo/worktree counts and record `ls-remote` call counts plus timings.
- Confirm the GitHub-hosted cold job remains within 400 ms / 420 ms. Judge the
  cache on same-host warm/delayed evidence rather than those cold budgets.
