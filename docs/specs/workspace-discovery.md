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

Detection uses a parent-scan strategy with immediate-child scanning only:

1. Scan the current directory's immediate children for git repository markers.
2. Determine the current repository root via the main working tree path when inside git.
3. Inspect the parent directory of the current repository root.
4. If the parent has a `.git` entry and the grandparent does not expose multiple git child repos, treat the parent as the workspace root.
5. If the parent is non-git and has two or more git child repos, treat the parent as the workspace root.
6. If no parent workspace root was found and the current directory has git children, treat the current directory as the workspace root only when its parent is non-git.
7. Otherwise, treat the current repository as a standalone single-repo workspace.

## Edge Cases

- `.git` files and `.git` directories both count as repository markers.
- `.git` files that point into another repository's `.git/worktrees/` directory are ignored for workspace discovery because they are worktree checkouts, not workspace members.
- Only immediate children are scanned; detection does not recurse through nested workspace structures.
- Child repositories are never treated as workspace roots.
- A workspace root may itself be a git repository; if so, it is included in `Repos`.
- A non-git workspace root is valid when it contains two or more git child repositories, or when it is the current directory and contains git children.

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
