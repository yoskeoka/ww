# Global project matching uses ordered path rules anchored on the main worktree root

## Status

Accepted on 2026-06-02.

## Context

Phase 5-02 extends global config so one user-owned `config.toml` can apply different settings to different repositories or workspace members. The open design questions were how project blocks should identify their targets, what path should be matched when the command runs from a secondary worktree, and whether overlapping matches should be resolved by ordering or by an implicit specificity rule.

## Decision

- Global config may define ordered `[[projects]]` blocks.
- Each project block must define exactly one target selector: `root` for exact absolute-path matches or `root_prefix` for path-prefix matches.
- Matching always evaluates against the repository main worktree root, not the current secondary worktree path.
- Project blocks are evaluated in declared order, and the first matching block wins.
- If no project block matches, `ww` uses only the base global config plus any repo-local overrides.
- Invalid project-target blocks fail config loading instead of being ignored.

## Consequences

- **Positive**: Matching stays deterministic and easy to audit because there is no hidden specificity scoring or multi-match merge behavior.
- **Positive**: Commands behave the same from the main checkout and from secondary worktrees because both use the same resolved anchor.
- **Positive**: Users can share one rule across related repositories with `root_prefix` while still overriding a specific repository earlier in the list with `root`.
- **Negative**: Absolute-path selectors are machine-local intent, so cross-machine portability still requires the user to keep directory conventions aligned.
- **Negative**: More expressive matching (for example, stacked rules or non-path attributes) remains future work and must be designed explicitly if needed.
