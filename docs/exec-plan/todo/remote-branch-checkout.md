# Remote Branch Checkout

**Execution**: Use `/execute-task` to implement this plan.

## Objective

Add a git-native way to create an isolated worktree from an existing remote branch, without introducing any `gh` dependency into `ww`.

The target workflow is:

```sh
ww create --guess-remote <branch>
```

When the named branch exists on the remote in a form Git can resolve, `ww` should create a worktree that checks out that branch instead of creating a new branch from the default base.

This mode exists for two adjacent workflows:

- local checkout for review, inspection, and debugging of remote work such as an open PR branch
- taking over an existing remote branch and continuing to add commits from a separate local worktree

To make that contract reliable, `ww` should refresh remote information before attempting the Git-native remote-guessing checkout path. Users should not need to remember a manual `git fetch` step before `ww create --guess-remote <branch>`.

## Context

- Project goal: `ww` should be a portable, git-native worktree CLI that works across environments without repo-host-specific assumptions.
- Core beliefs: keep behavior correct and explicit before optimizing convenience.
- Existing behavior:
  - `ww create <branch>` creates a new branch from `default_base`, `origin/HEAD`, or the heuristic fallback when the branch does not exist locally.
  - if the branch already exists locally, `ww` adds a worktree for that existing local branch.
- Gap reported in GitHub issue `#227`: users often need a local worktree for a branch that exists only on the remote, whether to review it locally or to continue work on top of that existing remote branch. Current `ww create <branch>` creates the wrong thing in that case because it starts a new branch instead.
- Constraint from the issue discussion: avoid `gh`-based PR resolution and prefer pure Git capabilities.

## Trade-offs

### Option A: Auto-detect remote branch existence inside plain `ww create`

If `origin/<branch>` exists, silently treat `ww create <branch>` as a remote checkout instead of a new branch.

- Pros: shortest user command.
- Cons: ambiguous when a user intentionally wants a new branch name that collides with a stale or unrelated remote branch; changes long-standing `ww create` behavior.
- Recommendation: reject for the first step.

### Option B: Add explicit `--guess-remote` support and delegate resolution to Git

Expose an opt-in flag on `ww create` that uses Git's own remote-guessing behavior for existing remote branches.

- Pros: no `gh` dependency; minimal new product surface; explicit intent; mirrors Git terminology; preserves current default behavior for normal feature creation.
- Cons: only covers the "same branch name as remote branch" case directly; depends on the Git version's `worktree add --guess-remote` behavior.
- Recommendation: use this approach first.

### Option C: Add a more general custom flag such as `--track <start-point>` or a new subcommand

Teach `ww` its own start-point selection model for remote refs or add a dedicated checkout command.

- Pros: more flexible and potentially closer to raw `git worktree add` power.
- Cons: larger CLI and spec surface; higher testing/documentation cost; unnecessary for the reported problem.
- Recommendation: defer unless execution finds that `--guess-remote` is too limited or not portable enough.

## Spec Changes

Update specs before code changes:

- `docs/specs/cli-commands.md`
  - add `--guess-remote` to `ww create`
  - define that this mode fetches the relevant remote information first, then requests checkout of an existing remote branch via Git-native resolution rather than new-branch creation from the default base
  - define the failure mode when Git cannot resolve a matching remote branch
  - define the failure mode when the installed Git does not support `git worktree add --guess-remote`
  - clarify precedence between three cases:
    - existing local branch
    - explicit `--guess-remote`
    - normal new-branch creation
- `docs/specs/git-operations.md`
  - add the exact Git operation used for this mode
  - clarify that the remote-branch checkout path is Git-driven and does not require `gh`
  - add the pre-checkout fetch contract
  - define the actionable error for unsupported Git versions: surface the original Git error, tell the user to upgrade Git, and include a manual `git worktree add ... --track ...` fallback path that still achieves the same isolated-worktree workflow
  - document any supported limitations discovered during implementation, especially around remote uniqueness and branch-name matching

Update human-facing usage docs if execution changes the supported command examples:

- `README.md`

## Code Changes

Planned implementation scope:

- `cmd/ww/sub_create.go`
  - add `--guess-remote` flag plumbing for `ww create`
- `cmd/ww/helpers.go`
  - thread the new create option into `worktree.CreateOpts`
- `worktree/worktree.go`
  - extend create logic with an explicit remote-checkout path that bypasses default-base resolution
  - preserve current behavior for existing local branches and normal new-branch creation
- `git/git.go`
  - add a dedicated helper for the Git-native remote-checkout operation if needed, instead of overloading the current "new branch" and "existing local branch" helpers
  - add a helper for the pre-checkout fetch path and any Git-version-sensitive error shaping needed for unsupported `--guess-remote`
- `internal/interactive/*`
  - decide whether the interactive create flow should stay unchanged for now or expose the new mode explicitly; if unchanged, document that this plan only covers the non-interactive CLI path

## Sub-tasks

- [ ] Confirm the chosen Git invocation and its behavior when:
  - the branch exists only on the remote
  - no matching remote branch exists
  - a same-named local branch already exists
  - the remote branch exists but was not fetched before the command started
  - the installed Git does not support `git worktree add --guess-remote`
- [ ] Update `docs/specs/cli-commands.md` with the explicit-flag contract.
- [ ] Update `docs/specs/git-operations.md` with the Git-native operation and limits.
- [ ] Implement `ww create --guess-remote <branch>` in the non-interactive CLI path.
- [ ] Make the `--guess-remote` path fetch remote information before attempting checkout.
- [ ] Return an actionable unsupported-Git error that includes:
  - the original Git error text
  - a request to upgrade Git
  - a manual `git worktree add -b <branch> --track <path> <remote>/<branch>` fallback path
- [ ] Add unit/integration coverage for remote-only branch checkout and failure cases.
- [ ] Update `README.md` examples if the new mode is accepted as public user-facing behavior.

## Verification

- `go test ./...`
- targeted tests covering:
  - create from a remote-only branch with `--guess-remote`
  - create from a remote branch that becomes available only after the command's fetch step
  - failure when `--guess-remote` cannot resolve a remote branch
  - failure when `git worktree add --guess-remote` is unsupported, including the actionable error text
  - unchanged behavior for normal `ww create <branch>`
  - unchanged behavior for existing local branches
- optional manual confirmation in a temp repo with a bare `origin` remote and a branch that exists remotely but not locally

## Design Decisions

No ADR update is expected if execution stays with the narrow explicit-flag shape. If the work uncovers portability problems with `--guess-remote` across supported Git versions, record the fallback decision before expanding the CLI to a custom `--track` or a separate checkout subcommand.
