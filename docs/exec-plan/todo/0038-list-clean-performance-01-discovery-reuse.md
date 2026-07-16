# List/Clean Performance 01: Reuse Workspace Discovery Results
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Reduce command-startup work shared by `ww list` and `ww clean` by reusing
workspace discovery results within one invocation. Preserve the accepted
nearest-containing-workspace algorithm, sandbox boundary, repository ordering,
no-follow symlink rule, linked-worktree exclusion, and error behavior.

Completion means repeated scans of the same candidate directory and redundant
main-worktree resolution are removed, the detected `Workspace` is unchanged for
all covered layouts, and the command-boundary benchmarks introduced by
[PR #280](https://github.com/yoskeoka/ww/pull/280) show before/after medians for
both commands on the same host and fixture.

Addresses: N/A - performance follow-up to the completed measurement issue
`docs/issues/done/0037-list-clean-performance-budget.md`.

## Static Analysis Findings

- `DetectWithOptions` scans `absStart` before it knows whether the start path is
  in Git, then `detectContainingWorkspace` can scan that same directory again.
- `isContainingWorkspaceRoot` scans each qualifying candidate to decide whether
  it contains enough repositories, and `reposAtWorkspaceRoot` scans the winning
  root again to construct the returned repository list.
- `scanImmediateRepos` sends every immediate child directory through
  `isStandaloneRepoRoot`; a surviving candidate can execute top-level,
  git-dir, and common-dir Git queries even when a cheap `.git` marker check
  could reject it first. Presence of a marker must remain insufficient by
  itself: stray markers still require full Git validation.
- `newManagerWithOptions` resolves the main worktree for pre-config project
  matching, `workspace.DetectWithOptions` resolves it again, and
  `loadManagerContext` resolves it once more. The first lookup remains needed
  before sandbox mode is known, but the detection result can carry enough
  context to avoid the final duplicate lookup.
- These costs occur before `Manager.List`, so they remain plausible contributors
  even if status calculation is later proven dominant.

## Existing Implementation References

- `workspace/workspace.go`
  - `DetectWithOptions`, lines 51-129: initial child scan, main-root resolution,
    sandbox split, and containing-workspace selection.
  - `detectContainingWorkspace` / `isContainingWorkspaceRoot`, lines 131-205:
    bounded candidate traversal and candidate qualification scans.
  - `reposAtWorkspaceRoot` / `scanImmediateRepos`, lines 207-236: final repo
    enumeration and immediate-child validation dispatch.
  - `isStandaloneRepoRoot`, lines 288-333: three Git-backed checks that preserve
    real-repository and linked-worktree semantics.
- `cmd/ww/main.go`
  - `newManagerWithOptions`, lines 142-195: pre-config main-root lookup and
    optional sandbox re-resolution.
  - `loadManagerContext`, lines 197-224: workspace detection followed by another
    main-worktree lookup.
- `workspace/workspace_test.go`, lines 25-298: standalone, workspace, symlink,
  unreadable-entry, and cheap-prefilter coverage.
- `docs/specs/workspace-discovery.md`, lines 30-80: bounded discovery, sandbox,
  and immediate-child behavior that must not change.
- `docs/specs/testing.md`, lines 12-27: the existing command-boundary benchmark
  and scalable fixture contract.
- `internal/testutil/performance_fixture.go`, lines 12-165: the versioned
  repository/worktree scale used by the benchmark.

## Code Change Map

- `workspace/workspace.go` (MODIFY)
  - Introduce an invocation-local discovery context that caches normalized
    `scanImmediateRepos` results by absolute candidate path.
  - Return defensive slice copies so appending the workspace root repository or
    normalizing results cannot mutate a cached candidate result.
  - Add a cheap, no-follow `.git` marker eligibility check after the existing
    `DirEntry` classification and before Git validation. Continue to validate
    every surviving marker with top-level/git-dir/common-dir checks.
  - Preserve missing-entry and permission-error handling and do not introduce a
    process-global cache whose repository set could become stale.
  - Expose detection context internally or additively so the already-resolved
    current main root can be reused by manager construction without changing
    `Detect` / `DetectWithOptions` callers.
- `workspace/workspace_test.go` (MODIFY)
  - Prove the same candidate is scanned once per detection invocation.
  - Prove returned repo slices do not alias cached state.
  - Cover real repos, linked worktrees, stray/partial `.git` markers, symlink
    children, disappearing entries, permission failures, and non-repo
    directories rejected before Git validation.
- `cmd/ww/main.go` (MODIFY)
  - Consume the detection result's main-root context in `loadManagerContext`.
  - Keep the pre-config lookup and the existing sandbox-config rerun semantics.
- `cmd/ww/config_test.go` (MODIFY)
  - Add regression coverage that repo-local/global project matching and
    config-enabled sandbox reruns resolve exactly as before.
- `integration_test.go` (MODIFY)
  - Retain end-to-end coverage for child-repo, git-backed workspace-root,
    non-git workspace-root, nearest parent/grandparent, and sandbox entry paths.

## Spec Changes

N/A - this is a behavior-preserving refactor. The black-box discovery contract
in `docs/specs/workspace-discovery.md` and the performance measurement contract
in `docs/specs/testing.md` remain authoritative and must pass unchanged.

## Sub-tasks

1. Capture current-main evidence
   - Run `make perf-check` and retain all three samples and medians for
     `BenchmarkWorkspaceList` and `BenchmarkWorkspaceCleanDryRun`.
   - Run the same benchmark command with a higher repository count using the
     existing `WW_PERF_REPOS` override to amplify discovery cost.
   - Collect `make perf-profile PERF_PROFILE_DIR=<path>` output before changing
     discovery and record whether startup/process work is visible.
2. Reuse discovery work
   - Add the invocation-local scan context, defensive copies, and cheap marker
     prefilter.
   - Thread the detected main root into manager initialization while preserving
     pre-config and sandbox rerun behavior.
3. Prove behavior and improvement
   - Add unit/integration regressions for every discovery boundary above.
   - Re-run the exact default and repository-heavy measurements on the same
     host. Record before/after samples in the implementation PR.
   - Do not land the refactor as a performance change if the target measurement
     does not improve outside the captured pre-change run-to-run range; keep
     any independently valuable correctness-neutral cleanup separately scoped.

## Dependencies and Parallelism

- This plan and `0039-list-clean-performance-02-tracking-batch.md` touch separate
  primary packages and may be executed independently.
- Complete this plan before bounded cross-repository parallelism so the later
  plan does not hide redundant discovery work behind concurrency.
- Do not combine patch-equivalence or clean-removal changes into this plan.

## Verification

- `go test ./workspace ./cmd/ww`
- `go test ./...`
- `go test -race ./workspace ./cmd/ww`
- `make test`
- `make test-all`
- `make lint`
- `make perf-check`
- Run the versioned benchmark command with the same higher
  `WW_PERF_REPOS` value before and after the change.
- Confirm the implementation PR's GitHub-hosted performance job remains within
  the versioned 400 ms list and 420 ms clean advisory budgets. Local runs use
  same-host before/after samples and are not required to meet those CI-specific
  thresholds; record PR #280's initial 258.1 ms / 269.6 ms GitHub medians only
  as historical context.
