# List/Clean Performance 06: Persistent Discovery Cache
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Reuse validated start-path, current-worktree, main-worktree, workspace-root, and
workspace-repository discovery results across separate `ww` invocations. Keep
the cache strictly optional: a missing, unreadable, unwritable, corrupt, stale,
or unknown-version cache must fall back to the normal discovery path without
changing command output, errors, sandbox boundaries, or repository membership.

Completion means a repeated `ww list` or `ww clean` from the same worktree can
avoid the Git subprocesses used to rediscover its main worktree and validate
unchanged workspace children, while a cold/mismatched entry produces exactly
the uncached result. PR #280's budgeted command benchmarks remain cold-cache
regression gates, and new warm-cache measurements demonstrate whether
cross-command persistence materially improves the reported slow environment.

Addresses: N/A - performance follow-up to the completed measurement issue
`docs/issues/done/0037-list-clean-performance-budget.md`.

## Design Decision

Use `filepath.Join(os.UserCacheDir(), "ww")` as the cache root:

- Unix: `$XDG_CACHE_HOME/ww` when set, otherwise `$HOME/.cache/ww`.
- macOS: `$HOME/Library/Caches/ww`.
- Other supported behavior follows Go's `os.UserCacheDir` contract.

Do not store cache data under `$XDG_CONFIG_HOME`, because config may be made
read-only for sandbox safety and cache data is non-essential rather than user
configuration. Do not fall back to `os.TempDir()` in normal execution. A temp
directory is neither guaranteed to exist nor be accessible, may be periodically
or reboot-cleaned, and a reusable shared-temp path adds ownership/symlink
validation that the user-cache location avoids. If `os.UserCacheDir` cannot be
resolved or used, run without caching. Tests should point the user-cache
environment/home at a test-owned temp directory.

The persistent cache stores topology hints only. It must not store final
`WorktreeInfo`, branch/head values, merged/stale/active status, patch-equivalence
results, or cleanable decisions. Normal commits, checkout, fetch, remote
changes, and raw Git commands can change those values outside `ww` even when
worktree creation/removal is normally performed through `ww`.

This location, fail-open policy, trust boundary, and absence of a temp fallback
are architectural choices. Record them in a new ADR during execution.

## Static Analysis Findings

- `newManagerWithOptions` and `loadManagerContext` repeatedly enter
  `MainWorktreeDir` / `DetectWithOptions` for every short-lived CLI process.
- `MainWorktreeDir` shells out to `git rev-parse --git-common-dir`; discovery
  then scans bounded candidates and validates child repositories with
  top-level, git-dir, and common-dir queries.
- Plan 0038 removes duplicate work inside one invocation, but explicitly does
  not reuse results across commands.
- A cached current-worktree root can match later descendant start directories
  by canonical containment and map them to the main root. Cached workspace
  candidates can be validated with cheap sorted directory/`.git` marker
  fingerprints before trusting their normalized repository set.
- Caching `git worktree list --porcelain` output would also cache displayed
  branch/head state. That can become stale after ordinary Git operations and is
  excluded from this plan.

## Existing Implementation References

- `cmd/ww/main.go`
  - `newManagerWithOptions`, lines 142-195: current pre-config resolution and
    optional sandbox rerun.
  - `loadManagerContext`, lines 197-224: workspace detection, main-root
    resolution, and config loading.
- `workspace/workspace.go`
  - `DetectOptions` / `DetectWithOptions`, lines 39-129: discovery API and
    sandbox-sensitive result construction.
  - `detectContainingWorkspace` through `scanImmediateRepos`, lines 131-236:
    bounded candidates and immediate-child enumeration.
  - `isStandaloneRepoRoot`, lines 288-333: authoritative Git validation.
  - `normalizeRepos`, lines 371-400: deterministic cached repository ordering.
- `git/git.go`
  - `MainWorktreeDir` / `TopLevelDir` / `GitCommonDir`, lines 528-552: current
    Git-backed path identity queries.
  - `WorktreeList`, lines 114-121: dynamic worktree/branch/head source that
    remains uncached.
- `docs/specs/workspace-discovery.md`, lines 30-90: black-box bounded and
  sandbox discovery behavior.
- `docs/design-decisions/adr/0014-global-config-path-and-overrides.md`, lines
  12-29: existing separation of explicit user config and sandbox behavior.
- `cmd/ww/performance_test.go`, lines 1-77, and
  `tools/performance-budget.json`, lines 1-28: PR #280's benchmark names,
  fixture, and advisory limits.

## Code Change Map

- `docs/design-decisions/README.md` (MODIFY)
  - Index the persistent-cache location/trust decision.
- `docs/design-decisions/adr/0017-persistent-cache-location-and-trust.md` (NEW)
  - Record `os.UserCacheDir()/ww`, no normal tmp fallback, fail-open behavior,
    topology-only data, validation before use, and sandbox non-expansion.
