# Base Detect Error Actionable
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Resolve `docs/issues/base-detect-error-not-actionable.md` by making base branch detection failures actionable when `ww create` cannot resolve a base branch from `default_base`, `origin/HEAD`, or the heuristic fallback path.

This supports the project plan's agent-friendly CLI goals: command failures should preserve enough detail for automation while also telling the operator how to fix common repository state problems.

## Current State

- `ww list` and `ww clean` already degrade gracefully when base detection fails, as completed by `016-unknown-status-default-base.md`.
- `017-heuristic-base-resolution.md` added fallback detection for common `main` / `master` cases when `origin/HEAD` is missing.
- The remaining issue is the hard failure path for `ww create`: when no base can be resolved, the error still exposes the raw `git symbolic-ref refs/remotes/origin/HEAD` failure and does not suggest setting `default_base` or repairing `origin/HEAD`.

## Prior Context

- `docs/project-plan.md` requires `ww` to be Git-native, composable, and agent-friendly.
- `docs/design-decisions/core-beliefs.md` favors correctness over speed and avoiding unnecessary refactors.
- `docs/design-decisions/adr.md` records a pattern of explicit CLI contracts and preserving Git behavior rather than hiding it.

## Approach

Recommended option: add a narrow, user-facing diagnostic for unresolved base branch failures, while keeping the underlying Git error available in the returned error chain or message context.

Viable options:

- Improve `git.Runner.DefaultBranch()` directly.
  - Pros: every caller gets a clearer error.
  - Cons: lower-level Git helper becomes aware of `.ww.toml` remediation language, which slightly mixes Git plumbing with `ww` configuration guidance.
- Improve `worktree.Manager.Create()` after authoritative and heuristic base resolution fail.
  - Pros: remediation text can be specific to create's requirement that a usable base must exist; keeps config guidance in worktree behavior.
  - Cons: other direct `DefaultBranch()` callers may still see the raw lower-level error if they surface it in new contexts later.
- Add a small typed/sentinel error around default branch detection in `git`, then format command-specific remediation in `worktree`.
  - Pros: clean separation and future-proofing.
  - Cons: more code than this issue needs unless more callers require nuanced handling.

Use the second option unless implementation reveals repeated formatting needs. It is the smallest change that matches the issue scope and the "do not refactor stable code for aesthetics" belief.

## Spec Changes

- Update `docs/specs/git-operations.md` to describe actionable diagnostics when base resolution fails for commands that require a base branch.
- Update `docs/specs/cli-commands.md` under `ww create` to state that unresolved base branch errors must explain:
  - no explicit `default_base` is configured,
  - `origin/HEAD` could not be used,
  - heuristic fallback could not find a usable `main` or `master` remote ref,
  - remediation: set `default_base` in `.ww.toml` or run `git remote set-head origin --auto` when the remote exposes a default branch.

No ADR update is expected because this is a CLI diagnostic refinement, not a new architectural decision.

## Code Changes

- `worktree/worktree.go`
  - Adjust the `Create` base-resolution failure path so the returned error is actionable.
  - Prefer a helper if the diagnostic is long enough to make inline formatting noisy.
- `worktree/worktree_test.go` or `integration_test.go`
  - Add coverage for `ww create` with no `default_base`, missing/unusable `origin/HEAD`, and no heuristic base.
  - Assert the error includes remediation text and does not only expose raw `git symbolic-ref` output.
- `docs/issues/base-detect-error-not-actionable.md`
  - Move to `docs/issues/done/` during execution after the fix lands.

## Sub-tasks

- [ ] [parallel] Update specs for the expected unresolved-base diagnostic.
- [ ] Implement the `ww create` base-resolution error formatting.
- [ ] Add focused test coverage for the failure message.
- [ ] Run `gofmt` if code changes require it.
- [ ] Run `make test`.
- [ ] Move `docs/issues/base-detect-error-not-actionable.md` to `docs/issues/done/`.

## Verification

- `make test`
- Manual or integration-level check that the failing `ww create` case emits guidance equivalent to:

```text
cannot determine base branch: origin/HEAD is not set and no heuristic base branch was found.
Set default_base in .ww.toml or run: git remote set-head origin --auto
```

The exact wording can differ, but it must be plain language and include both supported remediation paths.
