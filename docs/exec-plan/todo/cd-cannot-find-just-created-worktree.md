# Fix `ww cd` Immediate Lookup After `ww create`
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Investigate and fix the reported dogfooding failure where `ww create <branch>` succeeds from the workspace root, but `ww cd <branch>` from the same cwd immediately fails to resolve the just-created worktree.

This plan supports `ww`'s shell-integration goal in `docs/project-plan.md` and the workspace dogfooding contract that `ww create` and `ww cd` must compose reliably during normal task startup.

## Background

Issue reference: `docs/issues/ww-cd-cannot-find-just-created-worktree.md`

External workspace context: originally reported in `vibe-coding-workspace/docs/issues/ww-cd-cannot-find-just-created-worktree.md`

Reported reproduction in the real workspace:

- cwd: workspace root `vibe-coding-workspace/`
- `ww create feat/kb-bilingual-rendering` succeeds
- `ww cd feat/kb-bilingual-rendering` fails with `no worktree found for branch "feat/kb-bilingual-rendering"`

Relevant current spec expectations already require parity between `ww create` and `ww cd` repo resolution:

- `docs/specs/cli-commands.md`
- `docs/specs/workspace-discovery.md`

Planning note:

- A minimal synthetic Git-backed workspace reproduced the expected success path on installed `ww v0.4.1`.
- That suggests the failure is likely tied to a real-workspace condition not covered by current integration tests, not a universally broken `ww cd` path.

## Expected Outcome

- `ww cd <branch>` can immediately resolve a branch just created by `ww create <branch>` from the same workspace-root cwd when both commands target the same repository implicitly.
- The real failing condition is captured by automated coverage so the regression does not return silently.
- If the investigation shows the current spec is ambiguous rather than the code being wrong, the spec is tightened before implementation.

## Code Changes

- Add or extend regression coverage around workspace-root create/cd parity in:
  - `integration_test.go`
- Investigate and fix the relevant repo-resolution or worktree-selection path in:
  - `cmd/ww/sub_cd.go`
  - `cmd/ww/helpers.go`
  - `cmd/ww/main.go`
  - `workspace/workspace.go`
  - `worktree/worktree.go`
- Limit implementation changes to the concrete mismatch exposed by the reproduction. Do not broaden workspace detection or refactor unrelated command plumbing unless the failing case proves they are the direct cause.

## Spec Changes

- Re-read and confirm the intended contract in:
  - `docs/specs/cli-commands.md`
  - `docs/specs/workspace-discovery.md`
- If the current wording does not explicitly cover Git-backed workspace roots that are also managed repos, add the smallest clarification needed so the behavior under test is unambiguous.
- If the wording is already sufficient, leave the specs unchanged and treat this as a pure implementation/test gap.

## Design Decisions

- No new ADR is expected unless fixing this issue requires changing the repo-selection model for workspace-root Git repositories.
- Prefer preserving the existing public contract that `ww create` and `ww cd` share the same implicit target-repo behavior when `--repo` is omitted.

## Reproduction / Investigation

- Compare the real failing workspace with the minimal passing synthetic workspace and identify the missing condition.
- Check for differences in:
  - Git-backed workspace root inclusion in detected workspace repos
  - main-worktree resolution when the workspace root is itself a repo
  - branch-name normalization between `create` and `cd`
  - worktree enumeration behavior when secondary worktrees live under the centralized workspace `.worktrees/` directory
  - config-loading side effects that change repo/workspace mode between the two commands

## Sub-tasks

- [ ] Reproduce the failure in an automated test or isolate the exact workspace-specific precondition that triggers it
- [ ] [depends on: reproduced failing condition] Confirm whether the current spec already defines the expected behavior or needs a narrow clarification
- [ ] [depends on: reproduced failing condition] Fix the concrete repo-resolution or worktree-lookup mismatch causing `ww cd` to miss the new worktree
- [ ] [depends on: implementation] Add regression coverage for both:
  - implicit repo targeting from a Git-backed workspace root
  - the previously passing `--repo`-explicit path, to avoid regressing disambiguation behavior
- [ ] [depends on: implementation] Verify `ww create`, `ww cd <branch>`, and `ww cd` (no args) still behave correctly from:
  - a normal single repo
  - a non-git workspace root with `--repo`
  - a Git-backed workspace root without `--repo`

## Verification

- `go test ./...`
- Focused integration coverage for the reproduced workspace-root scenario
- Manual smoke check in a temporary workspace that matches the failing topology closely enough to prove the fix, if the automated fixture cannot fully mirror the original workspace

## Risks

- The original report may depend on workspace state that is hard to mirror exactly; if so, the implementation must not guess. Capture the missing condition first and only then patch behavior.
- Workspace-root Git repositories are a special case in detection because the root can be both a repo and a workspace container. A careless fix could break `--repo` disambiguation or single-repo fallback behavior.
