# worktree remove fails when worktree contains submodules

## Summary

`ww remove <branch>` (and `ww remove --force <branch>`) fails when the target worktree contains git submodules.

## Error

```
removing worktree: git worktree remove --force /path/to/worktree: exit status 128
fatal: working trees containing submodules cannot be moved or removed
```

## Root Cause

`git worktree remove --force` has a known limitation: it cannot remove worktrees that contain submodules. This is a git constraint, not a ww bug per se.

`git/git.go: (*git.Runner).WorktreeRemove(path string, force bool)` calls `git worktree remove` and adds `--force` only when `force` is true, but git still rejects the operation when submodules are present.

## Workaround

Manually remove the worktree directory and prune the git worktree list:

> **Warning:** The following command will permanently delete the specified worktree directory, including any uncommitted changes. Double-check `<worktree-path>` and ensure you have no uncommitted work before running it.

```bash
rm -rf <worktree-path>
git worktree prune
```

## Proposed Fix

In `(*worktree.Manager).Remove(branch string, opts RemoveOpts)`, when `git worktree remove` fails due to submodules, either:

1. **Auto-recover**: detect the submodule error, then fall back to `rm -rf <path>` + `git worktree prune`
2. **Guide user**: detect the submodule error and return a descriptive error message with the manual workaround steps

Option 1 is more ergonomic but carries a small risk of deleting unintended changes.
Option 2 is safer and still much better than the raw git error.

## Discovered During

Plan 008 (Docker integration tests) — the `ww@feat-008-docker-integration-tests` worktree contained the `vibe-coding-workspace` submodule.
