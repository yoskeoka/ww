# Sandbox-constrained mode overrides upward discovery and single-repo sibling layout

## Status

Accepted on 2026-04-23.

## Context

Normal workspace detection intentionally inspects a bounded parent/grandparent window so `ww` can infer practical meta-repo layouts without configuration. Normal single-repo mode also places worktrees beside the repository. Both choices are appropriate for unrestricted local shells, but they are poor defaults in filesystem-sandboxed AI agent environments where parent directories may be unreadable or writes beside the repo may be blocked.

## Decision

Add sandbox-constrained mode as `--sandbox` and `sandbox = true`. In sandbox mode, `ww` scans only the current directory's immediate children before falling back to the current repository. It does not inspect parent or grandparent directories to discover containing workspaces, and it does not scan parent siblings. Config lookup stops at the active sandbox boundary. Single-repo default worktrees move from sibling layout to `<repo_root>/.worktrees/<repo>@<branch>`, while current-directory workspace roots keep the workspace layout at `<workspace_root>/.worktrees/<repo>@<branch>`.

Explicit user intent remains honored: absolute `worktree_dir` values and existing git worktree relationships are not rejected solely for pointing outside the sandbox-friendly default area. Relative `worktree_dir` values that escape their anchor with `..` remain rejected.

## Consequences

- **Positive**: `ww` can operate from the repository or bounded workspace directory without depending on parent directory access.
- **Positive**: The feature name describes the complete behavior instead of only the upward-search mechanism.
- **Negative**: From inside a child repository, sandbox mode cannot discover parent workspace siblings; users must start at the workspace root to use `--repo`.
- **Negative**: Config-level `sandbox = true` can only help after a config file is reachable. In severely restricted environments, users should pass `--sandbox` explicitly.
