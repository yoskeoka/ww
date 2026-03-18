# 011: `--repo` Flag for create/remove

> **Execution**: Use `/execute-task` to implement this plan.

**Objective:** Add `--repo` flag to `ww create` and `ww remove` so users can target any repo in the workspace without `cd`-ing into it.

**Covers:** "ww create / ww remove Changes" section of Phase 2 design doc.

**Depends on:** 007 (Workspace Discovery — needed to resolve repo names to paths)

## Context

Phase 1 `ww create` and `ww remove` operate on the current directory's repo. Phase 2 adds `--repo <name>` to target a different repo in the workspace:

```
ww create feat/x --repo ai-arena
ww remove feat/x --repo ai-arena
```

- Value must match a repo name from the workspace's child repo list (directory name)
- Omitted → current directory's repo (Phase 1 compatible)
- Error if `--repo` used outside workspace or repo name not found

## Sub-tasks

- [ ] [parallel] **Spec update `docs/specs/cli-commands.md`**: Update `ww create` and `ww remove` sections:
  - `--repo <name>` flag description
  - Behavior: resolves repo name via workspace discovery, operates on that repo
  - Error cases: outside workspace, unknown repo name
- [ ] [depends on: Spec update] **Update `cmd/ww/sub_create.go`**: Add `--repo` flag:
  - If provided, look up repo path from `Workspace.Repos`
  - Create `git.Runner` and `worktree.Manager` targeting that repo
  - Worktree path uses workspace layout: `<workspace_root>/.worktrees/<repo>@<branch>`
  - If not in workspace mode, return error
- [ ] [depends on: Spec update] **Update `cmd/ww/sub_remove.go`**: Add `--repo` flag with same resolution logic
- [ ] [depends on: Spec update] **Refactor `cmd/ww/main.go` or `cmd/ww/helpers.go`**: Extract repo resolution logic (workspace lookup by name → git.Runner + worktree.Manager) into a shared helper to avoid duplication between create and remove
- [ ] [depends on: sub_create.go, sub_remove.go] **Integration tests**: Using Docker test infra (008), test:
  - `ww create feat/x --repo child-repo` from workspace root: worktree created in child repo
  - `ww remove feat/x --repo child-repo`: worktree removed from child repo
  - `--repo` with unknown name: error message
  - `--repo` outside workspace: error message
  - Omitted `--repo`: same behavior as Phase 1 (backward compatible)
  - Worktree path: verify it lands in `<workspace_root>/.worktrees/<repo>@<branch>`
- [ ] [depends on: Spec update] **Update `docs/spec-code-mapping.md`**: Update create/remove mappings

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/sub_create.go` | Add `--repo` flag, workspace repo resolution |
| `cmd/ww/sub_remove.go` | Add `--repo` flag, workspace repo resolution |
| `cmd/ww/helpers.go` | Extract shared repo resolution helper |
| `integration_test.go` | `--repo` flag tests |

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Update `ww create` and `ww remove` — `--repo` flag |
| `docs/spec-code-mapping.md` | Update mappings |

## Verification

- `make test` passes
- `make lint` passes
- `make test-all` passes
- `ww create feat/x --repo <name>` creates worktree in correct repo
- `ww remove feat/x --repo <name>` removes worktree from correct repo
- Phase 1 behavior unchanged when `--repo` is omitted
- Error messages are clear for invalid repo names and non-workspace contexts
