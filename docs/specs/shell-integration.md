# Shell Integration Specification

## Overview

Phase 3 shell integration adds explicit path-printing interfaces for shell navigation. The `ww` binary never changes the parent shell's current working directory directly. Instead, it prints absolute paths that shell wrappers or command substitution can consume.

## Contract

- Path-oriented success output goes to `stdout`.
- Human-readable context, warnings, and errors go to `stderr`.
- Successful path output is absolute, undecorated, and newline-terminated.
- JSON output takes precedence over text-mode path-only output when `--json` is set.

## `ww cd`

`ww cd` resolves a worktree path for navigation without mutating shell state.

### `ww cd`

- Resolves the most recently created secondary worktree for the target repository.
- Recency is determined by the `mtime` of directories under `.git/worktrees/` in the target repository's main working tree.
- The main worktree is never returned by no-argument mode because it has no `.git/worktrees/<name>/` admin directory.
- If no secondary worktrees exist, the command exits non-zero and prints `no secondary worktrees found` to `stderr`.

### `ww cd <branch>`

- Resolves the worktree for the specified branch.
- `refs/heads/<branch>` and `<branch>` are treated as the same branch name.
- Performs the normal immediate named lookup first, then retries up to 5 additional times with 100ms intervals when the branch is temporarily not yet discoverable.
- Preserves the existing `no worktree found for branch "<branch>"` error after the bounded retry budget is exhausted.
- Does not apply the retry contract to no-argument recency lookup.
- If no matching worktree exists, the command exits non-zero and prints `no worktree found for branch "<branch>"` to `stderr`.

### `ww cd --json`

- Returns the resolved worktree as one JSON object, using the same fields as `ww list` for that worktree entry.

### `ww cd --repo <name>`

- Reuses the same workspace-repo selection rules as `ww create` and `ww remove`.

## `ww create -q` / `--quiet`

`ww create -q <branch>` suppresses human-readable success output and prints only the created worktree path to `stdout`.

- This is intended for shell composition such as `cd "$(ww create -q feat/x)"`.
- In quiet mode, human-oriented progress is not printed to `stdout`.
- If `post_create_hook` runs in quiet mode, its output is routed to `stderr` so `stdout` remains path-only.
- With `--dry-run`, quiet mode prints the path that would be created.
- With `--json`, JSON output takes precedence over quiet text mode.
- With `--sandbox` in single-repo mode and no explicit `worktree_dir`, quiet mode prints the repo-local `.worktrees` path, e.g. `/path/to/repo/.worktrees/repo@feat-my-branch`.

## Supported Shell Patterns

### Wrapper Function

```sh
wcd() {
  cd "$(ww cd "$@")"
}
```

### Command Substitution

```sh
cd "$(ww create -q feat/my-branch)"
```

## Non-Goals

- `ww` does not directly mutate the parent shell's cwd.
- Shell completion, aliases, and prompt integration are outside this spec.
