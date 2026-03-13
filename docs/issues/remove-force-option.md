# No --force option for ww remove

**Source**: PR #3 review
**File**: `cmd/ww/sub_remove.go`, `worktree/worktree.go`
**Severity**: Medium

## Description

`ww remove` uses `git worktree remove` without `--force`, which fails on dirty worktrees. Users with uncommitted changes in a worktree have no way to force removal through `ww`.

## Proposed Solution

Add a `--force` flag to `ww remove` that passes `--force` to `git worktree remove`. Document the behavior clearly — force removal discards uncommitted changes.
