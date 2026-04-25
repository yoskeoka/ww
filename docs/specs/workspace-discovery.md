# Workspace Discovery Specification

## Overview

`ww` detects whether the current directory is part of a workspace that contains multiple git repositories. Detection is convention-based and does not require a dedicated workspace config field.

For workspace membership, a "real git repository" means an immediate child directory that:

- is not itself a symlink
- resolves its own git top-level to that child directory
- is not a linked worktree checkout

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
   - the main repo root
   - the parent of the main repo root
   - the grandparent of the main repo root
3. Test candidates in that order and pick the first qualifying workspace root.
4. A candidate qualifies only when:
   - it contains the current main repo root
   - it exposes at least two immediate child real git repositories
   - it is the current main repo root itself, or one of those immediate child repositories contains the current main repo root
5. If no candidate qualifies, treat the current repository as a standalone single-repo workspace.

## Sandbox Mode

Sandbox mode is enabled by the global `--sandbox` flag or by `sandbox = true` in `.ww.toml` when that config can be loaded. It constrains default discovery to the current sandbox boundary instead of trying to infer a containing workspace from parent directories.

When sandbox mode is enabled:

1. Scan only the current directory's immediate children for real git repositories.
2. If child repositories are found, treat the current directory as a workspace root, even if the current directory is not itself a git repository.
3. Otherwise, if the current directory is inside git, resolve the repository's main working tree root and operate in `single-repo` mode for that repository.
4. Otherwise, return `not a git repository`.

Sandbox mode does not inspect parent or grandparent directories while detecting a containing workspace. It also does not scan sibling repositories from a parent workspace candidate. From inside a child repository, sandbox mode therefore cannot discover parent workspace siblings, and commands using `--repo <name>` fail unless the current directory itself is a detected workspace root.

## Edge Cases

- `.git` presence alone is not sufficient for workspace membership. Candidate children are validated with git top-level and git-dir/common-dir resolution.
- Immediate child scanning may use cheap `DirEntry` metadata to prefilter obvious non-repository entries before git validation runs.
- Immediate child symlink entries are ignored during workspace-member discovery even when they survive that cheap prefilter. `ww` does not follow child symlinks by default.
- Linked worktree checkouts are ignored for workspace discovery even when their top-level path matches the child directory, because managed worktree checkouts are not real workspace members.
- Helper directories containing stray or partial `.git` contents do not count as repositories unless git resolves them as standalone repo roots.
- Only immediate children are scanned, and only within the bounded candidate window. Detection does not recurse through arbitrary ancestors or nested workspace structures.
- The current directory is tested first. If it already qualifies as a workspace root, it is selected immediately.
- If both the parent and grandparent of the main repo root qualify, the nearest containing candidate wins.
- Generic ancestors with unrelated repo children do not qualify. A containing candidate must be tied to the current repo through one of its immediate child repositories, unless the candidate is the current main repo root itself.
- A workspace root may itself be a git repository; if so, it is included in `Repos`.
- A non-git workspace root is valid when it is the current directory and contains one or more git child repositories, or when it contains two or more git child repositories while being selected as a containing candidate for the current repo.

## Worktree Path Layout

`worktree_dir` defaults by mode:

| Mode | Default | Resulting layout |
|------|---------|------------------|
| `workspace` | `.worktrees` | `<workspace_root>/.worktrees/<repo>@<branch>` |
| `single-repo` | `""` | `<repo-parent>/<repo>@<branch>` |
| `single-repo` with sandbox mode | `.worktrees` | `<repo_root>/.worktrees/<repo>@<branch>` |

Behavior:
- Explicit `worktree_dir` overrides the mode default in both modes.
- Relative `worktree_dir` values are resolved against the workspace root in workspace mode.
- Relative `worktree_dir` values are resolved against the repository parent in single-repo mode.
- In sandbox single-repo mode, relative `worktree_dir` values are resolved against the repository root.
- Relative `worktree_dir` values that escape their anchor with `..` are rejected, including in sandbox mode.
- Absolute `worktree_dir` values are used as-is.
- Absolute `worktree_dir` values are honored even if they point outside the sandbox-friendly default area. Any real sandbox denial is surfaced as the underlying filesystem or git error.

## CLI Prerequisite

`ww` can start from a non-git workspace root, but this plan does not add repo selection for that root yet. Until `--repo` lands, commands that require a current repo return `repo selection is not supported from a non-git workspace root`.