- `docs/specs/README.md` (MODIFY)
  - Link the cache contract.
- `docs/specs/cache.md` (NEW)
  - Define location, permissions, schema versioning, atomic writes, cache-hit
    validation, silent miss/failure behavior, sandbox boundaries, and data that
    must never be cached.
- `docs/specs/testing.md` (MODIFY)
  - Require test-owned user-cache isolation.
  - Keep existing budgeted benchmarks cold-cache and add non-budgeted warm-cache
    variants using the same PR #280 fixture and command boundary.
- `internal/cache/store.go` (NEW)
  - Resolve `os.UserCacheDir()/ww`, use application directory/file permissions
    no broader than `0700`/`0600`, hash path-derived filenames, validate schema,
    and replace files atomically with last-writer-wins behavior.
  - Treat all read/write/parse failures as misses; cache failures must not write
    warnings into normal stdout/stderr.
- `internal/cache/topology.go` (NEW)
  - Model canonical start/worktree/main/workspace paths, sandbox mode,
    deterministic repo lists, and validation fingerprints.
  - Reject entries whose paths disappear, change identity, escape the current
    sandbox/bounded candidate set, or no longer match sorted immediate-child and
    `.git` marker fingerprints.
- `internal/cache/cache_test.go` (NEW)
  - Cover path resolution, permissions, atomic replacement, concurrent writers,
    corrupt/unknown schemas, tampered paths, symlink/path escape attempts,
    fingerprint changes, and fail-open operation.
- `workspace/workspace.go` (MODIFY)
  - Add an optional discovery-cache seam to `DetectOptions` without changing
    uncached callers.
  - On a validated hit, return the same normalized `Workspace`; on any miss,
    execute current discovery and publish a new hint.
- `workspace/workspace_test.go` (MODIFY)
  - Prove hit/miss parity for single repo, linked worktree, git/non-git workspace
    roots, parent/grandparent selection, repo add/remove/move, and sandbox mode.
- `cmd/ww/main.go` (MODIFY)
  - Create the best-effort cache store once per invocation and pass it through
    initial and sandbox-rerun discovery.
  - Ensure a cached result cannot suppress config loading or change
    config-selected sandbox behavior.
- `internal/testutil/host.go` (MODIFY)
  - Isolate user cache under test-owned temp state and expose cache reset/prime
    helpers without writing to the developer's real cache.
- `cmd/ww/performance_test.go` (MODIFY)
  - Reset cache outside the timer for existing cold budgeted benchmarks.
  - Add stable warm variants that prime once outside the timer and assert the
    same entries/cleanable counts.
- `integration_test.go` (MODIFY)
  - Exercise separate CLI processes for cold/warm hits, invalidation/fallback,
    concurrent cache writers, read-only cache roots, and sandbox restrictions.

## Spec Changes

- Add `docs/specs/cache.md` with the observable cache contract above.
- Extend `docs/specs/testing.md` so tests never touch the operator's cache and
  PR #280's existing benchmark/budget remains a cold-cache comparison while a
  warm profile measures the new capability.
- No `ww list`, `ww clean`, workspace membership, or status output changes.

## Sub-tasks

1. Specify and isolate cache storage
   - Add ADR/spec first, implement `os.UserCacheDir()/ww`, atomic files, schema,
     permissions, and fail-open behavior.
   - Redirect unit/integration/performance cache state to test-owned temp roots.
2. Add validated topology reuse
   - Define canonical identity and immediate-child fingerprints.
   - Integrate optional cache lookup/publication without broadening sandbox
     discovery or caching dynamic Git/status data.
3. Prove safety and performance
   - Cover tamper, stale, move, deletion, concurrent process, read-only, and
     unknown-schema cases.
   - Run cold and warm PR #280 profiles on the same host and retain all samples.
   - Require warm improvement outside run-to-run variation and no meaningful
     cold regression before landing.

## Dependencies and Parallelism

- Depends on plan 0038 so persistent reuse builds on one canonical invocation-
  local discovery path rather than preserving duplicate work.
- May be implemented after 0039-0042; it must not be hidden behind plan 0041's
  repository concurrency when measuring the discovery-specific warm path.
- Plan 0044 depends on the cache storage/schema/test-isolation foundation here.
- Store/ADR/spec work and workspace hit/miss tests may proceed in parallel after
  the schema and trust boundary are fixed.

## Verification

- `go test ./internal/cache ./workspace ./cmd/ww`
- `go test -race ./internal/cache ./workspace ./cmd/ww`
- `go test ./...`
- `make test`
- `make test-all`
- `make lint`
- `make perf-check`
- Run documented cold and warm variants with identical PR #280 repo/worktree
  counts and record samples, cache-hit proof, and cache filesystem location.
- Confirm the implementation PR's GitHub-hosted cold performance job remains
  within 400 ms list / 420 ms clean. Judge warm improvement using same-host
  before/after measurements, not those runner-specific limits.
