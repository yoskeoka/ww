# Workspace DirEntry Prefilter Follow-up
> **Execution**: Use `/execute-task` to implement this plan.

## Objective

Reduce duplicate `Lstat` work in the child-entry prefilter inside `scanImmediateRepos()` in the `workspace` package, and make the boundary between `DirEntry` prefiltering and symlink/repo validation clearer. Keep public behavior unchanged while improving readability and maintainability.

## Context

- `docs/issues/workspace-direntry-prefilter-followup.md` identifies this as a low-priority refactor: current behavior is correct, but non-directory paths can trigger redundant `Lstat` calls.
- Existing spec expectations (`docs/specs/workspace-discovery.md`) must remain intact: immediate child symlinks are not followed by default, and only real git repositories are treated as workspace members.
- Consistent with ADR direction (2026-03-31, 2026-04-23), workspace detection should stay bounded and conservative.

## Options and Trade-offs

1. **No code change (keep as-is)**
   - Pros: Zero implementation risk and effort.
   - Cons: The issue intent (simplify redundant checks) remains unresolved, and readability cost persists for future edits.
2. **Lightweight refactor passing `os.DirEntry` into `isImmediateChildRepo` (recommended)**
   - Pros: Clarifies the boundary between cheap prefiltering and repo validation, and limits `Lstat` to places that actually need it. Low risk because behavior stays unchanged.
   - Cons: Function signature changes require unit test updates.
3. **Larger refactor to propagate richer file metadata end-to-end**
   - Pros: Could reduce repeated stat work more systematically.
   - Cons: Over-scoped for this low-priority follow-up and increases regression risk across workspace detection logic.

**Recommendation**: Option 2. Refine helper boundaries around `DirEntry` while preserving external behavior, so this issue is resolved without broad logic changes.
**Decision (2026-04-23)**: Human confirmed Option 2; this plan is locked to the lightweight refactor path.

## Spec Changes

- Default direction: **no spec behavior change** (public behavior remains as-is).
- Only if clarification is necessary, add a minimal note in the `Edge Cases` section of `docs/specs/workspace-discovery.md` clarifying that child-entry prefiltering still preserves the no-follow-symlink policy.

## Code Changes

- `workspace/workspace.go`
  - Restructure early filtering in `scanImmediateRepos` around `os.DirEntry`.
  - Adjust `isImmediateChildRepo` input boundaries so `Lstat` is only used when extra metadata is required.
  - Preserve current symlink-ignore behavior, standalone repo validation, and `os.ErrNotExist` handling.

## Test Changes

- `workspace/workspace_test.go`
  - Update or add tests proving non-directory and symlink child handling stays unchanged.
  - Add explicit regression coverage that the workspace member set does not change.

## Sub-tasks

- [ ] Refactor helper boundaries in `workspace/workspace.go` to use `DirEntry` where appropriate and remove duplicate `Lstat` calls
- [ ] [parallel] Update `workspace/workspace_test.go` coverage to confirm no behavior change
- [ ] [depends on: implementation and test updates] Add minimal clarification to `docs/specs/workspace-discovery.md` only if needed
- [ ] [depends on: implementation and test updates] Run `go test ./workspace` for targeted verification

## Verification

- `go test ./workspace`
- Optional if needed: `go test ./...`
- Manual check: create fixtures with regular files, symlink children, and real child repos; confirm detected repo set matches current behavior

## Expected Outcome

- `scanImmediateRepos` flow is easier to read and duplicate `Lstat` work is reduced.
- Public workspace discovery behavior (real repo membership and no-follow child symlink policy) remains unchanged.
