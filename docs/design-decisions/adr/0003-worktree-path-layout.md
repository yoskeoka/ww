# Worktree path layout: centralized `.worktrees/` in workspace mode

## Status

Accepted on 2026-03-18.

## Context

In single-repo mode, worktrees are created as siblings: `<repo-parent>/<repo>@<branch>`. In workspace mode with N child repos, sibling layout would scatter worktrees across child directories, making them hard to find and clean up.

## Decision

Workspace mode centralizes all worktrees under `<workspace_root>/.worktrees/<repo>@<branch>`. Single-repo mode keeps the sibling layout. An explicit `worktree_dir` in `.ww.toml` overrides either default.

## Consequences

- **Positive**: Single location for all worktrees across all repos. Easier cleanup, listing, and mental model.
- **Negative**: Worktrees are not adjacent to their source repo. Accepted — `ww list` provides the lookup and most editors/agents use absolute paths.
