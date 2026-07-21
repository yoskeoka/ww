# List/Clean Performance 04: Bound and Parallelize Repository Enumeration
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Run independent repository status enumeration concurrently in workspace mode
with a small fixed bound, while preserving deterministic repository/worktree
output order, conservative error behavior, single-repo behavior, and all status
semantics shared by `ww list` and `ww clean`.

Completion means a slow repository no longer serially blocks every later
repository, concurrency never becomes unbounded as workspace size grows, race
coverage passes, and before/after measurements use the list and clean benchmarks
introduced by [PR #280](https://github.com/yoskeoka/ww/pull/280).

Addresses: N/A - performance follow-up to the completed measurement issue
`docs/issues/done/0037-list-clean-performance-budget.md`.

## Static Analysis Findings

- `listWorkspace` loops over normalized `Workspace.Repos` and calls `listRepo`
  synchronously. Repository status computation has no shared mutable Git state;
  each call creates a runner rooted at that repository.
- Per-repository work includes local Git subprocesses and a live
  `ls-remote --heads` call per unique remote. Remote latency can therefore make
  wall time approximately additive across repositories even after local query
  counts are reduced.
- PR #280 uses local bare remotes, so it captures scheduling/process overhead
  but not real network latency. It remains the required regression indicator,
  while correctness tests with controlled delayed repository workers are needed
  to prove concurrency deterministically.
- Unbounded one-goroutine-per-repository execution could create a Git process
  storm in large workspaces. A fixed small cap and indexed result slots preserve
  both resource bounds and output order.

## Existing Implementation References

- `workspace/workspace.go`, lines 371-400: normalized, deterministic repository
  ordering supplied to the manager.
- `worktree/worktree.go`
  - `Manager.List`, lines 354-360: single/workspace dispatch.
  - `listWorkspace`, lines 444-454: current serial repository loop.
  - `listRepo`, lines 456-540: independent repository-scoped status work.
- `git/git.go`, lines 298-326: live per-repository remote enumeration that may
  block on remote latency.
- `cmd/ww/sub_list.go`, lines 26-73: output consumes the returned order directly.
- `cmd/ww/sub_clean.go`, lines 58-163: clean consumes the same ordered snapshot
  and continues per-target failures in that order.
- `worktree/worktree_test.go`, lines 833 onward: workspace list aggregation.
- `docs/specs/cli-commands.md`, lines 377-441 and 503 onward: deterministic
  observable list/clean contents and shared status contract.
- `docs/specs/testing.md`, lines 12-27: scalable PR #280 benchmark fixture.

## Design Choice

- Rejected: keep serial enumeration. It retains additive latency across
  independent repositories.
- Rejected: unbounded goroutines. It can overwhelm the host and remote servers
  as repository count grows.
- Chosen: a fixed small worker bound (four concurrent repositories) with one
  result slot per original repo index. Wait for in-flight work, then return the
  lowest-index error if any; otherwise flatten slots in original order.

The bound is an internal resource policy, not a user-facing flag. A later change
to make it configurable requires separate evidence and planning.

## Code Change Map

- `worktree/worktree.go` (MODIFY)
  - Replace the serial `listWorkspace` loop with a four-worker bounded executor.
  - Store each repository's infos/error by its original index and flatten
    successful results in the existing order.
  - If multiple repositories fail, return the error belonging to the earliest
    repository in deterministic workspace order. Do not emit partial output.
  - Keep single-repo `List` unchanged.
- `worktree/worktree_test.go` (MODIFY)
  - Add an internal list-repo seam or focused helper tests with controlled
    blocking workers.
  - Prove maximum concurrency, actual overlap, stable output order, deterministic
    earliest error, no partial result on error, empty workspace handling, and
    single-repo non-participation.
- `integration_test.go` (MODIFY)
  - Prove list/clean text and JSON ordering across multiple repositories remains
    unchanged and all cleanable targets are retained.

## Spec Changes

N/A - concurrency is internal. `docs/specs/cli-commands.md` remains unchanged:
repository/worktree order, status values, errors, and cleanable behavior are the
same. `docs/specs/testing.md` already supports increasing `WW_PERF_REPOS` for
this investigation.

## Sub-tasks

1. Capture serial baseline
   - Run `make perf-check` and a repository-heavy benchmark profile with a fixed
     higher `WW_PERF_REPOS` value.
   - Retain profiles and samples after plans 0038-0040 have landed so this plan
     measures concurrency rather than avoidable duplicate work.
2. Add bounded scheduling
   - Introduce the smallest testable internal seam and four-worker execution.
   - Preserve original-index result/error collection and avoid writes to shared
     slices without synchronization.
3. Verify behavior, races, and improvement
   - Run controlled concurrency tests and the race detector.
   - Re-run default and repository-heavy benchmarks on the same host.
   - Require repository-heavy improvement outside baseline variation and no
     meaningful default-profile regression before landing.

## Dependencies and Parallelism

- Depends on plans 0038, 0039, and 0040. Concurrency is deliberately last in
  the shared list/status path so it cannot mask or multiply redundant work.
- Test seam design and output-order regression cases may be prepared in
  parallel, but scheduling implementation depends on the final post-0040
  `listRepo` signature.
- Plan 0042 follows this plan and optimizes only clean's post-list phase.

## Verification

- `go test ./worktree`
- `go test -race ./worktree`
- `go test ./...`
- `make test`
- `make test-all`
- `make lint`
- `make perf-check`
- Run the same repository-heavy benchmark before and after.
- Confirm stable text/JSON ordering and deterministic earliest-repo error.
- Confirm the implementation PR's GitHub-hosted performance job keeps the
  unchanged default benchmarks within 400 ms / 420 ms. Judge local improvement
  against the same-host baseline; PR #280's 258.1 ms / 269.6 ms samples are
  historical runner calibration only.
