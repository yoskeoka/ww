# List/Clean Performance 02: Batch Branch Tracking Metadata
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Replace per-worktree branch-remote Git subprocesses in the `ww list` / `ww clean`
status path with one repository-scoped tracking metadata query. Preserve status
values, `merged > stale` precedence, remote-gone behavior, local-only active
behavior, error handling, and output ordering.

Completion means branch tracking metadata scales with repositories rather than
worktrees, all existing status scenarios remain byte-for-byte compatible at the
CLI boundary, and the PR records before/after medians from the command-boundary
benchmarks introduced by [PR #280](https://github.com/yoskeoka/ww/pull/280).

Addresses: N/A - performance follow-up to the completed measurement issue
`docs/issues/done/0037-list-clean-performance-budget.md`.

## Static Analysis Findings

- `listRepo` already batches remote branch existence to one `ls-remote --heads`
  per unique remote within a repository.
- Before that batch, it calls `BranchRemote` once for every non-main,
  non-merged worktree branch. `BranchRemote` shells out to
  `git config --get branch.<branch>.remote`, so this portion is O(worktrees)
  process startup even when every branch tracks the same remote.
- Tracking metadata is repository-local and can be read in one Git ref/config
  query, then projected back onto the existing entry order. The live
  `ls-remote` call cannot be replaced with local remote-tracking refs without
  changing stale semantics and is outside this plan.
- Patch-equivalence work is a separate larger source of O(worktrees/history)
  cost and is intentionally isolated in plan 0040.

## Existing Implementation References

- `worktree/worktree.go`
  - `Manager.List` / `listWorkspace`, lines 354-453: shared list/clean entry and
    repository traversal.
  - `listRepo`, lines 456-540: merged-set creation, per-entry `BranchRemote`
    loop, per-remote branch listing, and final status projection.
  - `resolveStatus`, lines 643-658: required merged/stale/active precedence.
- `git/git.go`
  - `BranchRemote`, lines 258-266: one config subprocess per branch.
  - `ListRemoteBranches`, lines 298-326: existing one-call-per-remote batch and
    parsing behavior that remains unchanged.
- `worktree/worktree_test.go`, lines 317-540: repository-backed status cases.
- `git/git_test.go`, lines 463-523 and nearby branch-helper coverage: real Git
  runner behavior.
- `docs/specs/cli-commands.md`, lines 377-441 and 503 onward: externally visible
  status and cleanable contract.
- `docs/specs/git-operations.md`, lines 105-153: current per-branch tracking and
  remote existence description.
- `docs/specs/testing.md`, lines 12-27: PR #280 benchmark/budget contract.

## Code Change Map

- `docs/specs/git-operations.md` (MODIFY)
  - Describe repository-scoped bulk tracking metadata resolution while
    preserving the meaning of configured remote names and live remote branch
    existence checks.
- `git/git.go` (MODIFY)
  - Add a batch helper that returns branch-to-remote tracking metadata for the
    requested local branches with one Git subprocess.
  - Read the configured `branch.<name>.remote` values in bulk rather than
    inferring them from a resolved upstream ref; current behavior does not
    require `branch.<name>.merge` to be present.
  - Use a machine-parseable delimiter/format that supports slashes and unusual
    valid branch names without whitespace ambiguity.
  - Preserve the distinction between no configured remote and a configured
    remote whose branch is absent.
- `git/git_test.go` (MODIFY)
  - Cover multiple tracked branches on one remote, multiple remotes,
    untracked/local-only branches, a configured remote without a merge ref, a
    configured-but-gone upstream, and branch names containing slashes.
- `worktree/worktree.go` (MODIFY)
  - Request tracking metadata once for the remaining non-main/non-merged
    branches and keep the current `ListRemoteBranches` cache per unique remote.
  - Preserve entry order and all status/error behavior.
- `worktree/worktree_test.go` (MODIFY)
  - Prove direct merged, patch merged, stale, local-only active, main, detached,
    and unknown-base entries classify exactly as before.
- `integration_test.go` (MODIFY)
  - Retain end-to-end `ww list --cleanable` / `ww clean --dry-run` parity with
    several worktree branches sharing one remote and branches spanning multiple
    configured remotes.

## Spec Changes

- `docs/specs/git-operations.md`: update the low-level Git query contract from
  one `git config --get` invocation per branch to a repository-scoped bulk
  tracking query.
- `docs/specs/cli-commands.md`: no change. Status values, precedence, output,
  cleanable selection, and live remote-gone semantics remain unchanged.

## Sub-tasks

1. Capture baseline and command shape
   - Run `make perf-check` on current main and retain samples/medians.
   - Run the same benchmark with a higher
     `WW_PERF_WORKTREES_PER_REPO` value to expose per-branch scaling.
   - Confirm via profiling or a test Git wrapper that the current path invokes
     branch tracking lookup once per eligible branch.
2. Add and adopt the batch query
   - Update the Git-operations spec first.
   - Implement and test the batch parser/helper.
   - Replace only the `BranchRemote` loop in `listRepo`; keep patch-equivalence,
     base resolution, and live remote enumeration unchanged.
3. Verify semantics and performance
   - Run status unit/integration coverage and compare text/JSON output.
   - Re-run default and worktree-heavy benchmark profiles on the same host and
     record before/after samples in the implementation PR.
   - If the target profile does not improve outside the baseline variation,
     stop and use the evidence to re-evaluate this refactor before landing it.

## Dependencies and Parallelism

- May execute independently of plan 0038.
- Must land before plan 0040 because both change Git/status aggregation and the
  later plan should benchmark against the already-batched tracking path.
- Plan 0041's bounded concurrency waits for this per-repository subprocess
  reduction so concurrency does not multiply avoidable Git processes.

## Verification

- `go test ./git ./worktree`
- `go test ./...`
- `make test`
- `make test-all`
- `make lint`
- `make perf-check`
- Run the versioned benchmark command with the same increased
  `WW_PERF_WORKTREES_PER_REPO` value before and after.
- Confirm the implementation PR's GitHub-hosted performance job stays within
  the 400 ms / 420 ms advisory budgets. Use same-host samples, not those limits,
  for local before/after judgment; PR #280's 258.1 ms / 269.6 ms medians are
  historical GitHub-runner context only.
