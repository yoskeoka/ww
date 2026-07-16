# List/Clean Performance 03: Amortize Patch-Equivalence Status Work
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Reduce repeated Git history and patch-id work when many non-directly-merged
worktree branches are checked for rebase/cherry-pick/squash equivalence against
the same base. Preserve the exact conservative `merged` classification: no
branch may become cleanable unless its branch-intended changes are already
integrated in the resolved base.

Completion means base-side patch data is computed at most once per relevant
merge-base within one repository listing, direct-merge and `git cherry` fast
paths remain intact, all merge-style regressions pass, and PR #280's list and
clean command-boundary measurements are recorded before and after the change.

Addresses: N/A - performance follow-up to the completed measurement issue
`docs/issues/done/0037-list-clean-performance-budget.md`.

## Static Analysis Findings

- `PatchEquivalentBranches` iterates every candidate serially.
- A candidate not fully integrated according to `git cherry` enters
  `branchSquashEquivalent`, which resolves a merge-base, computes the aggregate
  branch patch-id, lists every base commit after that merge-base, and computes a
  patch-id for each base commit until it finds a match.
- Worktree branches in one repository commonly share the same merge-base.
  Current code repeats `rev-list` and base commit `show | patch-id` work for each
  such branch, creating O(candidate branches x base history) subprocess and
  diff work.
- PR #280's default fixture uses empty commits for secondary branches. It
  exercises candidate dispatch but can exit before non-empty aggregate patch
  and base-history comparison, so the existing benchmark alone may understate
  this hot path. A supplementary non-empty branch-shape mode is needed while
  retaining the original default benchmark and budgets.

## Existing Implementation References

- `git/git.go`
  - `PatchEquivalentBranches`, lines 152-165: serial candidate dispatch.
  - `branchPatchEquivalent`, lines 167-178: `git cherry` fast path and squash
    fallback.
  - `branchSquashEquivalent`, lines 202-236: per-candidate merge-base, branch
    aggregate, base history, and base commit patch-id loop.
  - `diffPatchID`, lines 238-256: diff/show to stable patch-id conversion.
- `worktree/worktree.go`, lines 478-505: selection of non-directly-merged
  worktree branches and incorporation into the merged set.
- `docs/specs/git-operations.md`, lines 105-137: direct, commit-equivalent, and
  squash-equivalent merged-detection contract.
- `docs/design-decisions/adr/0005-worktree-status-precedence.md`, lines 1-23:
  conservative merged-before-stale decision.
- `git/git_test.go` and `worktree/worktree_test.go`, especially the
  patch-equivalent and status scenarios related to `PatchEquivalentBranches`.
- `integration_test.go`, lines 2360 onward: squash/rebase cleanable CLI coverage.
- `internal/testutil/performance_fixture.go`, lines 12-165: scalable fixture and
  current empty-commit branch shape.
- `cmd/ww/performance_test.go`, lines 1-77: stable list and clean benchmarks.
- `docs/specs/testing.md`, lines 12-27: versioned default performance contract.

## Code Change Map

- `docs/specs/testing.md` (MODIFY)
  - Retain the default budgeted profile unchanged.
  - Document a supplementary, non-budgeted branch-content/profile override used
    for patch-equivalence investigations and before/after evidence.
- `internal/testutil/performance_fixture.go` (MODIFY)
  - Add an opt-in fixture mode that creates deterministic non-empty branch
    changes and enough base history to execute aggregate patch comparison.
  - Keep the default PR #280 fixture shape and expected counts unchanged.
- `cmd/ww/performance_test.go` (MODIFY)
  - Consume the supplementary fixture option without renaming or weakening the
    existing budgeted benchmark assertions.
- `git/git.go` (MODIFY)
  - Introduce invocation-local patch-equivalence state for a repository.
  - Preserve the direct merged-set and `git cherry` short circuits.
  - Cache base commit patch IDs by merge-base and compute them lazily only when
    a candidate reaches squash-equivalence comparison.
  - Do not create a process-global or cross-command cache; repository refs may
    change between invocations.
- `git/git_test.go` (MODIFY)
  - Cover multiple candidate branches sharing a merge-base and candidates with
    distinct merge-bases.
  - Prove direct merge, rebase/cherry-pick equivalence, exact squash
    equivalence, empty diff, partial integration, and extra unpublished changes.
- `worktree/worktree_test.go` (MODIFY)
  - Preserve merged/stale/active precedence when cached results are reused.
- `integration_test.go` (MODIFY)
  - Exercise multiple cleanable and non-cleanable merge styles in one repo and
    prove `ww list --cleanable` and `ww clean --dry-run` agree.

## Spec Changes

- `docs/specs/testing.md`: add only the supplementary patch-heavy investigation
  profile. The versioned default fixture, benchmark names, repetition count,
  and budgets remain unchanged.
- `docs/specs/cli-commands.md` and `docs/specs/git-operations.md`: no semantic
  change. Existing conservative merge-equivalence behavior remains mandatory.

## Sub-tasks

1. Make the hot path measurable
   - Add the opt-in non-empty branch-shape fixture mode as a measurement-only
     first commit.
   - Before implementing caching, run the default PR #280 benchmark and the new
     patch-heavy mode repeatedly on the same host; retain samples and profiles.
2. Cache base-side equivalence data
   - Refactor patch-equivalence evaluation around repository-local,
     merge-base-keyed lazy state.
   - Keep error attribution actionable by branch/base even when cached work
     produced the underlying failure.
3. Prove conservative behavior and improvement
   - Run merge-style unit/integration coverage, including negative cases.
   - Re-run default and patch-heavy profiles with identical scale/environment.
   - Require a patch-heavy improvement outside pre-change variation and no
     meaningful default-profile regression before landing.

## Dependencies and Parallelism

- Depends on plan 0039 to avoid overlapping Git/status refactors and to measure
  patch work after branch tracking subprocesses are already batched.
- Must land before plan 0041 so bounded repository parallelism does not conceal
  repeated base-history work or amplify it across repositories.
- Fixture-mode work and negative merge-style test expansion can proceed in
  parallel before the production cache is adopted.

## Verification

- `go test ./git ./worktree`
- Targeted integration coverage for direct, rebase/cherry-pick, squash,
  partially integrated, and extra-work branches.
- `go test ./...`
- `make test`
- `make test-all`
- `make lint`
- `make perf-check`
- Run the documented patch-heavy profile before and after with identical repo
  and worktree counts and record all samples in the PR.
- Confirm the implementation PR's GitHub-hosted performance job keeps the
  unchanged default benchmarks within 400 ms / 420 ms. Local pass/fail uses the
  same-host before/after range; PR #280's 258.1 ms / 269.6 ms medians remain
  historical GitHub-runner context only.
