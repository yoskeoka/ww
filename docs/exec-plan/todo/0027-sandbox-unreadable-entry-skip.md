# 0027: Sandbox Create Should Skip Unreadable Non-Repo Entries
> **Execution**: Use `/execute-task` to implement this plan.

## Objective

Prevent `ww --sandbox create <branch>` from aborting when sandboxed filesystem rules deny metadata reads on irrelevant non-repository entries such as a gitignored `.env` file in the repo root. Preserve the existing sandbox discovery contract and workspace detection model while narrowing the failure surface to entries that are actually needed for repository detection.

Addresses: https://github.com/yoskeoka/ww/issues/211

## Context

- `docs/project-plan.md` already treats sandbox support as a bounded-discovery feature, not a broad policy engine. The sandbox contract should avoid unnecessary parent/sibling access without changing explicit user intent or normal git behavior.
- `docs/design-decisions/adr.md` records that sandbox mode keeps current-directory child repo scanning so `--repo` still works from a workspace root. Any fix should preserve that behavior unless there is a stronger reason to redefine the mode.
- The current failure is reproducible on macOS inside Claude Code sandbox mode: plain `git worktree add` succeeds, but `ww --sandbox create` fails on `lstat <repo>/.env: operation not permitted`.
- Current `main` shows a likely cause in `workspace.DetectWithOptions()`: sandbox mode always runs `scanImmediateRepos(absStart)` first, and `classifyImmediateChild()` may call `os.Lstat()` on immediate children whose `DirEntry` type is unknown. That makes unreadable non-repo files part of command setup even though they are irrelevant to worktree creation.

## Options and Trade-offs

1. **Redefine sandbox detection order to resolve the current repo before scanning children**
   - Pros: Minimizes child-entry touching when the command starts inside a repository.
   - Cons: Risks changing the established zero-config workspace-root behavior for repo roots that also contain child repos. Broader than the issue requires and may need ADR/spec rewrites.
2. **Keep the detection model, but make immediate-child scanning tolerant of unreadable non-repo entries (recommended)**
   - Pros: Fixes the concrete failure without changing the workspace-vs-single-repo contract. Matches the existing direction to keep detection bounded and conservative. Limits behavior change to entries that are not proven child repos.
   - Cons: `ww` may still attempt a metadata read on some unknown-type entries before deciding to skip them. A deterministic test seam may be needed because ordinary Unix permissions do not reproduce sandbox `EPERM` reliably.
3. **Add a new flag/config to disable discovery-side file touching in sandbox mode**
   - Pros: Gives explicit operator control.
   - Cons: Adds surface area and user burden for what should be a safe default. Does not address the unnecessary access in the default path.

**Recommendation**: Option 2. Treat unreadable immediate children that are not needed for repo membership as ignorable during workspace-child scanning, while preserving hard failures for errors that affect actual repo resolution or explicit git operations.
**Decision (2026-05-23)**: Human confirmed Option 2. Keep the existing sandbox detection model and fix the unnecessary child-entry access/error handling path narrowly.

## Spec Changes

- Update `docs/specs/workspace-discovery.md`:
  - Clarify that immediate-child scanning is best-effort for non-repository entries.
  - State that unreadable immediate children which cannot be established as real child repos are skipped instead of aborting detection.
  - Preserve the existing rules that child symlinks are ignored and only real child repo roots count as workspace members.
- Update `docs/specs/cli-commands.md` only if needed to make the `ww create --sandbox` expectation explicit: unreadable gitignored/non-repo files encountered during detection must not block create.

## Code Changes

- `workspace/workspace.go`
  - Narrow `scanImmediateRepos()` / `isImmediateChildRepo()` / `classifyImmediateChild()` so unreadable immediate children that are not established as repos do not abort detection.
  - Keep current behavior for real directory child repos, child symlink exclusion, and linked-worktree exclusion.
  - Prefer the smallest change that preserves the current detection order and candidate-window logic.
- If the cleanest implementation requires a small helper seam for testing sandbox-style `Lstat` failures, add that seam locally within the `workspace` package rather than broadening public APIs.

## Test Changes

- `workspace/workspace_test.go`
  - Add regression coverage for unreadable-or-erroring non-repo child handling during `scanImmediateRepos()` / sandbox detection.
  - Confirm that readable real child repos are still discovered and that unreadable non-repo siblings do not change the detected repo set.
- Add a narrow unit test seam if needed to simulate `os.Lstat(...)=EPERM` for an immediate child with unknown `DirEntry` type.
- Optional integration/manual verification note:
  - Re-run the reported macOS Claude Code sandbox reproduction after implementation because that environment cannot be reproduced deterministically in the current automated test suite.

## Design Decisions

- No ADR update if the fix stays within the existing sandbox detection contract.
- If implementation pressure pushes toward changing sandbox detection order or workspace-root semantics, stop and add an ADR update instead of silently broadening scope.

## Sub-tasks

- [ ] Update `docs/specs/workspace-discovery.md` with the unreadable-non-repo skip contract
- [ ] [parallel] Decide whether `docs/specs/cli-commands.md` needs a matching create-specific clarification
- [ ] [depends on: spec update] Implement the narrow workspace detection fix in `workspace/workspace.go`
- [ ] [depends on: implementation] Add regression coverage in `workspace/workspace_test.go`, including any minimal helper seam needed for deterministic permission-error simulation
- [ ] [depends on: implementation and tests] Run targeted verification plus a documented manual sandbox re-check

## Verification

- `go test ./workspace`
- `go test ./cmd/ww ./workspace ./worktree`
- Manual reproduction in the reported environment:
  - macOS
  - Claude Code sandbox mode denying `.env`
  - repo root containing gitignored `.env`
  - confirm `ww --sandbox create <branch>` no longer aborts before `git worktree add`

## Expected Outcome

- `ww --sandbox create` no longer fails just because the current directory contains unreadable gitignored/non-repo files such as `.env`.
- Sandbox mode keeps its current bounded-discovery semantics, including workspace-root support from the current directory.
- The fix stays narrowly scoped to unnecessary child-entry access instead of expanding into a broader sandbox-mode redesign.
