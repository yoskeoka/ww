# Git Operations Specification

## Overview

`ww` wraps the `git` CLI for all git operations. All operations are performed by shelling out to `git`, not via a library. This maximizes compatibility with any git version and configuration.

## Prerequisites

`git` must be available in PATH. If not found, `ww` reports a clear error and exits.

## Main Working Tree Resolution

`ww` must always resolve back to the **main working tree** for path computations, regardless of which worktree the user is in. This is achieved via:

```
git rev-parse --path-format=absolute --git-common-dir
```

This returns the shared `.git` directory; its parent is the main working tree root. The repository name is derived from this path.

## Operations

### Worktree Management

**Create worktree with new branch:**
```
git worktree add -b <branch> <path> <base>
```

**Create worktree for existing branch:**
```
git worktree add <path> <branch>
```

**List all worktrees (porcelain format):**
```
git worktree list --porcelain
```

The output is parsed into structured entries. The first entry is always the main working tree and is marked accordingly. Each entry contains: path, HEAD (abbreviated), branch name, bare flag, and main worktree flag.

**Remove worktree:**
```
git worktree remove <path>
```

### Branch Operations

**Delete branch (safe):**
```
git branch -d <branch>
```

Uses `-d` (safe delete) to prevent deleting unmerged branches. If the branch has unmerged work, git refuses and the error is surfaced to the user.

**List merged branches:**
```
git branch --merged <base>
```

Returns the local branch names that are merged into `<base>`.

**Check branch existence:**
```
git rev-parse --verify refs/heads/<branch>
```

**Read branch remote tracking:**
```
git config --get branch.<branch>.remote
```

Returns the configured remote name for a local branch. If no remote is configured, ww treats the branch as having no tracking remote.

**Check remote branch existence:**
```
git ls-remote --heads <remote> <branch>
```

Returns whether a remote branch exists by checking for matching `refs/heads/<branch>` output.

**Detect default branch:**
```
git symbolic-ref refs/remotes/origin/HEAD
```

Extracts the branch name (e.g., `refs/remotes/origin/main` → `origin/main`).

**Default base resolution order:**

1. Explicit `default_base` from config (authoritative).
2. `git symbolic-ref refs/remotes/origin/HEAD` (authoritative).
3. If neither is available, base detection fails.

When base detection fails, `ww list` and `ww clean` degrade gracefully: worktrees are still listed but receive `unknown` status instead of `merged`/`stale`/`active`. The `ww create` command still requires a resolvable base and returns an error if detection fails.

### Other

**Fetch from origin:**
```
git fetch origin
```

## Error Handling

All git errors include:
- The git command that was run
- The stderr output from git

Errors are wrapped with context to make debugging straightforward.

When `git` is not found in PATH, the error message must clearly state that git is required rather than showing a raw exec error.
