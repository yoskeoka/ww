# CLI Commands Specification

## Overview

`ww` is a git worktree manager CLI. It uses a subcommand dispatch pattern with POSIX-style `--flag` syntax.

## Prerequisites

- `git` must be available in PATH. If not found, `ww` exits with a clear error: `git not found in PATH`.
- Workspace-sensitive commands use the nearest containing workspace root selected by the workspace discovery algorithm.
- Detected workspace repositories are limited to real child repo roots. Immediate child symlinks, linked worktree checkouts, and helper directories with stray `.git` markers are excluded from workspace membership.
- `ww` may be started from a non-git workspace root. `ww list` and `ww clean` work there without extra flags. `ww create` and `ww remove` require `--repo <name>` from that location; without it they exit with: `repo selection is not supported from a non-git workspace root`.
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
1. Resolve the target repository:
   - If `--repo <name>` is omitted, use the current repository exactly as in Phase 1.
   - If `--repo <name>` is provided, require detected workspace mode, find the matching entry in the workspace child repo list by directory name, and operate on that repository.
   - If `--repo` is provided outside workspace mode, return an error: `--repo can only be used inside a detected workspace`.
   - If `--repo` names no detected repository, return an error: `repo "<name>" not found in workspace`.
2. If a worktree directory already exists at the target path, return an error: `worktree already exists at <path>`.
3. If the branch does not exist: create a new branch from `default_base` (config) or `origin/HEAD` and add a worktree for it.
4. If the branch already exists: add a worktree that checks out the existing branch.
5. After worktree creation, copy files listed in `copy_files` config.
6. Create symlinks for files listed in `symlink_files` config.
7. Run `post_create_hook` if configured. In text mode, print `Running post_create_hook: <command>` immediately before streaming the hook's own output.

**Worktree path:** mode-dependent default, or explicit `worktree_dir` override. Slashes in branch names are replaced with `-`.
In workspace mode with `--repo`, the default path remains centralized at `<workspace_root>/.worktrees/<repo>@<branch>`.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--repo` | string | empty | Target a detected workspace repository by name instead of the current repo |
| `-q`, `--quiet` | bool | false | Suppress human-readable output and print only the created worktree path on `stdout` |
| `--dry-run` | bool | false | Show planned actions without executing |
| `--json` | bool | false | Output NDJSON (one JSON object per line) |

**Output (text):**
```text
Running post_create_hook: npm install
Created worktree at /path/to/repo@branch (branch: feat/my-feature)
```

**Output (quiet text):**
```text
/path/to/repo@branch
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

**Dry-run output (quiet text):**
```text
/path/to/repo@branch
```

**Exit codes:** 0 on success, 1 on error.

### `ww cd [branch]`

Print the absolute path of a worktree for shell navigation.

**Behavior:**
1. Resolve the target repository:
   - If `--repo <name>` is omitted, use the current repository exactly as in Phase 1.
   - If `--repo <name>` is provided, require detected workspace mode, find the matching entry in the workspace child repo list by directory name, and operate on that repository.
   - If `--repo` is provided outside workspace mode, return an error: `--repo can only be used inside a detected workspace`.
   - If `--repo` names no detected repository, return an error: `repo "<name>" not found in workspace`.
2. If no branch argument is provided, resolve the most recently created secondary worktree for the target repository.
3. If a branch argument is provided, match it against worktree branch names. `refs/heads/<branch>` and `<branch>` are treated as equivalent.
4. On success in text mode, print only the absolute path to `stdout`, terminated by a newline.
5. If no matching secondary worktree exists, return an error:
   - no-argument mode: `no secondary worktrees found`
   - named mode: `no worktree found for branch "<branch>"`

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--repo` | string | empty | Target a detected workspace repository by name instead of the current repo |
| `--json` | bool | false | Output a single JSON object describing the resolved worktree |

**Output (text):**
```text
/path/to/repo@feat-my-feature
```

**Output (JSON):**
```json
{"repo":"repo","path":"/path/to/repo@feat-my-feature","branch":"feat/my-feature","head":"abc1234","status":"active"}
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
| `unknown` | Base branch could not be determined; status classification was skipped |

When the base branch is resolved heuristically (for example, by inferring `origin/main` after `origin/HEAD` lookup fails), `ww list` still emits normal `active` / `merged` / `stale` statuses. In text output, those render as `active(heuristic-base)`, `merged(heuristic-base)`, or `stale(heuristic-base)`. In JSON output, the `status` value remains `active`, `merged`, or `stale`, and `status_detail=heuristic-base` is emitted separately for every listed worktree entry.

