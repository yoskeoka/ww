# List/Clean Performance 05: Reuse Clean Snapshot for Dry-Run
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Avoid reloading the same repository config and re-running `git worktree list`
for every cleanable target during `ww clean --dry-run`. Reuse the authoritative
`Manager.List` snapshot only for non-mutating preview; preserve per-target
revalidation for real removals, lifecycle-hook preview order, JSON/text output,
failure continuation, and removal safety.

Completion means dry-run post-list work scales with repositories rather than
cleanable worktrees, real `ww clean` retains its current revalidation/mutation
path, and PR #280's `BenchmarkWorkspaceCleanDryRun` plus the companion list
benchmark provide recorded before/after evidence.

Addresses: N/A - performance follow-up to the completed measurement issue
`docs/issues/done/0037-list-clean-performance-budget.md`.

## Static Analysis Findings

- `listCleanableWorktrees` already obtains one complete, status-bearing snapshot
  through `Manager.List`.
- `executeCleanWorktrees` then calls `managerForRepo` for every cleanable item.
  In workspace mode this reloads the selected repo config, potentially twice
  when sandbox mode must first be discovered.
- `Manager.Remove` calls `WorktreeList` again for every target before it reaches
  its dry-run branch. This is necessary safety for mutation, but redundant for a
  preview based on the snapshot produced immediately above.
- PR #280's default fixture contains one cleanable worktree per repository. It
  measures the extra pass but does not expose repeated same-repository config
  and worktree-list costs. An opt-in cleanable-density fixture setting is needed
  without changing the default budgeted profile.

## Existing Implementation References

- `cmd/ww/sub_clean.go`
  - `listCleanableWorktrees`, lines 58-64: authoritative status snapshot and
    cleanable filter.
  - `executeCleanWorktrees`, lines 66-163: per-target manager resolution,
    removal/preview dispatch, output, and failure continuation.
- `cmd/ww/helpers.go`
  - `managerForRepo`, lines 40-74: per-call repo selection and config reload.
  - `loadRepoConfigForSelection`, lines 76-108: non-sandbox pre-load and final
    effective config resolution.
- `worktree/worktree.go`
  - `Manager.Remove`, lines 688-762: per-target `WorktreeList` lookup followed by
    the dry-run rendering or real mutation path.
- `docs/specs/cli-commands.md`, lines 503 onward: clean selection, output,
  continuation, dry-run, and removal safety contract.
- `internal/testutil/performance_fixture.go`, lines 12-165: current one-merged-
  worktree-per-repository fixture shape.
- `cmd/ww/performance_test.go`, lines 58-77: clean dry-run benchmark assertion.
- `tools/performance-budget.json`, lines 1-28: default 6 x 5 profile and 420 ms
  clean budget introduced by PR #280.
- `docs/specs/testing.md`, lines 12-27: fixture/budget and scale-override contract.

## Code Change Map

- `docs/specs/testing.md` (MODIFY)
  - Retain the default budgeted profile and add a documented optional
    cleanable-worktrees-per-repo override for targeted clean-heavy comparison.
- `internal/testutil/performance_fixture.go` (MODIFY)
  - Add a validated opt-in cleanable-density setting; default it to the existing
    one merged worktree per repository.
  - Create deterministic additional merged targets while retaining at least one
    active tracked branch and exact expected output counts.
- `tools/performance-budget.json` (MODIFY)
  - Record the default cleanable density explicitly without changing benchmark
    names, repetitions, or the existing default command shape and limits.
- `tools/check-performance-budget.go` (MODIFY)
  - Parse/validate/report the additional versioned fixture-profile field.
- `tools/check-performance-budget_test.go` (MODIFY)
  - Cover valid/invalid cleanable density and report formatting.
- `worktree/worktree.go` (MODIFY)
  - Extract shared validated preview construction from `Remove` after entry
    lookup.
  - Add a narrow preview method that accepts a `WorktreeInfo` originating from
    the current list snapshot, rejects main/branchless/invalid inputs, and emits
    the exact same `RemoveResult` and dry-run log without invoking Git.
  - Keep non-dry-run `Remove` lookup and mutation behavior unchanged.
- `worktree/worktree_test.go` (MODIFY)
  - Prove snapshot preview and normal `Remove(...DryRun:true)` are equivalent for
    hooks, path, branch, worktree index, keep-branch behavior, and errors.
- `cmd/ww/sub_clean.go` (MODIFY)
  - Cache `managerForRepo` results once per repository while iterating the
    original target order.
  - Use snapshot preview only when `glOpts.dryRun`; route real removals through
    existing `Manager.Remove` so each mutation revalidates current Git state.
  - Preserve per-target failure accumulation and text/JSON ordering.
- `cmd/ww/sub_clean_test.go` (NEW)
  - Cover repo-manager caching, dry-run snapshot dispatch, real-remove dispatch,
    ordered output, and continued failures with focused seams.
- `integration_test.go` (MODIFY)
  - Compare `ww list --cleanable` with text/JSON `ww clean --dry-run` in a repo
    containing multiple cleanable targets and lifecycle hook configuration.

## Spec Changes

- `docs/specs/testing.md`: add the optional cleanable-density investigation
  profile while keeping PR #280's default budgeted profile intact.
- `docs/specs/cli-commands.md`: no semantic change. Dry-run output and actual
  cleanup safety/revalidation remain as currently specified.

## Sub-tasks

1. Make repeated clean preview work measurable
   - Add the optional cleanable-density fixture setting as a measurement-only
     first commit and capture a clean-heavy baseline before optimization.
   - Also run the unchanged default `make perf-check` and retain list/clean
     samples so fixture/tool changes cannot hide a regression.
2. Reuse repository and snapshot state safely
   - Extract and test shared preview rendering.
   - Cache selected repo managers for the command invocation.
   - Use snapshot preview only for dry-run; preserve fresh lookup before every
     real mutation.
3. Verify parity and performance
   - Prove normal `remove --dry-run`, clean snapshot preview, text, and JSON
     outputs agree.
   - Re-run default and clean-heavy measurements on the same host.
   - Require clean-heavy improvement outside baseline variation, unchanged list
     behavior, and no meaningful default-profile regression before landing.

## Dependencies and Parallelism

- Follows plan 0041 so the shared list snapshot reflects all preceding
  discovery/status/enumeration improvements.
- The fixture-density extension and preview parity tests may proceed in
  parallel. Command adoption depends on the shared preview helper.
- Do not optimize actual mutation by trusting the initial snapshot in this plan;
  weakening per-target revalidation would be a separate safety decision.

## Verification

- `go test ./cmd/ww ./worktree ./tools/...`
- `go test ./...`
- `make test`
- `make test-all`
- `make lint`
- `make perf-check`
- Run the documented clean-heavy profile before and after with identical
  repository/worktree/cleanable counts.
- Confirm the implementation PR's GitHub-hosted performance job keeps default
  list and clean within 400 ms / 420 ms. Judge local improvement from same-host
  before/after samples; PR #280's 258.1 ms list and 269.6 ms clean medians are
  historical GitHub-runner context only.
