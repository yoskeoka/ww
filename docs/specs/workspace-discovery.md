# Workspace Discovery Specification

## Overview

`ww` detects whether the current directory is part of a workspace that contains multiple git repositories. Detection is convention-based and does not require a dedicated workspace config field.

## Public Types

### `workspace.Repo`

| Field | Type | Description |
|------|------|-------------|
| `Name` | string | Repository directory name |
| `Path` | string | Absolute path to the repository root |

### `workspace.Workspace`

| Field | Type | Description |
|------|------|-------------|
| `Root` | string | Absolute path to the detected workspace root |
| `Repos` | `[]Repo` | Repositories detected in the workspace |
| `Mode` | workspace mode enum | `single-repo` or `workspace` |

## Detection Algorithm

Detection is anchored on the current repository's main working tree root and uses immediate-child scanning only.

When the start directory is not inside git:

1. Scan the current directory's immediate children for real git repositories.
2. If at least one child repository is found, treat the current directory as the workspace root.
3. Otherwise, return `not a git repository`.

When the start directory is inside git:

1. Resolve the current repository's main working tree root.
2. Build a bounded candidate window consisting of:
   - the current directory
   - the parent of the main repo root
   - the grandparent of the main repo root
3. Test candidates in that order and pick the first qualifying workspace root.
4. A candidate qualifies only when:
   - it contains the current main repo root
   - it exposes at least two immediate child real git repositories
5. If no candidate qualifies, treat the current repository as a standalone single-repo workspace.

## Edge Cases

- `.git` files and `.git` directories both count as repository markers.
- `.git` files that point into another repository's `.git/worktrees/` directory are ignored for workspace discovery because managed worktree checkouts are not real workspace members.
- Only immediate children are scanned, and only within the bounded candidate window. Detection does not recurse through arbitrary ancestors or nested workspace structures.
- The current directory is tested first. If it already qualifies as a workspace root, it is selected immediately.
- If both the parent and grandparent of the main repo root qualify, the nearest containing candidate wins.
- A workspace root may itself be a git repository; if so, it is included in `Repos`.
- A non-git workspace root is valid when it is the current directory and contains one or more git child repositories, or when it contains two or more git child repositories while being selected as a containing candidate for the current repo.

## Worktree Path Layout

`worktree_dir` defaults by mode:

| Mode | Default | Resulting layout |
|------|---------|------------------|
| `workspace` | `.worktrees` | `<workspace_root>/.worktrees/<repo>@<branch>` |
| `single-repo` | `""` | `<repo-parent>/<repo>@<branch>` |

Behavior:
- Explicit `worktree_dir` overrides the mode default in both modes.
- Relative `worktree_dir` values are resolved against the workspace root in workspace mode.
- Relative `worktree_dir` values are resolved against the repository parent in single-repo mode.
- Absolute `worktree_dir` values are used as-is.

## CLI Prerequisite

`ww` can start from a non-git workspace root, but this plan does not add repo selection for that root yet. Until `--repo` lands, commands that require a current repo return `repo selection is not supported from a non-git workspace root`.
