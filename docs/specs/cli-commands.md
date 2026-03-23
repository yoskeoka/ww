# CLI Commands Specification

## Overview

`ww` is a git worktree manager CLI. It uses a subcommand dispatch pattern with POSIX-style `--flag` syntax.

## Prerequisites

- `git` must be available in PATH. If not found, `ww` exits with a clear error: `git not found in PATH`.
- `ww` may be started from a non-git workspace root, but commands that need a current repo still require repo selection. `ww list` is the exception and works from a detected workspace root. Until repo selection exists, other commands that need a current repo exit with: `repo selection is not supported from a non-git workspace root`.
- If the current directory is neither a git repository nor a detected workspace root, `ww` exits with: `not a git repository`.

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
1. If a worktree directory already exists at the target path, return an error: `worktree already exists at <path>`.
2. If the branch does not exist: create a new branch from `default_base` (config) or `origin/HEAD` and add a worktree for it.
3. If the branch already exists: add a worktree that checks out the existing branch.
4. After worktree creation, copy files listed in `copy_files` config.
5. Create symlinks for files listed in `symlink_files` config.
6. Run `post_create_hook` if configured.

**Worktree path:** mode-dependent default, or explicit `worktree_dir` override. Slashes in branch names are replaced with `-`.

**Flags:** Inherits global flags only.

**Output (text):**
```text
Created worktree at /path/to/repo@branch (branch: feat/my-feature)
```

**Output (JSON):**
```json
{"path":"/path/to/repo@branch","branch":"feat/my-feature","created":true,"base":"origin/main"}
```

**Dry-run output (text):**
```text
Would create worktree at /path/to/repo@branch (branch: feat/my-feature, base: origin/main)
Would copy: .env, .vscode/settings.json
Would symlink: node_modules
Would run hook: npm install
```

**Exit codes:** 0 on success, 1 on error.

### `ww list`

List all worktrees for the current repository, including the main working tree.

The main working tree (the original repo checkout) is always included and marked with `(main worktree)` in text output or `"main":true` in JSON output. This distinguishes it from secondary worktrees created by `ww create`.

In workspace mode, `ww list` includes worktrees from every detected repository. When run from a non-git workspace root, it still lists the workspace repositories.

Note: `ww list` shows **worktrees**, not branches. Branches that do not have an associated worktree are not shown. Use `git branch` to see all branches.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--cleanable` | bool | false | Show only worktrees with `merged` or `stale` status |

**Status values:**
| Status | Meaning |
|--------|---------|
| `active` | Main worktree, or a branch that is neither merged nor stale |
| `merged` | Branch is present in `git branch --merged <base>` |
| `stale` | Branch has tracking configured, the remote branch no longer exists, and it is not merged |

**Output (text):**
```text
PATH                                  BRANCH              HEAD     STATUS
/path/to/repo (main worktree)        main                abc1234  active
/path/to/repo@feat-auth              feat/auth           def5678  merged
```

**Output (workspace mode text):**
```text
REPO  PATH                                  BRANCH              HEAD     STATUS
ww    /path/to/workspace/.worktrees/ww@feat  feat                abc1234  active
ai    /path/to/workspace/.worktrees/ai@done  done                def5678  stale
```

**Output (JSON, NDJSON):**
```json
{"repo":"repo","path":"/path/to/repo","branch":"main","head":"abc1234","main":true,"status":"active"}
{"repo":"repo","path":"/path/to/repo@feat-auth","branch":"feat/auth","head":"def5678","status":"merged"}
```

### `ww remove <branch>`

Remove the worktree for the given branch and optionally delete the branch.

**Behavior:**
1. Look up the branch in `git worktree list` output. If no worktree entry exists for the branch, return an error: `no worktree found for branch "<branch>"`.
2. If the matching entry is the main worktree (`Main == true`), reject with error: `cannot remove the main worktree`.
3. Remove the git worktree using the path from the worktree list entry.
4. Attempt to delete the branch (default behavior) using a safe delete (`git branch -d`). If deletion fails (for example, because the branch is not fully merged or is the current branch of the main worktree), print a warning and continue; in this case, the branch is not deleted.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | false | Force removal even if the worktree is dirty (passes `--force` to `git worktree remove`) |
| `--keep-branch` | bool | false | Do not delete the branch after removing the worktree |

**Output (text):**
```text
Removed worktree at /path/to/repo@branch
Deleted branch feat/my-feature
```

**Output (JSON):**
```json
{"path":"/path/to/repo@branch","branch":"feat/my-feature","removed":true,"branch_deleted":true}
```

If safe branch deletion fails, JSON output includes `"branch_deleted":false` and a `branch_error` field with the git error message.

**Exit codes:** 0 on success, 1 on error.

### `ww clean`

Remove all cleanable worktrees for the current repository or detected workspace.

Cleanable worktrees are those whose `STATUS` is `merged` or `stale` in `ww list`.
Main worktrees and `active` worktrees are never removed by this command.

In workspace mode, `ww clean` operates across all detected repositories. When run
from a non-git workspace root, it still cleans the workspace repositories.

**Behavior:**
1. List worktrees and determine their status using the same rules as `ww list`.
2. Filter to worktrees with status `merged` or `stale`.
3. For each cleanable worktree, remove the git worktree and delete the local branch.
4. If safe branch deletion fails for a worktree, print a warning in text mode or include `branch_error` in JSON output, then continue. This does not make the command fail by itself.
5. If one worktree fails to remove, report that failure, continue processing the remaining cleanable worktrees, and exit non-zero after all attempts complete.
6. If there are no cleanable worktrees, exit successfully with no output.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | false | Force removal even if the worktree is dirty and force-delete the branch (`git worktree remove --force` + `git branch -D`) |

**Output (text):**
```text
Removed worktree at /path/to/repo@feat-old
Deleted branch feat/old
Removed worktree at /path/to/repo@feat-stale
Deleted branch feat/stale
```

**Dry-run output (text):**
```text
Would remove worktree at /path/to/repo@feat-old
Would delete branch feat/old
Would remove worktree at /path/to/repo@feat-stale
Would delete branch feat/stale
```

**Output (JSON, NDJSON):**
```json
{"repo":"repo","path":"/path/to/repo@feat-old","branch":"feat/old","status":"merged","removed":true,"branch_deleted":true}
{"repo":"repo","path":"/path/to/repo@feat-stale","branch":"feat/stale","status":"stale","removed":true,"branch_deleted":false,"branch_error":"git branch -d feat/stale: ..."}
```

**Failure output (JSON, NDJSON):**
```json
{"repo":"repo","path":"/path/to/repo@feat-dirty","branch":"feat/dirty","status":"stale","removed":false,"branch_deleted":false,"error":"removing worktree: ..."}
```

**Exit codes:** 0 when all cleanable worktrees are processed successfully or none exist, 1 if any cleanable worktree fails.

### `ww version`

Print the version (commit hash) and exit.

**Output:**
```text
ww version <commit-hash>
```
