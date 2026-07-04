# Clean Squash Rebase Merged Branches
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Make `ww list --cleanable` and `ww clean` treat worktree branches whose changes
are already integrated into the resolved base branch via squash merge or rebase
merge as `merged`, even when `git branch --merged <base>` does not report them
and the local branch has no tracking remote. Completion means an operator can
run `ww clean` after a squash-merged or rebase-merged PR flow and have the
corresponding worktree cleaned without depending on upstream tracking state.

Addresses: https://github.com/yoskeoka/ww/issues/263

## Existing Implementation References

- `worktree/worktree.go`
  - `listRepoWithStatus`, lines 355-418 - current merged/stale status pipeline,
    including the `git branch --merged` fast path and stale fallback.
  - `resolveStatus`, lines 521-536 - current precedence order:
    main/branchless -> merged -> stale -> active.
- `git/git.go`
  - `MergedBranches`, lines 115-133 - current direct-merge detection based on
    `git branch --merged <base>`.
  - `BranchRemote`, `BranchMergeRef`, `ListRemoteBranches`, lines 136-198 -
    existing branch tracking and remote existence helpers that must continue to
    power stale detection after merged detection is widened.
- `worktree/worktree_test.go`
  - merged/stale status fixture setup and `resolveStatus` table, lines 192-260 -
    current unit-level expectations and precedence coverage.
- `integration_test.go`
  - heuristic-base cleanable flow, lines 2242-2318 - current end-to-end proof
    that `ww list --cleanable` and `ww clean` share one status contract.
- `docs/specs/cli-commands.md`
  - list/clean status contract, lines 320-446 - current user-visible definition
    of `merged`, `stale`, and `cleanable`.
- `docs/specs/git-operations.md`
  - create/unset-upstream behavior, lines 44-54 - why normal `ww create` leaves
    no upstream.
  - merged-branch detection contract, lines 105-110 - current Git-level
    description that is limited to `git branch --merged <base>`.

## Code Change Map

- `git/git.go` (MODIFY)
  - Add Git-native helpers for patch-equivalence merged detection against the
    resolved base branch, keeping the existing `git branch --merged` fast path
    and avoiding any GitHub/`gh` dependency.
- `git/git_test.go` (MODIFY)
  - Add focused helper tests for the new merged-detection parsing or Git command
    wrapper behavior.
- `worktree/worktree.go` (MODIFY)
  - Expand merged-set construction so branches already integrated by squash
    merge or rebase merge are marked `merged` before stale detection runs.
- `worktree/worktree_test.go` (MODIFY)
  - Add unit-level precedence coverage for patch-equivalent merged branches,
    ensuring merged still wins over stale and local-only active remains active.
- `integration_test.go` (MODIFY)
  - Add repo-level scenarios proving `ww list --cleanable` and `ww clean` both
    cover direct merge, squash merge, rebase merge, tracked stale, and active
    branches under the same base-resolution path.
- `docs/specs/cli-commands.md` (MODIFY)
  - Redefine the externally visible `merged` status and cleanable contract so it
    includes branches whose changes are already integrated into the base via
    direct merge, squash merge, or rebase merge.
- `docs/specs/git-operations.md` (MODIFY)
  - Document the Git-native merged-detection sequence: direct ancestor fast path
    first, patch-equivalence fallback second, stale tracking checks only after
    merged detection remains false.

## Spec Changes

- `docs/specs/cli-commands.md`
  - Update `ww list` status definitions and `ww clean` behavior so `merged`
    means "already integrated into the resolved base branch", not only
    "reported by `git branch --merged`".
  - Clarify that squash-merged and rebase-merged branches are cleanable even if
    the local branch does not track a remote branch.
- `docs/specs/git-operations.md`
  - Describe the low-level Git detection strategy used to classify integrated
    branches, including the existing direct-merge fast path and the new
    patch-equivalence fallback for non-ancestor merge styles.

## Sub-tasks

- [ ] [parallel] Update `docs/specs/cli-commands.md` and
  `docs/specs/git-operations.md` to define the widened merged contract.
- [ ] [parallel] Add Git runner support for patch-equivalence merged detection
  against a resolved base branch.
- [ ] [depends on: Git runner support for patch-equivalence merged detection against a resolved base branch]
  Update `worktree/worktree.go` merged-set construction and status precedence.
- [ ] [depends on: Update docs/specs/cli-commands.md and docs/specs/git-operations.md to define the widened merged contract, Update worktree/worktree.go merged-set construction and status precedence]
  Extend unit and integration coverage for direct, squash, rebase, stale, and
  active branches.
- [ ] [depends on: Extend unit and integration coverage for direct, squash, rebase, stale, and active branches]
  Run targeted and full verification for the clean/status surface.

## Design Decisions

- Rejected for this task: remote-gone-only cleanup heuristics. They depend on
  branch tracking state and still miss local branches created without upstreams.
- Rejected for this task: GitHub API / `gh`-based merged PR detection. It adds a
  network/tooling dependency to a local Git-native status path.
- Chosen direction: keep `git branch --merged <base>` as the cheap fast path,
  then apply a Git-native patch-equivalence fallback for remaining candidate
  branches so squash merge and rebase merge land in the same `merged` status.
- Expected behavior boundary: a branch should only become `merged` through patch
  equivalence when its branch-intended changes are already present in the base;
  branches with extra unpublished work must remain `active` or `stale`.
- ADR update is not expected unless implementation reveals a materially new
  long-term contract beyond the status semantics captured in specs.

## Parallelism

- The two spec updates can be drafted in parallel with the low-level Git helper
  work.
- Unit-test additions for `git/git.go` and `worktree/worktree_test.go` can be
  developed in parallel once the fallback command shape is settled.
- Integration coverage depends on the chosen detection shape because the test
  fixtures must mirror the final merged-precedence contract.

## Verification

- `go test ./...`
- `make test`
- `make lint`
- Targeted integration coverage proving:
  - direct-merge branches remain `merged`
  - squash-merged branches become `merged` without relying on tracking
  - rebase-merged branches become `merged` without relying on tracking
  - tracked remote-gone branches with unapplied changes remain `stale`
  - local-only unapplied branches remain `active`
