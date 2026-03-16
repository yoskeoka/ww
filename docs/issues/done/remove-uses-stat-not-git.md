# Remove checks os.Stat instead of git worktree list

**Source**: PR #3 review
**File**: `worktree/worktree.go:221`
**Severity**: Medium

## Description

The `Remove` command checks if the computed worktree path exists on disk via `os.Stat`. If the worktree directory was manually deleted but is still registered in git's worktree list, `ww remove` will fail with "no worktree found" instead of cleaning up the stale registration.

## Proposed Solution

Check against `git worktree list` output instead of (or in addition to) filesystem existence. This would also allow `ww remove` to clean up stale worktree entries.
