# CLI Commands Specification

## Overview

`ww` is a git worktree manager CLI. It follows a custom subcommand dispatch pattern with `pflag` for POSIX-style flags.

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

List all worktrees for the current repository.

**Flags:** Inherits global flags only.

**Output (text):**
```
PATH                          BRANCH              HEAD
/path/to/repo                 main                abc1234
/path/to/repo@feat-auth       feat/auth           def5678
```

**Output (JSON, NDJSON):**
```
{"path":"/path/to/repo","branch":"main","head":"abc1234","bare":false}
{"path":"/path/to/repo@feat-auth","branch":"feat/auth","head":"def5678","bare":false}
```

### `ww remove <branch>`

Remove the worktree for the given branch and optionally delete the branch.

**Behavior:**
1. Remove the git worktree.
2. Delete the branch (default behavior). The branch is always deleted unless it is the current branch of the main worktree.

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
