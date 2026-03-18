# 009: `ww list` Enhancements — REPO, STATUS, `--cleanable`

**Objective:** Extend `ww list` to show worktree health status and repository context in workspace mode. Add filtering for cleanable worktrees.

**Covers:** "ww list Changes" section of Phase 2 design doc.

**Depends on:** 007 (Workspace Discovery), 008 (Docker test infra)

## Context

Phase 1 `ww list` shows PATH, BRANCH, HEAD columns for a single repo. Phase 2 adds:
- **REPO column**: Shows which repo each worktree belongs to (workspace mode only, omitted in single-repo mode)
- **STATUS column**: `active`, `merged`, or `stale` per worktree
- **`--cleanable` flag**: Filters to only `merged` and `stale` worktrees

STATUS determination:
| Status | Condition |
|--------|-----------|
| `active` | Neither merged nor stale |
| `merged` | Branch is in `git branch --merged <base>` |
| `stale` | Remote tracking configured + remote branch gone + not merged |

- Main worktrees always show `active`
- `merged` takes precedence if both conditions met
- Base branch: `default_base` from config, or `origin/HEAD` auto-detect
- No remote tracking → not stale (just `active`)

## Sub-tasks

- [ ] [parallel] **Spec update `docs/specs/cli-commands.md`**: Update `ww list` section:
  - REPO column (workspace mode only)
  - STATUS column with determination rules
  - `--cleanable` flag behavior
  - Updated JSON output schema (`"repo"`, `"status"` fields)
- [ ] [parallel] **Add git operations to `git/git.go`**:
  - `MergedBranches(base string) ([]string, error)` — wraps `git branch --merged <base>`
  - `BranchRemote(branch string) (string, error)` — wraps `git config branch.<name>.remote`
  - `RemoteBranchExists(remote, branch string) (bool, error)` — wraps `git ls-remote --heads <remote> <branch>`
  - Unit tests for each
- [ ] [depends on: git operations] **Add status determination to `worktree/`**: Create a function/method that takes a `WorktreeEntry` and returns its status (`active`/`merged`/`stale`). Logic:
  1. Main worktree → `active`
  2. Branch in merged list → `merged`
  3. Has remote tracking + remote branch gone → `stale`
  4. Otherwise → `active`
- [ ] [depends on: status determination] **Update `worktree.Manager.List()`**: Return `WorktreeInfo` with new `Repo` and `Status` fields. In workspace mode, iterate over all child repos.
- [ ] [depends on: Update List] **Update `cmd/ww/sub_list.go`**: Add REPO and STATUS columns to table output. Add `--cleanable` flag. Update JSON output.
- [ ] [depends on: Spec update] **Update `docs/spec-code-mapping.md`**: Add/update rows for new git operations and status logic.
- [ ] [depends on: Update sub_list.go] **Integration tests**: Using Docker test infra (008), test:
  - Workspace mode: REPO column present, multiple repos listed
  - Single-repo mode: REPO column absent (backward compatible)
  - STATUS: create merged branch → shows `merged`; push then delete remote branch → shows `stale`; normal branch → shows `active`
  - `--cleanable`: filters correctly
  - `--json`: includes `repo` and `status` fields

## Code Changes

| File | Change |
|------|--------|
| `git/git.go` | Add `MergedBranches()`, `BranchRemote()`, `RemoteBranchExists()` |
| `git/git_test.go` | Unit tests for new git operations |
| `worktree/worktree.go` | Add status determination logic, update `List()` for workspace mode |
| `worktree/worktree_test.go` | Unit tests for status determination |
| `cmd/ww/sub_list.go` | REPO column, STATUS column, `--cleanable` flag |
| `integration_test.go` | Workspace-mode list tests |

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Update `ww list` — REPO, STATUS, `--cleanable`, JSON schema |
| `docs/specs/git-operations.md` | Add `branch --merged`, `config branch.<name>.remote`, `ls-remote --heads` |
| `docs/spec-code-mapping.md` | Update mappings |

## Verification

- `make test` passes
- `make lint` passes
- `make test-docker` passes with workspace-mode list tests
- Single-repo `ww list` output unchanged (backward compatible)
- `ww list --json` includes `repo` and `status` fields
- `ww list --cleanable` shows only merged/stale worktrees
