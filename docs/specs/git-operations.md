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

## Workspace Member Validation

When validating whether an immediate child directory is a standalone workspace repository, `ww` uses git's resolved paths rather than trusting `.git` markers alone:

```text
git rev-parse --path-format=absolute --show-toplevel
git rev-parse --path-format=absolute --git-dir
git rev-parse --path-format=absolute --git-common-dir
```

A child counts as a standalone repository only when:

- the resolved top-level path matches the child directory
- the resolved git dir equals the resolved git common dir

This excludes linked worktree checkouts while still allowing valid standalone repositories whose git metadata is represented by a `.git` file rather than a `.git` directory.

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

Git refuses to move or remove working trees containing submodules and reports:

```text
fatal: working trees containing submodules cannot be moved or removed
```

`ww` detects this specific `git worktree remove` failure by matching Git's error
text. It does not retry with recursive deletion. Instead, the shared worktree
removal path returns guided remediation telling the user to manually remove the
worktree directory and then run `git worktree prune`, with a warning that manual
directory removal permanently deletes uncommitted work.

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
3. Heuristic fallback when `origin/HEAD` is unavailable:
   - Try `main`, then `master`.
   - If local `<candidate>` tracks `origin/<candidate>`, use `origin/<candidate>`.
   - Otherwise, if any local branch tracks `origin/<candidate>`, use `origin/<candidate>`.
   - Otherwise, if `git ls-remote --heads origin <candidate>` reports the branch exists, use `origin/<candidate>`.
4. If none succeed, base detection fails.

When base detection fails, `ww list` and `ww clean` degrade gracefully: all worktrees are still listed; the main worktree (and branchless/detached entries) remain `active`, while status classification for non-main branch worktrees is skipped and they receive `unknown(base-detect-failed)` instead of `merged`/`stale`. The `ww create` command still requires a resolvable base and returns an error if detection fails.

When heuristic fallback succeeds, `ww list` and `ww clean` use the resolved base exactly like an authoritative base. Status classification still produces normal `active` / `merged` / `stale` values, and `status_detail=heuristic-base` is attached to each listed worktree entry to show that the base came from the heuristic path rather than `default_base` or `origin/HEAD`.

Commands that require a base branch, such as `ww create` for a new branch, must make unresolved base failures actionable. The error must state that no explicit `default_base` is configured, `origin/HEAD` could not be used, and heuristic fallback could not find a usable `origin/main` or `origin/master`. It must also tell the operator to set `default_base` in `.ww.toml` or repair the remote default branch with `git remote set-head origin --auto` when the remote exposes a default branch. The underlying Git failure must remain available in the message or error chain for debugging.

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
