# 013: Phase 3 — Shell Integration

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan**: `docs/exec-plan/todo/phase3-polish.md`

**Objective**: Add `ww cd` subcommand and `-q`/`--quiet` flag to `ww create`, enabling shell-oriented worktree navigation without changing existing default output behavior.

## Inherited Rule

All path-oriented output goes to `stdout`. All human-readable context goes to `stderr`. This rule is non-negotiable and inherited from the parent plan's "Shell Integration Design Choice" section.

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Add `ww cd` command spec and `-q`/`--quiet` flag to `ww create` |
| `docs/specs/shell-integration.md` | New spec: path-printing contract, stdout/stderr rules, shell wrapper examples |

### `ww cd` Spec Details

- **`ww cd` (no arguments)**: Print the absolute path of the most recently created worktree.
  - Recency is determined by the `mtime` of `.git/worktrees/<name>/` directories (git creates these at worktree creation and they are not normally modified afterward).
  - Excludes the main worktree (it has no `.git/worktrees/<name>/` entry).
  - If no secondary worktrees exist, exit with a non-zero status and print an error to `stderr`.
- **`ww cd <name>`**: Print the absolute path of the worktree matching `<name>` (branch name match, with or without prefix).
  - If no match, exit non-zero with error to `stderr`.
- **`ww cd --json`**: Print the matching `WorktreeInfo` as JSON (consistent with other subcommands).
- **`ww cd --repo <name>`**: In workspace mode, resolve within the specified repo (reuse `managerForSelectedRepo()` pattern).
- Output: absolute path only on `stdout`, terminated by a newline. No trailing decoration.

### `ww create -q` Spec Details

- **`-q` / `--quiet`**: Suppress human-readable output. Print only the created worktree's absolute path on `stdout`.
- Combined with `--json`: `--json` takes precedence (emit full `WorktreeInfo` JSON, not just path).
- Combined with `--dry-run`: Print the path that would be created (from `WorktreeInfo.Path`).

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Append ADR: "Phase 3 shell integration contract — explicit path-only interfaces, no hidden shell-state mutation, `stderr` for human context, `-q`/`--quiet` over `--path`/`--print-path`" |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/sub_cd.go` | New file: `cd` subcommand implementation |
| `cmd/ww/main.go` | Register `cd` in the `[]command` slice |
| `cmd/ww/sub_create.go` | Add `-q`/`--quiet` flag; when set, print only `info.Path` to `stdout` |
| `worktree/worktree.go` | Add `MostRecent()` method to `Manager` — enumerate `.git/worktrees/` entries, sort by mtime descending, return the top match as `*WorktreeInfo` |
| `worktree/worktree.go` | Add `FindByName(name string)` method — match by branch name (with/without `refs/heads/` prefix) |

### Implementation Notes

- `MostRecent()` reads `.git/worktrees/` directory entries via `os.ReadDir()` on `<repo>/.git/worktrees/`, calls `os.Stat()` for each to get `ModTime()`, sorts descending, then resolves the top entry to a `WorktreeInfo` via the existing `List()` + filter approach (or directly from the porcelain entry's path field in `.git/worktrees/<name>/gitdir`).
- `sub_cd.go` should follow the same patterns as `sub_create.go`: accept `globalOpts`, use `pflag.FlagSet`, support `--json` and `--repo`.
- The `cd` subcommand does NOT need `--dry-run` (it is read-only).

## Sub-tasks

- [ ] [parallel] Add `docs/specs/shell-integration.md` with path-printing contract and shell wrapper examples
- [ ] [parallel] Update `docs/specs/cli-commands.md` with `ww cd` and `ww create -q`/`--quiet`
- [ ] [parallel] Append ADR entry for shell integration contract
- [ ] [depends on: specs] Implement `MostRecent()` and `FindByName()` on `worktree.Manager`
- [ ] [depends on: specs] Implement `ww cd` subcommand in `cmd/ww/sub_cd.go` and register in `main.go`
- [ ] [depends on: specs] Add `-q`/`--quiet` flag to `ww create`
- [ ] [depends on: implementation] Add unit tests for `MostRecent()` and `FindByName()`
- [ ] [depends on: implementation] Add integration tests for `ww cd` (no-arg, named, `--json`, error cases)
- [ ] [depends on: implementation] Add integration tests for `ww create -q`

## Verification

- `ww cd` with no args prints the absolute path of the most recently created worktree
- `ww cd feat/x` prints the path of the worktree for branch `feat/x`
- `ww cd` with no worktrees exits non-zero with error on `stderr`
- `ww create -q feat/x` prints only the path, no human-readable decoration
- `ww create -q --json feat/x` emits full JSON (JSON takes precedence)
- All output follows stdout/stderr separation: paths on `stdout`, messages on `stderr`
- `make test` and `make test-all` pass
- `make lint` passes
