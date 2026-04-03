# Workspace child-entry prefilter duplicates some `Lstat` work

**Type:** refactor | **Priority:** Low

## Problem

`workspace.scanImmediateRepos()` now does a cheap prefilter before git-based repo validation:

- regular files are skipped immediately
- symlink-like entries are allowed through for explicit rejection

That behavior is intentional and avoids unnecessary `git rev-parse` calls for non-directory entries. However, the current structure still repeats some filesystem work:

- `scanImmediateRepos()` may call `os.Lstat(candidate)` for non-directory entries
- `isImmediateChildRepo()` then calls `os.Lstat(dir)` again and rejects symlinks

This is not a correctness bug, and it is likely cheap in practice, but the control flow is slightly redundant and harder to read than necessary.

**File:** `workspace/workspace.go`

## Current behavior

- Correctly skips regular files before repo validation
- Correctly ignores child symlinks by default
- May perform duplicate `Lstat` work on some entries

## Proposed Solution

If this area is touched again, consider reshaping the helper boundary to use the `os.DirEntry` information directly, for example:

- `isImmediateChildRepo(entry os.DirEntry, dir string)`
- use `entry.IsDir()` / `entry.Type()` for the cheap first pass
- keep any fallback `Lstat` only where the extra metadata is actually needed

This would preserve the current behavior while making the prefilter and symlink handling less repetitive.
