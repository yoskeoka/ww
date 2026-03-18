# 010: `ww clean` New Command

**Objective:** Implement `ww clean` to bulk-remove merged and stale worktrees across the workspace.

**Covers:** "ww clean (New Command)" section of Phase 2 design doc.

**Depends on:** 009 (`ww list` STATUS determination logic is reused)

## Context

`ww clean` is the complement to `ww list --cleanable`. It removes all worktrees whose status is `merged` or `stale`, performing both `git worktree remove` and `git branch -d` for each.

| Flag | Behavior |
|------|----------|
| (none) | Safe delete: `git worktree remove` + `git branch -d` |
| `--dry-run` | Show what would be deleted, do not execute |
| `--force` | Force delete: `git worktree remove --force` + `git branch -D` |
| `--json` | Output results as JSON |

No confirmation prompt — `ww clean` is explicit intent. Users preview with `ww list --cleanable` or `ww clean --dry-run`.

## Sub-tasks

- [ ] [parallel] **Spec update `docs/specs/cli-commands.md`**: Add `ww clean` section:
  - Description, flags (`--dry-run`, `--force`, `--json`)
  - Behavior: removes all `merged` + `stale` worktrees
  - Output format (table and JSON)
  - Error handling: partial failures (some worktrees fail to remove) — report per-worktree, continue with remaining
- [ ] [depends on: Spec update] **Implement `cmd/ww/sub_clean.go`**: New subcommand file:
  - Reuse status determination from `worktree/` (plan 009)
  - Iterate cleanable worktrees, call `worktree.Manager.Remove()` for each
  - Collect results, output table or JSON
  - `--dry-run`: list what would be removed without executing
  - `--force`: pass force flag to Remove
- [ ] [depends on: Spec update] **Register `clean` subcommand in `cmd/ww/main.go`**: Add to command list
- [ ] [depends on: sub_clean.go] **Integration tests**: Using Docker test infra (008), test:
  - Setup: create worktrees, merge some branches, delete some remote branches
  - `ww clean --dry-run`: lists cleanable worktrees, nothing removed
  - `ww clean`: removes merged/stale worktrees, active worktrees untouched
  - `ww clean --force`: force-removes worktrees with uncommitted changes
  - `ww clean --json`: structured output
  - Workspace mode: cleans across all repos
  - Empty case: no cleanable worktrees → clean exit, no output
- [ ] [depends on: Spec update] **Update `docs/spec-code-mapping.md`**: Add `ww clean` mapping

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/sub_clean.go` | New — `ww clean` subcommand |
| `cmd/ww/main.go` | Register `clean` subcommand |
| `integration_test.go` | Clean command tests |

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Add `ww clean` section |
| `docs/spec-code-mapping.md` | Add `ww clean` mapping |

## Design Notes

- `ww clean` reuses `worktree.Manager.Remove()` — no new removal logic needed.
- STATUS determination is shared with `ww list` (implemented in 009). No duplication.
- Partial failure: if one worktree fails to remove, report the error and continue. Exit code non-zero if any failures occurred.

## Verification

- `make test` passes
- `make lint` passes
- `make test-docker` passes
- `ww clean --dry-run` shows correct preview
- `ww clean` removes only merged/stale worktrees
- Active worktrees are never removed by `ww clean`