When the base branch cannot be resolved at all (no `default_base` config, `origin/HEAD` detection fails, and heuristic fallback fails), all non-main worktrees that have an associated branch receive `unknown` status with a `status_detail` field indicating the reason (e.g., `base-detect-failed`). Worktrees without an associated branch (for example, detached HEAD entries where `branch` is absent from `git worktree list --porcelain`) remain `active`.

In text output, any non-empty `status_detail` renders as `<status>(<detail>)`. In JSON output, `status` and `status_detail` are emitted as separate fields.

`--cleanable` and `ww clean` only act on `merged` and `stale` worktrees. `unknown` worktrees are never eligible for cleanup.

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

**Output when base branch is unknown (text):**
```text
PATH                                  BRANCH              HEAD     STATUS
/path/to/repo (main worktree)        main                abc1234  active
/path/to/repo@feat-auth              feat/auth           def5678  unknown(base-detect-failed)
```

**Output when base branch is unknown (JSON):**
```json
{"repo":"repo","path":"/path/to/repo","branch":"main","head":"abc1234","main":true,"status":"active"}
{"repo":"repo","path":"/path/to/repo@feat-auth","branch":"feat/auth","head":"def5678","status":"unknown","status_detail":"base-detect-failed"}
```

**Output when base branch is resolved heuristically (JSON):**
```json
{"repo":"repo","path":"/path/to/repo","branch":"main","head":"abc1234","main":true,"status":"active","status_detail":"heuristic-base"}
{"repo":"repo","path":"/path/to/repo@feat-auth","branch":"feat/auth","head":"def5678","status":"merged","status_detail":"heuristic-base"}
```

**Output when base branch is resolved heuristically (text):**
```text
PATH                                  BRANCH              HEAD     STATUS
/path/to/repo (main worktree)        main                abc1234  active(heuristic-base)
/path/to/repo@feat-auth              feat/auth           def5678  merged(heuristic-base)
```

### `ww remove <branch>`

Remove the worktree for the given branch and optionally delete the branch.

**Behavior:**
1. Resolve the target repository:
   - If `--repo <name>` is omitted, use the current repository exactly as in Phase 1.
   - If `--repo <name>` is provided, require detected workspace mode, find the matching entry in the workspace child repo list by directory name, and operate on that repository.
   - If `--repo` is provided outside workspace mode, return an error: `--repo can only be used inside a detected workspace`.
   - If `--repo` names no detected repository, return an error: `repo "<name>" not found in workspace`.
2. Look up the branch in `git worktree list` output. If no worktree entry exists for the branch, return an error: `no worktree found for branch "<branch>"`.
3. If the matching entry is the main worktree (`Main == true`), reject with error: `cannot remove the main worktree`.
4. Remove the git worktree using the path from the worktree list entry.
5. Attempt to delete the branch unless `--keep-branch` is set. By default this uses a safe delete (`git branch -d`). When `--force` is set, it uses a force delete (`git branch -D`) to match the forced worktree removal behavior.
6. If safe branch deletion fails (for example, because the branch is not fully merged or is the current branch of the main worktree), print a warning and continue; in this case, the branch is not deleted.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--repo` | string | empty | Target a detected workspace repository by name instead of the current repo |
| `--force` | bool | false | Force removal even if the worktree is dirty and force-delete the branch (`git worktree remove --force` + `git branch -D`) |
| `--keep-branch` | bool | false | Do not delete the branch after removing the worktree |
| `--dry-run` | bool | false | Show planned actions without executing |
| `--json` | bool | false | Output NDJSON (one JSON object per line) |

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

When `--force` is used and branch deletion succeeds, JSON output still reports `"branch_deleted":true`; there is no separate field indicating whether safe or forced deletion was used.

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
| `--dry-run` | bool | false | Show planned actions without executing |
| `--json` | bool | false | Output NDJSON (one JSON object per line) |

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

Print version information and exit.

Tagged release builds print the SemVer tag. Untagged builds print a dev identifier plus the short commit hash.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Output JSON |

**Output (text, tagged release build):**
```text
ww version v0.3.0
```

**Output (text, dev build):**
```text
ww version dev+abc1234
```

**Output (JSON, tagged release build):**
```json
{"version":"v0.3.0","commit":"abc1234"}
```

**Output (JSON, dev build):**
```json
{"version":"dev","commit":"abc1234"}
```
