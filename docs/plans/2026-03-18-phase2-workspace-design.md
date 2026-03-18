# Phase 2 Design: Workspace Discovery & Cross-Repo Operations

## Overview

Phase 2 adds workspace awareness to ww. The primary goals are cross-repo visibility (list worktrees across all repos in a workspace), worktree health status, and bulk cleanup of stale worktrees.

Phase 2 does NOT add multi-repo batch create/remove (FR-19) or shell navigation (FR-20). Those are future scope.

## Workspace Discovery

### Detection Algorithm

0. Scan current directory's immediate children for `.git` entries (files or directories). If found → current directory becomes a **parent candidate**, children are recorded as child repos.
1. Determine current git repo root (if inside a git repo) using `git rev-parse --is-inside-work-tree` / `git rev-parse --show-toplevel`.
2. Look at the parent directory.
3. If parent has a `.git` entry (file or directory) → parent is workspace root.
4. If parent is non-git → scan parent's immediate children for `.git` entries. If siblings found → parent is workspace root (non-git workspace).
5. [Reserved / future] Potential override via `.ww.toml` (for example, a `workspace` flag) is out of scope until the configuration spec defines such a field. Do not implement config-based workspace overrides yet.
6. If steps 2-4 find no parent but step 0 found a parent candidate → current directory becomes workspace root.
7. None of the above → single-repo mode (Phase 1 compatible).

### Child Repository Definition

All directories that contain a `.git` entry (file or directory) found at:
- workspace root's immediate children
- current directory's immediate children

Identified by absolute path. If the parent candidate from step 0 becomes workspace root (step 6), its children are already detected — skip redundant scanning.

### Edge Cases

- **Current directory is non-git with git children**: operates as non-git workspace. Parent's own worktrees are not applicable, but child repos' worktrees are managed. This relaxes the current CLI prerequisite that `ww` must be run inside a git repository/worktree and will be implemented in Phase 2 alongside an update to `docs/specs/cli-commands.md`.
- **Child repos are never workspace roots**: detection never treats child repos as workspace roots, which prevents unintended recursive workspace nesting. (See FR-22 for any future changes to this invariant.)
- **Workspace root is a git repo**: included in the child repo list alongside its children.

### Config

- `.ww.toml` is found via upward search (same as Phase 1).
- Per-child-repo `.ww.toml` override is out of scope (FR-21).

## Worktree Path Layout

The default `worktree_dir` changes based on mode:

| Mode | `worktree_dir` default | Layout |
|------|----------------------|--------|
| workspace | `".worktrees"` | `<workspace_root>/.worktrees/<repo>@<branch>` |
| single-repo | `""` | `<repo-parent>/<repo>@<branch>` (sibling) |

In workspace mode, all worktrees from all repos are collected under `<workspace_root>/.worktrees/`. This keeps the workspace root clean and worktrees centrally managed.

An explicit `worktree_dir` in `.ww.toml` overrides the default in both modes.

Examples:

**Workspace mode (default):**
```
workspace/
├── .worktrees/
│   ├── ww@feat-x/
│   └── ai-arena@feat-x/
├── .ww.toml
├── ww/
└── ai-arena/
```

**Single-repo mode (default):**
```
projects/
├── ww/
├── ww@feat-x/
└── ww@fix-bug/
```

## ww list Changes

### STATUS Column

Each non-main worktree gets a status:

| Status | Condition | `ww clean` target |
|--------|-----------|-------------------|
| `active` | Neither merged nor stale | No |
| `merged` | Branch is in `git branch --merged <base>` | Yes |
| `stale` | Remote tracking configured + remote branch gone + not merged | Yes |

- Main worktrees always show `active`.
- `merged` takes precedence if both merged and stale conditions are met.
- Base branch: `default_base` from config, or `origin/HEAD` auto-detect.
- Remote tracking detection: `git config branch.<name>.remote` is set. Branches that were never pushed (no tracking) are not stale — they are `active`.

### REPO Column

- Workspace mode: REPO column shows the repository directory name.
- Single-repo mode: REPO column is omitted (Phase 1 compatible).

### New Flag: --cleanable

Filters output to show only `merged` and `stale` worktrees. Works with `--json`.

### JSON (NDJSON) Output

Adds `"repo"` and `"status"` fields. Output is newline-delimited JSON (NDJSON), one object per line:

```jsonl
{"repo":"ww","path":"/path/to/ww@feat-x","branch":"feat/x","head":"def5678","status":"active"}
{"repo":"ww","path":"/path/to/ww@feat-done","branch":"feat/done","head":"789abcd","status":"merged"}
```

## ww clean (New Command)

Removes all `merged` and `stale` worktrees across the workspace.

| Flag | Behavior |
|------|----------|
| (none) | Safe delete: `git worktree remove` + `git branch -d` |
| `--dry-run` | Show what would be deleted, do not execute |
| `--force` | Force delete: `git worktree remove --force` + `git branch -D` |
| `--json` | Output results as JSON |

No confirmation prompt. Running `ww clean` is the user's explicit intent to delete. Use `ww list --cleanable` or `ww clean --dry-run` to preview.

## ww create / ww remove Changes

### New Flag: --repo

Targets a specific repo in the workspace instead of the current directory's repo.

```
ww create feat/x --repo ai-arena
ww remove feat/x --repo ai-arena
```

- Value must match a repo name from the workspace's child repo list.
- Omitted → current directory's repo (Phase 1 compatible).
- Error if `--repo` is used outside a workspace or repo name not found.

## Test Strategy

Phase 2 introduces workspace structures (parent + multiple child repos + worktree directories) that are more complex than Phase 1's single-repo tests.

**Integration tests should run in Docker** to isolate from host environment (git global config, filesystem layout). The container needs only `git` and the `ww` binary.

**Unit tests** (e.g., `git/` package) can continue using `t.TempDir()` — sufficient for isolated git operations.

Exec-plan should include a sub-task for setting up a Dockerfile/docker-compose for integration tests before implementing workspace features.

## Out of Scope (Future FRs)

Recorded in project-plan.md:

- **FR-19**: Multi-repo batch worktree operations (`--repos repo1,repo2,...`)
- **FR-20**: `ww cd` — shell navigation between worktrees
- **FR-21**: Child repo `.ww.toml` override
- **FR-22**: Recursive workspace detection (child `workspace = true`)
- **FR-23**: Time-based stale detection (`--stale-days`)
