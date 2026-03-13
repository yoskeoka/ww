# CLI Commands Specification

## Overview

`ww` is a git worktree manager CLI. It uses a subcommand dispatch pattern with POSIX-style `--flag` syntax.

## Prerequisites

- `git` must be available in PATH. If not found, `ww` exits with a clear error: `git not found in PATH`.
- `ww` must be run inside a git repository (or a worktree of one). If not, `ww` exits with: `not a git repository`.

When run from a secondary worktree, `ww` resolves back to the main working tree for all path computations. This means all commands work identically regardless of which worktree the user is in.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Output NDJSON (one JSON object per line) |
| `--dry-run` | bool | false | Show planned actions without executing |
| `--version` | bool | false | Print version and exit |

## Commands

### `ww create <branch>`

Create a new worktree for the given branch.

**Behavior:**
1. If the branch does not exist: create a new branch from `default_base` (config) or `origin/HEAD` and add a worktree for it.
2. If the branch already exists: add a worktree that checks out the existing branch.
3. After worktree creation, copy files listed in `copy_files` config.
4. Create symlinks for files listed in `symlink_files` config.
5. Run `post_create_hook` if configured.

**Worktree path:** `<worktree_dir>/<repo>@<branch>` where slashes in branch names are replaced with `-`.

**Flags:** Inherits global flags only.

**Output (text):**
```
Created worktree at /path/to/repo@branch (branch: feat/my-feature)
```

**Output (JSON):**
```json
{"path":"/path/to/repo@branch","branch":"feat/my-feature","created":true,"base":"origin/main"}
```

**Dry-run output (text):**
```
Would create worktree at /path/to/repo@branch (branch: feat/my-feature, base: origin/main)
Would copy: .env, .vscode/settings.json
Would symlink: node_modules
Would run hook: npm install
```

**Exit codes:** 0 on success, 1 on error.

### `ww list`

List all worktrees for the current repository, including the main working tree.

The main working tree (the original repo checkout) is always included and marked with `(main worktree)` in text output or `"main":true` in JSON output. This distinguishes it from secondary worktrees created by `ww create`.

Note: `ww list` shows **worktrees**, not branches. Branches that do not have an associated worktree are not shown. Use `git branch` to see all branches.

**Flags:** Inherits global flags only.

**Output (text):**
```
PATH                                  BRANCH              HEAD
/path/to/repo (main worktree)        main                abc1234
/path/to/repo@feat-auth              feat/auth           def5678
```

**Output (JSON, NDJSON):**
```
{"path":"/path/to/repo","branch":"main","head":"abc1234","main":true}
{"path":"/path/to/repo@feat-auth","branch":"feat/auth","head":"def5678"}
```

### `ww remove <branch>`

Remove the worktree for the given branch and optionally delete the branch.

**Behavior:**
1. Verify the worktree exists. If not, return an error.
2. Remove the git worktree.
3. Delete the branch (default behavior). The branch is always deleted unless it is the current branch of the main worktree.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--keep-branch` | bool | false | Do not delete the branch after removing the worktree |

**Output (text):**
```
Removed worktree at /path/to/repo@branch
Deleted branch feat/my-feature
```

**Output (JSON):**
```json
{"path":"/path/to/repo@branch","branch":"feat/my-feature","removed":true,"branch_deleted":true}
```

**Exit codes:** 0 on success, 1 on error.

### `ww version`

Print the version (commit hash) and exit.

**Output:**
```
ww version <commit-hash>
```
